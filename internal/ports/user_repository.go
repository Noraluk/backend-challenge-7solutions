package ports

import (
	"context"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
)

type UserUpdate struct {
	Name  *string
	Email *string
}

type UserRepository interface {
	Create(ctx context.Context, user domain.User) (domain.User, error)
	GetByID(ctx context.Context, id domain.UserID) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id domain.UserID, update UserUpdate) (domain.User, error)
	Delete(ctx context.Context, id domain.UserID) error
	Count(ctx context.Context) (int64, error)
}
