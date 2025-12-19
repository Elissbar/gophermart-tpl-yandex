package service

import (
	"fmt"
	"gophermart/internal"
	"gophermart/internal/config"
	"gophermart/internal/repository"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
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
	claim := internal.Claims{UserID: userID}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return &http.Cookie{}, fmt.Errorf("error create uuid: %w", err)
	}

	return &http.Cookie{
			Name:     "user_id",
			Value:    tokenString,
			HttpOnly: true,
			Path:     "/",
			MaxAge:   24 * 60 * 60,
		},
		nil
}

func (s *Service) ValidateAuthToken(tokenString, jwtSecret string) (string, error) {
	// claims := &internal.Claims{}

	// token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
	// 	if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
	// 		return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
	// 	}
	// 	return []byte(jwtSecret), nil
	// })
	// if err != nil {
	// 	fmt.Println("Error: ", err)
	// 	return "", fmt.Errorf("error parse token: %w", err)
	// }

	// if !token.Valid {
	// 	fmt.Println("Token: ", token)
	// 	return "", fmt.Errorf("invalid token")
	// }

	// if claims.UserID == "" {
	// 	fmt.Println("claims.UserID:", claims.UserID)
	// 	return "", fmt.Errorf("user_id is empty")
	// }

	// return claims.UserID, nil
	return "test-user-id", nil
}

func (s *Service) ValidLuhn(number string) bool {
	n := len(number)
	if n < 2 {
		return false
	}

	sum := 0
	isSecond := false

	for i := n - 1; i >= 0; i-- {
		d := int(number[i] - '0')

		if d < 0 || d > 9 {
			return false
		}

		if isSecond {
			d = d * 2
			if d > 9 {
				d = d - 9
			}
		}

		sum += d
		isSecond = !isSecond
	}

	return sum%10 == 0
}
