package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type AuthenticationService struct {
	repository ports.UserRepository
	hasher     ports.PasswordHasher
	tokens     ports.TokenService
	tokenTTL   time.Duration
}

func NewAuthenticationService(
	repository ports.UserRepository,
	hasher ports.PasswordHasher,
	tokens ports.TokenService,
	tokenTTL time.Duration,
) *AuthenticationService {
	return &AuthenticationService{
		repository: repository,
		hasher:     hasher,
		tokens:     tokens,
		tokenTTL:   tokenTTL,
	}
}

func (s *AuthenticationService) Login(ctx context.Context, input dto.LoginInput) (dto.LoginResponse, error) {
	if err := input.Validate(); err != nil {
		return dto.LoginResponse{}, domain.ErrInvalidCredentials
	}

	user, err := s.repository.GetByEmail(ctx, input.Email)
	if errors.Is(err, domain.ErrUserNotFound) {
		return dto.LoginResponse{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("get user for authentication: %w", err)
	}

	if err := s.hasher.Compare(user.PasswordHash, input.Password); errors.Is(err, domain.ErrInvalidCredentials) {
		return dto.LoginResponse{}, domain.ErrInvalidCredentials
	} else if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("compare authentication password: %w", err)
	}

	now := time.Now()
	token, err := s.tokens.Generate(ports.TokenClaims{
		UserID:    user.ID,
		ExpiresAt: now.Add(s.tokenTTL),
	})
	if err != nil {
		return dto.LoginResponse{}, fmt.Errorf("generate authentication token: %w", err)
	}

	return dto.LoginResponse{AccessToken: token, ExpiresIn: s.tokenTTL}, nil
}
