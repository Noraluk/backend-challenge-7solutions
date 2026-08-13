package application

import (
	"context"
	"fmt"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type UserService struct {
	repository ports.UserRepository
}

func NewUserService(repository ports.UserRepository) *UserService {
	return &UserService{repository: repository}
}

func (s *UserService) GetUser(ctx context.Context, id string) (dto.UserResponse, error) {
	input := dto.UserIDInput{ID: id}
	if err := input.Validate(); err != nil {
		return dto.UserResponse{}, err
	}

	user, err := s.repository.GetByID(ctx, input.ID)
	if err != nil {
		return dto.UserResponse{}, fmt.Errorf("get user: %w", err)
	}
	return dto.NewUserResult(user), nil
}

func (s *UserService) ListUsers(ctx context.Context) ([]dto.UserResponse, error) {
	users, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	results := make([]dto.UserResponse, len(users))
	for index, user := range users {
		results[index] = dto.NewUserResult(user)
	}
	return results, nil
}

func (s *UserService) UpdateUser(ctx context.Context, input dto.UpdateUserInput) (dto.UserResponse, error) {
	if err := input.Validate(); err != nil {
		return dto.UserResponse{}, err
	}

	user, err := s.repository.Update(ctx, input.ID, input)
	if err != nil {
		return dto.UserResponse{}, fmt.Errorf("update user: %w", err)
	}
	return dto.NewUserResult(user), nil
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	input := dto.UserIDInput{ID: id}
	if err := input.Validate(); err != nil {
		return err
	}
	if err := s.repository.Delete(ctx, input.ID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
