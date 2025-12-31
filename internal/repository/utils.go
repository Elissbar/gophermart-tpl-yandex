package repository

import (
	"database/sql"
	"errors"
	"gophermart/internal"
	"gophermart/internal/model"
)

func scanOrders(rows *sql.Rows) (*[]model.Order, error) {
	var orders []model.Order
	for rows.Next() {
		var order model.Order

		err := rows.Scan(
			&order.ID,
			&order.Number,
			&order.Status,
			&order.Accrual,
			&order.UploadedAt,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}

	return &orders, nil
}

func scanWithdrawals(rows *sql.Rows) (*[]model.Withdraw, error) {
	var withdrawals []model.Withdraw
	for rows.Next() {
		var withdrawn model.Withdraw

		err := rows.Scan(
			&withdrawn.Order,
			&withdrawn.Sum,
			&withdrawn.ProcessedAt,
		)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawn)
	}

	return &withdrawals, nil
}

func scanBalance(row *sql.Row) (*model.Balance, error) {
	var balance model.Balance
	if err := row.Scan(&balance.Current, &balance.Withdrawn); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internal.ErrNoRows
		}
		return nil, err
	}

	return &balance, nil
}


func scanUser(row *sql.Row) (*model.User, error) {
	var existedUser model.User

	if err := row.Scan(&existedUser.ID, &existedUser.Login, &existedUser.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internal.ErrUserNotFound
		}
		return nil, err
	}
	return &existedUser, nil
}