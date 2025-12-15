package internal

import "errors"

var (
	ErrLoginExists error = errors.New("login already exists")
	ErrUserNotFound error = errors.New("user not found")
	ErrOrderAlreadyUploadedByUser error = errors.New("number already uploaded by user")
	ErrOrderUploadConflict error = errors.New("order uploaded by other user")
)
