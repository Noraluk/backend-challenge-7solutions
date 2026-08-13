package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"github.com/go-playground/validator/v10"
	"go.uber.org/mock/gomock"
)

func TestRegistrationServiceRegister(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	hasher := mocks.NewMockPasswordHasher(controller)
	beforeRegister := time.Now().UTC()
	var persisted domain.User

	hasher.EXPECT().Hash("plain-password").Return("hashed-password", nil)
	repository.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, user domain.User) (domain.User, error) {
			persisted = user
			user.ID = "user-id"
			return user, nil
		},
	)
	service := NewRegistrationService(repository, hasher)

	result, err := service.Register(context.Background(), dto.RegistrationInput{
		Name:     " Ada Lovelace ",
		Email:    "ADA@Example.COM",
		Password: "plain-password",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if persisted.Name != "Ada Lovelace" || persisted.Email != "ADA@Example.COM" {
		t.Errorf("persisted user = %#v", persisted)
	}
	if persisted.PasswordHash != "hashed-password" {
		t.Errorf("PasswordHash = %q, want hashed password", persisted.PasswordHash)
	}
	afterRegister := time.Now().UTC()
	if persisted.CreatedAt.Before(beforeRegister) || persisted.CreatedAt.After(afterRegister) {
		t.Errorf("CreatedAt = %s, want between %s and %s", persisted.CreatedAt, beforeRegister, afterRegister)
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
		{name: "duplicate email", createError: domain.ErrEmailAlreadyExists, want: domain.ErrEmailAlreadyExists},
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
			service := NewRegistrationService(repository, hasher)

			if _, err := service.Register(context.Background(), validInput); !errors.Is(err, test.want) {
				t.Errorf("Register() error = %v, want %v", err, test.want)
			}
		})
	}
}
