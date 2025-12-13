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
	}
	return userID, nil
}

func (db *DBStorage) GetUser(ctx context.Context, login string) (*model.User, error) {
	row := db.DB.QueryRowContext(ctx, "SELECT login, password_hash FROM users WHERE login=$1", login)
	var existedUser model.User
	
	if err := row.Scan(&existedUser.Login, &existedUser.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internal.ErrUserNotFound
		}
		return nil, fmt.Errorf("error get user from DB: %w", err)
	}
	return &existedUser, nil
}