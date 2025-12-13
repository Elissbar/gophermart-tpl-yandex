package main

import (
	"log"
	"net/http"

	"gophermart/internal/app"
	mart "gophermart/internal/handler/gophermart"
	"gophermart/internal/service"
)

func Run(srvc *service.Service) error {
	martHandler := mart.NewGophermart(srvc)
	return http.ListenAndServe(srvc.Config.RunAddr, martHandler.Router())
}

func main() {
	srvc, err := app.Setup()
	if err != nil {
		log.Fatal(err)
	}

	if err := Run(srvc); err != nil {
		log.Fatal(err)
	}
}
