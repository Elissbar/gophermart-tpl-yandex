package accrual_handler

import (
	"net/http"

	"gophermart/internal/service"

	"github.com/go-chi/chi/v5"
)

type AccrualHandler struct {
	service *service.Service
}

func NewAccrual(srvc *service.Service) *AccrualHandler {
	return &AccrualHandler{
		service: srvc,
	}
}

func (h *AccrualHandler) Router() chi.Router {
	r := chi.NewRouter()

	r.Get("/api/orders/{number}", h.GetOrderInfo)
	return r
}

func (h *AccrualHandler) GetOrderInfo(w http.ResponseWriter, r *http.Request) {}
