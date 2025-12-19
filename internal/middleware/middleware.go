package middleware

import (
	"context"
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
		// Таймаут на всю авторизацию
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		cookie, err := r.Cookie("user_id")
		if err != nil {
			http.Error(rw, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Канал для асинхронной проверки
		userIDChan := make(chan string, 1)
		errChan := make(chan error, 1)

		go func() {
			userID, err := m.srvc.ValidateAuthToken(cookie.Value, m.srvc.Config.JWTSecret)
			if err != nil {
				errChan <- err
				return
			}
			userIDChan <- userID
		}()

		select {
		case <-ctx.Done():
			http.Error(rw, "Authorization timeout", http.StatusRequestTimeout)
			return
		case <-errChan:
			http.Error(rw, "Invalid token", http.StatusUnauthorized)
			return
		case userID := <-userIDChan:
			ctx2 := context.WithValue(r.Context(), internal.UserIDKey, userID)
			next.ServeHTTP(rw, r.WithContext(ctx2))
		}
	})
}
