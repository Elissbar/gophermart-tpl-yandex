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
	var userID string
	err := db.DB.QueryRowContext(ctx, "INSERT INTO users (login, password_hash) VALUES ($1, $2) RETURNING id", user.Login, user.Password).Scan(&userID)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return "", internal.ErrLoginExists
		}
		return "", fmt.Errorf("error register user: %w", err)
	}
	return userID, nil
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

func (db *DBStorage) GetOrders(ctx context.Context) ([]model.Order, error) {
	rows, err := db.DB.QueryContext(ctx, "SELECT number, status, accrual, uploaded_at FROM orders")
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
	row := db.DB.QueryRowContext(ctx, "SELECT current, withdrawn FROM balances")

	var balance model.Balance
	if err := row.Scan(&balance.Current, &balance.Withdrawn); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Balance{}, internal.ErrNoRows
		}
		return model.Balance{}, fmt.Errorf("error get balance from DB: %w", err)
	}

	return balance, nil
}
