package repository

// import (
// 	"context"
// 	"gophermart/internal/model"
// )

// type Storage interface {
// 	RegisterUser(ctx context.Context, user model.User) (string, error)
// 	GetUser(ctx context.Context, login string) (*model.User, error)
// 	UploadNumber(ctx context.Context, userID string, number string) error
// 	GetUserOrders(ctx context.Context, userID string) ([]model.Order, error)
// 	GetAllOrders(ctx context.Context, query string) ([]model.Order, error)
// 	GetBalance(ctx context.Context, userID string) (model.Balance, error)
// 	PostWithdraw(ctx context.Context, userID string, withdraw model.Withdraw) error
// 	GetWithdrawals(ctx context.Context, userID string) ([]model.Withdraw, error)
// 	UpdateOrderStatus(ctx context.Context, userID string, result model.Order) error
// }
