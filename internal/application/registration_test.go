package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"github.com/go-playground/validator/v10"
	"go.uber.org/mock/gomock"
)

func TestRegistrationServiceRegister(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	hasher := mocks.NewMockPasswordHasher(controller)
	now := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.FixedZone("test", 7*60*60))
	var persisted domain.User

	hasher.EXPECT().Hash("plain-password").Return("hashed-password", nil)
	repository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, user domain.User) (domain.User, error) {
			persisted = user
			user.ID = "user-id"
			return user, nil
		},
	)
	service := NewRegistrationService(repository, hasher, func() time.Time { return now })

	result, err := service.Register(context.Background(), dto.RegistrationInput{
		Name:     " Ada Lovelace ",
		Email:    " ADA@Example.COM ",
		Password: "plain-password",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if persisted.Name != "Ada Lovelace" || persisted.Email != "ada@example.com" {
		t.Errorf("persisted user = %#v", persisted)
	}
	if persisted.PasswordHash != "hashed-password" {
		t.Errorf("PasswordHash = %q, want hashed password", persisted.PasswordHash)
	}
	if !persisted.CreatedAt.Equal(now.UTC()) {
		t.Errorf("CreatedAt = %s, want %s", persisted.CreatedAt, now.UTC())
	}
	if result.ID != "user-id" || result.Name != persisted.Name || result.Email != persisted.Email {
		t.Errorf("result = %#v", result)
	}
}

func TestRegistrationServiceRejectsValidationBeforeDependencies(t *testing.T) {
	controller := gomock.NewController(t)
	service := NewRegistrationService(
		mocks.NewMockUserRepository(controller),
		mocks.NewMockPasswordHasher(controller),
		time.Now,
	)

	if _, err := service.Register(context.Background(), dto.RegistrationInput{}); err == nil {
		t.Error("Register() error = nil, want validation error")
	} else {
		var validationErrors validator.ValidationErrors
		if !errors.As(err, &validationErrors) {
			t.Errorf("Register() error type = %T, want validator.ValidationErrors", err)
		}
	}
}

func TestRegistrationServiceUsesDefaultClock(t *testing.T) {
	controller := gomock.NewController(t)
	service := NewRegistrationService(
		mocks.NewMockUserRepository(controller),
		mocks.NewMockPasswordHasher(controller),
		nil,
	)
	if service.now == nil {
		t.Fatal("NewRegistrationService() did not set a default clock")
	}
}

func TestRegistrationServiceErrors(t *testing.T) {
	hashingError := errors.New("hashing failed")
	repositoryError := errors.New("database failed")
	validInput := dto.RegistrationInput{Name: "Ada", Email: "ada@example.com", Password: "password"}

	tests := []struct {
		name        string
		hashError   error
		createError error
		want        error
	}{
		{name: "hash failure", hashError: hashingError, want: hashingError},
		{name: "duplicate email", createError: ports.ErrEmailAlreadyExists, want: ErrEmailAlreadyExists},
		{name: "repository failure", createError: repositoryError, want: repositoryError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			hasher := mocks.NewMockPasswordHasher(controller)
			hasher.EXPECT().Hash(validInput.Password).Return("hash", test.hashError)
			if test.hashError == nil {
				repository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, user domain.User) (domain.User, error) {
						return user, test.createError
					},
				)
			}
			service := NewRegistrationService(repository, hasher, time.Now)

			if _, err := service.Register(context.Background(), validInput); !errors.Is(err, test.want) {
				t.Errorf("Register() error = %v, want %v", err, test.want)
			}
		})
	}
}
