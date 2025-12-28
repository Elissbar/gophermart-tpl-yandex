package main

import (
	"net/http"

	"gophermart/internal/app"
	mart "gophermart/internal/handler/gophermart"
)

func main() {
	srvc, err := app.Setup()
	if err != nil {
		srvc.Logger.Fatal(err)
	}

	gophermartHandler := mart.NewGophermart(srvc)
	if err := http.ListenAndServe(srvc.Config.RunAddr, gophermartHandler.Router()); err != nil {
		srvc.Logger.Fatal(err)
	}
}
