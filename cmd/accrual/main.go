package accrual

import (
	"gophermart/internal/app"
	accrual "gophermart/internal/handler/accrual"
	"gophermart/internal/service"
	"log"
	"net/http"
)

func Run(srvc *service.Service) error {
	accrualHandler := accrual.NewAccrual(srvc)
	return http.ListenAndServe(srvc.Config.AccrualAddr, accrualHandler.Router())
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
