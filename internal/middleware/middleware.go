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
        // ВРЕМЕННО: пропускай все запросы
        userID := "test-user-id"
        ctx := context.WithValue(r.Context(), internal.UserIDKey, userID)
        next.ServeHTTP(rw, r.WithContext(ctx))
    })
}
