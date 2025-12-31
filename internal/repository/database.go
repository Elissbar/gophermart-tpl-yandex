package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"gophermart/internal"
	"gophermart/internal/model"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"go.uber.org/zap"

	"github.com/jackc/pgerrcode"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
)

type DBStorage struct {
	DB              *sql.DB
	logger          *zap.SugaredLogger
	ordersRepo      *Repository[model.Order]
	withdrawalsRepo *Repository[model.Withdraw]
	balancesRepo    *Repository[model.Balance]
	loginRepo       *Repository[model.User]
}

func NewDatabaseStorage(connectionData string, logger *zap.SugaredLogger) (*DBStorage, error) {
	db, err := sql.Open("postgres", connectionData)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	// проверяем соединение с БД
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.Info("DB connection success")
	// применяем миграции
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}
	logger.Info("Migrations applied successfully")

	return &DBStorage{
		DB:              db,
		logger:          logger,
		ordersRepo:      &Repository[model.Order]{DB: db, Table: "orders"},
		withdrawalsRepo: &Repository[model.Withdraw]{DB: db, Table: "withdrawals"},
		balancesRepo:    &Repository[model.Balance]{DB: db, Table: "balances"},
		loginRepo:       &Repository[model.User]{DB: db, Table: "users"},
	}, nil
}

func Migrate(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err = m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

func (db *DBStorage) RegisterUser(ctx context.Context, user model.User) (string, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("error begin transaction: %w", err)
	}
	defer tx.Rollback()

	var userID string
	err = tx.QueryRowContext(ctx, "INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id", user.Login, user.Password).Scan(&userID)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return "", internal.ErrLoginExists
		}
		return "", fmt.Errorf("error register user: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO balances (user_id) VALUES ($1)",
		userID,
	)
	if err != nil {
		return "", fmt.Errorf("error create balance in DB: %w", err)
	}

	return userID, tx.Commit()
}

func (db *DBStorage) UploadNumber(ctx context.Context, userID string, number string) error {
	var existingUserID string
	err := db.DB.QueryRowContext(ctx, "SELECT user_id FROM orders WHERE number=$1", number).Scan(&existingUserID)
	if err == nil {
		if existingUserID == userID {
			return internal.ErrOrderAlreadyUploadedByUser
		}
		return internal.ErrOrderUploadConflict
	}

	_, err = db.DB.ExecContext(ctx, "INSERT INTO orders (user_id, number) VALUES ($1, $2)", userID, number)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return internal.ErrOrderUploadConflict
		}
		return fmt.Errorf("error upload number to DB: %w", err)
	}
	return nil
}

func (db *DBStorage) PostWithdraw(ctx context.Context, userID string, withdraw model.Withdraw) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		"INSERT INTO withdrawals (user_id, order_number, sum) VALUES ($1, $2, $3)",
		userID, withdraw.Order, withdraw.Sum,
	)
	if err != nil {
		return fmt.Errorf("error insert withdraw to DB: %w", err)
	}

	_, err = tx.ExecContext(
		ctx,
		"UPDATE balances SET current = current - $1, withdrawn = withdrawn + $1 WHERE user_id = $2",
		withdraw.Sum, userID,
	)
	if err != nil {
		return fmt.Errorf("error update balance in DB: %w", err)
	}

	return tx.Commit()
}

func (db *DBStorage) UpdateOrderStatus(ctx context.Context, userID string, result model.Order) error {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx,
		"UPDATE orders SET status = $1, accrual = $2 WHERE number = $3",
		result.Status, result.Accrual, result.OrderNum,
	)
	if err != nil {
		return fmt.Errorf("error update order status in DB: %w", err)
	}

	// Обновляем баланс пользователя
	if result.Status == "PROCESSED" {
		_, err = tx.ExecContext(
			ctx,
			"UPDATE balances SET current = current + $1 WHERE user_id = $2",
			result.Accrual, userID,
		)
		if err != nil {
			return fmt.Errorf("error update balance in DB: %w", err)
		}
	}

	return tx.Commit()
}

func (db *DBStorage) GetUser(ctx context.Context, login string) (*model.User, error) {
	return db.loginRepo.GetOneRow(
		ctx, 
		scanUser,
		"SELECT id, login, password_hash FROM users WHERE login=$1",
		[]any{login},
	)
}

func (db *DBStorage) GetUserOrders(ctx context.Context, userID string) (*[]model.Order, error) {
	return db.ordersRepo.GetAllRows(
		ctx,
		scanOrders,
		"SELECT user_id, number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC",
		[]any{userID},
	)
}

func (db *DBStorage) GetBalance(ctx context.Context, userID string) (*model.Balance, error) {
	return db.balancesRepo.GetOneRow(
		ctx,
		scanBalance,
		"SELECT current, withdrawn FROM balances WHERE user_id=$1",
		[]any{userID},
	)
}

func (db *DBStorage) GetWithdrawals(ctx context.Context, userID string) (*[]model.Withdraw, error) {
	return db.withdrawalsRepo.GetAllRows(
		ctx,
		scanWithdrawals,
		"SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC",
		[]any{userID},
	)
}

func (db *DBStorage) GetAllOrders(ctx context.Context) (*[]model.Order, error) {
	return db.ordersRepo.GetAllRows(
		ctx,
		scanOrders,
		"SELECT user_id, number, status, accrual, uploaded_at FROM orders WHERE status in ('NEW', 'PROCESSING')",
		[]any{},
	)
}
