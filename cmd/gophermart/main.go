package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gophermart/internal/app"
	mart "gophermart/internal/handler/gophermart"

	"golang.org/x/sync/errgroup"
)

func main() {
	shutdownCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	srvc, err := app.Setup()
	if err != nil {
		srvc.Logger.Fatal(err)
	}

	gophermartHandler := mart.NewGophermart(srvc)

	httpServer := &http.Server{
		Addr: srvc.Config.RunAddr,
		BaseContext: func(l net.Listener) context.Context {
			return shutdownCtx
		},
		Handler: gophermartHandler.Router(),
	}

	g, gCtx := errgroup.WithContext(shutdownCtx)
	g.Go(func() error {
		srvc.Logger.Infof("Server starting on %s", srvc.Config.RunAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		srvc.Logger.Info("Shutting down server...")

		timeoutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := httpServer.Shutdown(timeoutCtx); err != nil {
			return fmt.Errorf("server shutdown error: %w", err)
		}

		srvc.Logger.Info("Server stopped")
		return nil
	})

	// Ждем завершения всех горутин
	if err := g.Wait(); err != nil {
		srvc.Logger.Errorf("Application error: %v", err)
		os.Exit(1)
	}

	srvc.Logger.Info("Application stopped")
}
