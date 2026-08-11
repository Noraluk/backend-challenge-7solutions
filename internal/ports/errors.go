package ports

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidUserID      = errors.New("invalid user ID")
	ErrInvalidUpdate      = errors.New("invalid user update")
)
