package main

import (
	"log"
	"net/http"

	"gophermart/internal/app"
	mart "gophermart/internal/handler/gophermart"
)


func main() {
	srvc, err := app.Setup()
	if err != nil {
		log.Fatal(err)
	}

	martHandler := mart.NewGophermart(srvc)
	if err := http.ListenAndServe(srvc.Config.RunAddr, martHandler.Router()); err != nil {
		log.Fatal(err)
	}
}
