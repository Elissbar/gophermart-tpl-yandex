package service

import (
	"fmt"
	"gophermart/internal/config"
	"gophermart/internal/repository"
	"net/http"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	Storage repository.Storage
	Config  *config.Config
	Logger  *zap.SugaredLogger
}

func (s *Service) CalcPassHash(pass string) (string, error) {
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash pass error: %w", err)
	}
	return string(hashedPass), nil
}

func (s *Service) CheckHashedPass(hashedPass, pass string) bool {
	errCompare := bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(pass))
	return errCompare == nil // Если nil, пароли совпадают
}

func (s *Service) GenerateAuthToken(userID, jwtSecret string) (*http.Cookie, error) {
	claim := struct {
		UserID string
		jwt.StandardClaims
	}{UserID: userID}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return &http.Cookie{}, fmt.Errorf("error create uuid: %w", err)
	}

	return &http.Cookie{Name: "user_id", Value: tokenString, HttpOnly: true}, nil
}
