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

type RegistrationService struct {
	repository ports.UserRepository
	hasher     ports.PasswordHasher
	now        func() time.Time
}

func NewRegistrationService(repository ports.UserRepository, hasher ports.PasswordHasher, now func() time.Time) *RegistrationService {
	if now == nil {
		now = time.Now
	}

	return &RegistrationService{repository: repository, hasher: hasher, now: now}
}

func (s *RegistrationService) Register(ctx context.Context, input dto.RegistrationInput) (dto.UserResult, error) {
	input, err := ValidateRegistration(input)
	if err != nil {
		return dto.UserResult{}, err
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return dto.UserResult{}, fmt.Errorf("hash registration password: %w", err)
	}

	user, err := s.repository.Create(ctx, domain.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: passwordHash,
		CreatedAt:    s.now().UTC(),
	})
	if errors.Is(err, ports.ErrEmailAlreadyExists) {
		return dto.UserResult{}, ErrEmailAlreadyExists
	}
	if err != nil {
		return dto.UserResult{}, fmt.Errorf("create user: %w", err)
	}

	return userResult(user), nil
}
