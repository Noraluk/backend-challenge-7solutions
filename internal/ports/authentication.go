package ports

import (
	"time"
)

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
