package middleware

import (
	"context"
	"gophermart/internal"
	"gophermart/internal/service"
	"net/http"
)

type Middleware struct {
	srvc *service.Service
}

func NewMiddleware(srvc *service.Service) *Middleware {
	return &Middleware{
		srvc: srvc,
	}
}

func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_id")

		if err == nil && cookie.Value != "" {
			var userID string
			userID, err = m.srvc.ValidateAuthToken(cookie.Value, m.srvc.Config.JWTSecret)
			if err != nil {
				http.Error(rw, "Invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), internal.UserIDKey, userID)
			next.ServeHTTP(rw, r.WithContext(ctx))
			return
		}
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
	})
}
