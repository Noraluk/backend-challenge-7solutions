package ports

import (
	"context"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
)

type RegistrationUseCase interface {
	Register(context.Context, dto.RegistrationInput) (dto.UserResult, error)
}

type AuthenticationUseCase interface {
	Authenticate(context.Context, dto.LoginInput) (dto.AuthenticationResult, error)
}
