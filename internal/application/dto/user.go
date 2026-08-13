package dto

import (
	"strings"
	"time"

	userv1 "github.com/Noraluk/backend-challenge-7solutions/gen/user/v1"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/go-playground/validator/v10"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

type RegistrationInput struct {
	Name     string `validate:"required"`
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

func (input *RegistrationInput) Validate() error {
	input.Name = strings.TrimSpace(input.Name)
	validationInput := *input
	validationInput.Password = strings.TrimSpace(validationInput.Password)
	return validate.Struct(validationInput)
}

type UserIDInput struct {
	ID string `validate:"required,len=24,hexadecimal"`
}

func (input UserIDInput) Validate() error {
	return validate.Struct(input)
}

type UpdateUserInput struct {
	ID    string  `validate:"required,len=24,hexadecimal"`
	Name  *string `validate:"required_without=Email,omitempty,min=1"`
	Email *string `validate:"required_without=Name,omitempty,min=1,email"`
}

func (input UpdateUserInput) Validate() error {
	return validate.Struct(input)
}

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

func (user UserResponse) ToProto() *userv1.User {
	return &userv1.User{
		Id:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}

func NewUserResult(user domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	}
}
