package gophermart_handler

import (
	"encoding/json"
	"errors"
	"gophermart/internal"
	"gophermart/internal/model"
	"gophermart/internal/service"
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
