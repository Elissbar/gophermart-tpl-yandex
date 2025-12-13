package internal

import "errors"

var (
	ErrLoginExists error = errors.New("login already exists")
	ErrUserNotFound error = errors.New("user not found")
)
