package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type Repository[T any] struct {
	DB       *sql.DB
	Table string
}

func (r *Repository[T]) GetAllRows(
	ctx context.Context,
	scan func(rows *sql.Rows) (*[]T, error),
	query string,
	args []any,
) (*[]T, error) {
	rows, err := r.DB.QueryContext(
		ctx,
		query,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("error get %s from DB: %w", r.Table, err)
	}
	defer rows.Close()

	values, err := scan(rows)
	if err != nil {
		return nil, fmt.Errorf("error scan %s from DB: %w", r.Table, err)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows for %s: %w", r.Table, err)
	}

	return values, nil
}

func (r *Repository[T]) GetOneRow(
	ctx context.Context,
	scan func(row *sql.Row) (*T, error),
	query string,
	args []any,
) (*T, error) {
	row := r.DB.QueryRowContext(ctx, query, args...)
	value, err := scan(row)
	if err != nil {
		return value, fmt.Errorf("error scan %s from DB: %w", r.Table, err)
	}
	return value, nil
}
