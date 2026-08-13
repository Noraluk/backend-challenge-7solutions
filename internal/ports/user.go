package ports

import (
	"context"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
)

type UserUseCase interface {
	GetUser(context.Context, string) (dto.UserResponse, error)
	ListUsers(context.Context) ([]dto.UserResponse, error)
	UpdateUser(context.Context, dto.UpdateUserInput) (dto.UserResponse, error)
	DeleteUser(context.Context, string) error
}

type UserRepository interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id string, update dto.UpdateUserInput) (domain.User, error)
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}
