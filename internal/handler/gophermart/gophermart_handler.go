package gophermart_handler

import (
	"encoding/json"
	"errors"
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

	middleware := middleware.NewMiddleware(h.service)

	r.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Post("/orders", h.UploadOrderNumber)
		r.Get("/orders", h.GetOrders)
		// r.Get("/api/user/balance", h.GetBalance)
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
	rw.Header().Set("Content-Type", "application/json")

	orders, err := h.service.Storage.GetOrders(r.Context())
	if err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		rw.WriteHeader(http.StatusNoContent)
		return
	}

	enc := json.NewEncoder(rw)
	if err := enc.Encode(orders); err != nil {
		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// func (h *GophermartHandler) GetBalance(rw http.ResponseWriter, r *http.Request) {
// 	rw.Header().Set("Content-Type", "application/json")

// 	userID := r.Context().Value(internal.UserIDKey).(string)
// 	balance, err := h.service.Storage.GetBalance(r.Context(), userID)
// 	if err != nil {
// 		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	enc := json.NewEncoder(rw)
// 	if err := enc.Encode(balance); err != nil {
// 		http.Error(rw, "Error: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// }
