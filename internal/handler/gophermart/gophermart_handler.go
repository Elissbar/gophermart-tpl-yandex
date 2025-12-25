package gophermart_handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"gophermart/internal"
	"gophermart/internal/middleware"
	"gophermart/internal/model"
	"gophermart/internal/service"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type GophermartHandler struct {
	service *service.Service
}

func NewGophermart(srvc *service.Service) *GophermartHandler {
	return &GophermartHandler{
		service: srvc,
	}
}

func (h *GophermartHandler) Router() chi.Router {
	r := chi.NewRouter()

	r.Post("/api/user/register", h.RegisterUser)
	r.Post("/api/user/login", h.LoginUser)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware := middleware.NewMiddleware(h.service)

	r.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Post("/orders", h.UploadOrderNumber)
		r.Get("/orders", h.GetOrders)
		r.Get("/balance", h.GetBalance)
		r.Post("/balance/withdraw", h.PostWithdraw)
		r.Get("/withdrawals", h.GetWithdrawals)
	})

	return r
}

func (h *GophermartHandler) RegisterUser(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var user model.User
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&user); err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusBadRequest)
		return
	}

	hashedPass, err := h.service.CalcPassHash(user.Password)
	if err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	user.Password = hashedPass

	userID, err := h.service.Storage.RegisterUser(r.Context(), user)
	if err != nil {
		if errors.Is(err, internal.ErrLoginExists) {
			rw.WriteHeader(http.StatusConflict)
			return
		}
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cookie, err := h.service.GenerateAuthToken(userID, h.service.Config.JWTSecret)
	if err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(rw, cookie)
	rw.WriteHeader(http.StatusOK)
}

func (h *GophermartHandler) LoginUser(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var user model.User
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&user); err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusBadRequest)
		return
	}

	existedUser, err := h.service.Storage.GetUser(r.Context(), user.Login)
	if err != nil {
		if errors.Is(err, internal.ErrUserNotFound) {
			rw.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !h.service.CheckHashedPass(existedUser.Password, user.Password) {
		rw.WriteHeader(http.StatusUnauthorized)
		return
	}

	cookie, err := h.service.GenerateAuthToken(existedUser.ID, h.service.Config.JWTSecret)
	if err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(rw, cookie)
	rw.WriteHeader(http.StatusOK)
}

func (h *GophermartHandler) UploadOrderNumber(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID := r.Context().Value(internal.UserIDKey).(string)

	if r.Header.Get("Content-Type") != "text/plain" {
		http.Error(rw, "Content-Type is not text/plain", http.StatusBadRequest)
		return
	}

	order, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !h.service.ValidLuhn(string(order)) {
		http.Error(rw, "Invalid format of order number", http.StatusUnprocessableEntity)
		return
	}

	err = h.service.Storage.UploadNumber(r.Context(), userID, string(order))
	if err != nil {
		if errors.Is(err, internal.ErrOrderUploadConflict) {
			rw.WriteHeader(http.StatusConflict)
			return
		} else if errors.Is(err, internal.ErrOrderAlreadyUploadedByUser) {
			rw.WriteHeader(http.StatusOK)
			return
		}
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusAccepted)
}

func (h *GophermartHandler) GetOrders(rw http.ResponseWriter, r *http.Request) {
	fmt.Print("handler GetOrders")

	userID := r.Context().Value(internal.UserIDKey).(string)

	orders, err := h.service.Storage.GetUserOrders(r.Context(), userID)
	if err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(rw)
	if err := enc.Encode(orders); err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h *GophermartHandler) GetBalance(rw http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(internal.UserIDKey).(string)
	balance, err := h.service.Storage.GetBalance(r.Context(), userID)
	if err == nil || errors.Is(err, internal.ErrNoRows) {
		rw.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(rw)
		if err := enc.Encode(balance); err != nil {
			http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		return
	}
	http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
}

func (h *GophermartHandler) PostWithdraw(rw http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(internal.UserIDKey).(string)

	var withdraw model.Withdraw
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&withdraw); err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	balance, err := h.service.Storage.GetBalance(r.Context(), userID)
	if err != nil && !errors.Is(err, internal.ErrNoRows) {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if balance.Current < withdraw.Sum {
		rw.WriteHeader(http.StatusPaymentRequired)
		return
	}

	if !h.service.ValidLuhn(withdraw.Order) {
		http.Error(rw, "Invalid format of order number", http.StatusUnprocessableEntity)
		return
	}

	err = h.service.Storage.PostWithdraw(r.Context(), userID, withdraw)
	if err != nil {
		http.Error(rw, "Invalid format of order number", http.StatusUnprocessableEntity)
		return
	}
}

func (h *GophermartHandler) GetWithdrawals(rw http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(internal.UserIDKey).(string)
	withdrawals, err := h.service.Storage.GetWithdrawals(r.Context(), userID)
	if err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	enc := json.NewEncoder(rw)
	if err := enc.Encode(withdrawals); err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
