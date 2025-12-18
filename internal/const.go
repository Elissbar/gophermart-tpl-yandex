package internal

import "github.com/golang-jwt/jwt/v5"

type contextKey string

const UserIDKey contextKey = "user_id"

type Claims struct {
	UserID string
	jwt.RegisteredClaims
}
