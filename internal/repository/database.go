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

	"github.com/jackc/pgerrcode"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
)

type DBStorage struct {
	DB *sql.DB
}

func NewDatabaseStorage(connectionData string) (*DBStorage, error) {
	db, err := sql.Open("postgres", connectionData)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	// проверяем соединение с БД
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	// применяем миграции
	if err := Migrate(db); err != nil {
		return nil, fmt.Errorf("failed to migrate: %w", err)
	}

	return &DBStorage{DB: db}, nil
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

func (db *DBStorage) GetUser(ctx context.Context, login string) (*model.User, error) {
	row := db.DB.QueryRowContext(ctx, "SELECT id, login, password_hash FROM users WHERE login=$1", login)
	var existedUser model.User

	if err := row.Scan(&existedUser.ID, &existedUser.Login, &existedUser.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internal.ErrUserNotFound
		}
		return nil, fmt.Errorf("error get user from DB: %w", err)
	}
	return &existedUser, nil
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

func (db *DBStorage) GetUserOrders(ctx context.Context, userID string) ([]model.Order, error) {
	fmt.Println("DB GetOrders")

	rows, err := db.DB.QueryContext(
		ctx,
		"SELECT number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("error get orders from DB: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order

		err = rows.Scan(
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scan order from DB: %w", err)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (db *DBStorage) GetBalance(ctx context.Context, userID string) (model.Balance, error) {
	row := db.DB.QueryRowContext(ctx, "SELECT current, withdrawn FROM balances WHERE user_id=$1", userID)

	var balance model.Balance
	if err := row.Scan(&balance.Current, &balance.Withdrawn); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Balance{}, internal.ErrNoRows
		}
		return model.Balance{}, fmt.Errorf("error get balance from DB: %w", err)
	}

	return balance, nil
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

func (db *DBStorage) GetWithdrawals(ctx context.Context, userID string) ([]model.Withdraw, error) {
	rows, err := db.DB.QueryContext(
		ctx,
		"SELECT order_number, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC",
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("error get withdrawals from DB: %w", err)
	}
	defer rows.Close()

	var withdrawals []model.Withdraw
	for rows.Next() {
		var withdrawn model.Withdraw

		err = rows.Scan(
			&withdrawn.Order,
			&withdrawn.Sum,
			&withdrawn.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scan withdrawn from DB: %w", err)
		}
		withdrawals = append(withdrawals, withdrawn)
	}

	return withdrawals, nil
}

func (db *DBStorage) GetAllOrders(ctx context.Context, query string) ([]model.Order, error) {
	rows, err := db.DB.QueryContext(
		ctx,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("error get all orders from DB: %w", err)
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order

		err = rows.Scan(
			&order.ID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scan order from DB: %w", err)
		}
		orders = append(orders, order)
	}
	
	return orders, nil
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