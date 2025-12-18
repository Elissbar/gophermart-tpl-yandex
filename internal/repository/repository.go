package repository

import (
	"context"
	"gophermart/internal/model"
)

type Storage interface {
	RegisterUser(ctx context.Context, user model.User) (string, error)
	GetUser(ctx context.Context, login string) (*model.User, error)
	UploadNumber(ctx context.Context, userID string, number string) error
	GetOrders(ctx context.Context) ([]model.Order, error)
	// GetBalance(ctx context.Context) (model.Balance, error)
}
