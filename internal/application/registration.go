package application

import (
	"context"
	"fmt"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type RegistrationService struct {
	repository ports.UserRepository
	hasher     ports.PasswordHasher
}

func NewRegistrationService(repository ports.UserRepository, hasher ports.PasswordHasher) *RegistrationService {
	return &RegistrationService{repository: repository, hasher: hasher}
}

func (s *RegistrationService) Register(ctx context.Context, input dto.RegistrationInput) (dto.UserResponse, error) {
	if err := input.Validate(); err != nil {
		return dto.UserResponse{}, err
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return dto.UserResponse{}, fmt.Errorf("hash registration password: %w", err)
	}

	now := time.Now().UTC()
	user, err := s.repository.Create(ctx, domain.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
	})
	if err != nil {
		return dto.UserResponse{}, fmt.Errorf("create user: %w", err)
	}

	return dto.NewUserResult(user), nil
}
