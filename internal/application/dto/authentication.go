package dto

import (
	"strings"
	"time"
)

type LoginInput struct {
	Email    string `validate:"required"`
	Password string `validate:"required"`
}

func (input LoginInput) Validate() error {
	input.Email = strings.TrimSpace(input.Email)
	input.Password = strings.TrimSpace(input.Password)
	return validate.Struct(input)
}

type LoginResponse struct {
	AccessToken string
	ExpiresIn   time.Duration
}
