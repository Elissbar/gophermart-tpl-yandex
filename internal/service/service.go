package service

import (
	"context"
	"fmt"
	"gophermart/internal"
	"gophermart/internal/config"
	"gophermart/internal/model"
	"gophermart/internal/repository"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"github.com/go-resty/resty/v2"
)

type Service struct {
	Storage *repository.DBStorage // repository.Storage
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
	claims := &internal.Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		fmt.Println("Error: ", err)
		return "", fmt.Errorf("error parse token: %w", err)
	}

	if !token.Valid {
		fmt.Println("Token: ", token)
		return "", fmt.Errorf("invalid token")
	}

	if claims.UserID == "" {
		fmt.Println("claims.UserID:", claims.UserID)
		return "", fmt.Errorf("user_id is empty")
	}

	return claims.UserID, nil
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

func (s *Service) UpdateOrderStatus() error {
	ticker := time.NewTicker(15*time.Second)
	defer ticker.Stop()

	tasks := make(chan model.Order, 100)

	// Start workers
	for i := 0; i < 5; i++ {
		go s.worker(tasks)
	}
	defer close(tasks)

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)

		query := "SELECT user_id, number, status, accrual, uploaded_at FROM orders WHERE status in ('NEW', 'PROCESSING')"
		orders, err := s.Storage.GetAllOrders(ctx, query)
		cancel()
		if err != nil {
			return err
		}

		for _, order := range orders {
			select {
			case tasks <- order:
			default:
                s.Logger.Warnf("Tasks channel full, skipping order %s", order.Number)
			}
		}
	}
	return nil
}

func (s *Service) worker(tasks <-chan model.Order) {
	for order := range tasks {
		var result model.Order 

		client := resty.New()
		resp, err := client.R().
			SetResult(&result).
			Get("http://"+s.Config.AccrualAddr+"/api/orders/"+order.Number)
		
		fmt.Println("Status Code:", resp.StatusCode())
		fmt.Println("Resp:", resp)
		fmt.Println("Result:", result)
		fmt.Println("Order Status:", result.Status)

		if err != nil {
			s.Logger.Warnf("API error for order %s: %v", order.Number, err)
			return
		}
		
		if resp.StatusCode() == 200 && order.Status != result.Status {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        	defer cancel()

			if err := s.Storage.UpdateOrderStatus(ctx, order.ID, result); err != nil {
				s.Logger.Errorf("Failed to update order %s: %v", order.Number, err)
			} else {
				s.Logger.Infof("Order %s updated to %s", order.Number, result.Status)
			}
		}
	}
}
