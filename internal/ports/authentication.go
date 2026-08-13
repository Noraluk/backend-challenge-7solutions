package ports

import (
	"context"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
)

type RegistrationUseCase interface {
	Register(context.Context, dto.RegistrationInput) (dto.UserResponse, error)
}

type AuthenticationUseCase interface {
	Login(context.Context, dto.LoginInput) (dto.LoginResponse, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
}

type TokenClaims struct {
	UserID    string
	ExpiresAt time.Time
}

type TokenService interface {
	Generate(claims TokenClaims) (string, error)
	Validate(token string) (TokenClaims, error)
}
