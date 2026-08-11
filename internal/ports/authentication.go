package ports

import (
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(passwordHash, password string) error
}

type TokenClaims struct {
	UserID    domain.UserID
	ExpiresAt time.Time
}

type TokenService interface {
	Generate(claims TokenClaims) (string, error)
	Validate(token string) (TokenClaims, error)
}
