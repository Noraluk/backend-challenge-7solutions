package dto

import "time"

type RegistrationInput struct {
	Name     string `validate:"required"`
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

type UpdateUserInput struct {
	Name  *string `validate:"omitempty,min=1"`
	Email *string `validate:"omitempty,min=1,email"`
}

type UserResult struct {
	ID        string
	Name      string
	Email     string
	CreatedAt time.Time
}
