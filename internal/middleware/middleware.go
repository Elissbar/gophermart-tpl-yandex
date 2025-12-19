package middleware

import (
	"context"
	"fmt"
	"gophermart/internal"
	"gophermart/internal/service"
	"net/http"
	"time"
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
		// ВРЕМЕННО: логируем все запросы
		fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), r.URL.Path)

		cookie, err := r.Cookie("user_id")
		if err != nil {
			fmt.Printf("  No cookie: %v\n", err)
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
			return
		}

		fmt.Printf("  Cookie length: %d\n", len(cookie.Value))

		userID, err := m.srvc.ValidateAuthToken(cookie.Value, m.srvc.Config.JWTSecret)
		if err != nil {
			fmt.Printf("  Token error: %v\n", err)
			http.Error(rw, "Invalid token", http.StatusUnauthorized)
			return
		}

		fmt.Printf("  UserID: %s\n", userID)

		ctx := context.WithValue(r.Context(), internal.UserIDKey, userID)
		next.ServeHTTP(rw, r.WithContext(ctx))
	})
}
