package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type AuthenticationService struct {
	repository ports.UserRepository
	hasher     ports.PasswordHasher
	tokens     ports.TokenService
	tokenTTL   time.Duration
	now        func() time.Time
}

func NewAuthenticationService(
	repository ports.UserRepository,
	hasher ports.PasswordHasher,
	tokens ports.TokenService,
	tokenTTL time.Duration,
	now func() time.Time,
) *AuthenticationService {
	if now == nil {
		now = time.Now
	}

	return &AuthenticationService{
		repository: repository,
		hasher:     hasher,
		tokens:     tokens,
		tokenTTL:   tokenTTL,
		now:        now,
	}
}

func (s *AuthenticationService) Authenticate(ctx context.Context, input dto.LoginInput) (dto.AuthenticationResult, error) {
	email := normalizeEmail(input.Email)
	if email == "" || strings.TrimSpace(input.Password) == "" {
		return dto.AuthenticationResult{}, ErrInvalidCredentials
	}

	user, err := s.repository.GetByEmail(ctx, email)
	if errors.Is(err, ports.ErrUserNotFound) {
		return dto.AuthenticationResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return dto.AuthenticationResult{}, fmt.Errorf("get user for authentication: %w", err)
	}

	if err := s.hasher.Compare(user.PasswordHash, input.Password); errors.Is(err, ports.ErrInvalidCredentials) {
		return dto.AuthenticationResult{}, ErrInvalidCredentials
	} else if err != nil {
		return dto.AuthenticationResult{}, fmt.Errorf("compare authentication password: %w", err)
	}

	token, err := s.tokens.Generate(ports.TokenClaims{
		UserID:    user.ID,
		ExpiresAt: s.now().Add(s.tokenTTL),
	})
	if err != nil {
		return dto.AuthenticationResult{}, fmt.Errorf("generate authentication token: %w", err)
	}

	return dto.AuthenticationResult{AccessToken: token, ExpiresIn: s.tokenTTL}, nil
}
