package middleware

import (
	"compress/gzip"
	"context"
	"gophermart/internal"
	"gophermart/internal/service"
	"net/http"
	"strings"
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

func (m *Middleware) Logging(handler http.Handler) http.Handler {
	logFn := func(rw http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		uri := r.RequestURI
		metnod := r.Method

		handler.ServeHTTP(rw, r)

		duration := time.Since(startTime)

		m.srvc.Logger.Infow("Request/Response data: ",
			"uri", uri,
			"method", metnod,
			"duration", int(duration),
		)
	}

	return http.HandlerFunc(logFn)
}

func (m *Middleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("user_id")

		if err == nil && cookie.Value != "" {
			var userID string
			userID, err = m.srvc.ValidateAuthToken(cookie.Value, m.srvc.Config.JWTSecret)
			if err != nil {
				http.Error(rw, "Invalid token: "+err.Error(), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), internal.UserIDKey, userID)
			next.ServeHTTP(rw, r.WithContext(ctx))
			return
		}
		http.Error(rw, "Unauthorized", http.StatusUnauthorized)
	})
}

type gzipResponseWriter struct {
	Writer *gzip.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func (m *Middleware) Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			gw := &gzipResponseWriter{
				Writer:         gzip.NewWriter(rw),
				ResponseWriter: rw,
			}
			defer gw.Writer.Close()

			rw.Header().Set("Content-Encoding", "gzip")
			next.ServeHTTP(gw, r)
			return
		}

		next.ServeHTTP(rw, r)
	})
}

func (m *Middleware) Ungzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			defer gz.Close()

			r.Body = gz
		}

		next.ServeHTTP(w, r)
	})
}
