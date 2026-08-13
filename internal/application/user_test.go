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

const validUserID = "507f1f77bcf86cd799439011"

func TestUserServiceGetUser(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	user := domain.User{ID: validUserID, Name: "Ada", Email: "ada@example.com", PasswordHash: "secret", CreatedAt: time.Now()}
	repository.EXPECT().GetByID(gomock.Any(), validUserID).Return(user, nil)

	result, err := NewUserService(repository).GetUser(context.Background(), validUserID)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if result.ID != user.ID || result.Name != user.Name || result.Email != user.Email {
		t.Errorf("GetUser() = %#v", result)
	}
}

func TestUserServiceGetUserErrors(t *testing.T) {
	databaseError := errors.New("database failed")
	tests := []struct {
		name       string
		id         string
		repoError  error
		want       error
		validation bool
	}{
		{name: "invalid length", id: "invalid", validation: true},
		{name: "invalid hex", id: "zzzzzzzzzzzzzzzzzzzzzzzz", validation: true},
		{name: "not found", id: validUserID, repoError: domain.ErrUserNotFound, want: domain.ErrUserNotFound},
		{name: "repository", id: validUserID, repoError: databaseError, want: databaseError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			if test.id == validUserID {
				repository.EXPECT().GetByID(gomock.Any(), test.id).Return(domain.User{}, test.repoError)
			}
			_, err := NewUserService(repository).GetUser(context.Background(), test.id)
			if test.validation {
				var validationErrors validator.ValidationErrors
				if !errors.As(err, &validationErrors) {
					t.Errorf("GetUser() error = %v, want validation error", err)
				}
				return
			}
			if !errors.Is(err, test.want) {
				t.Errorf("GetUser() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestUserServiceListUsers(t *testing.T) {
	tests := []struct {
		name      string
		users     []domain.User
		repoError error
		wantLen   int
	}{
		{name: "users", users: []domain.User{{ID: validUserID, PasswordHash: "secret"}}, wantLen: 1},
		{name: "empty", users: nil, wantLen: 0},
		{name: "repository", repoError: errors.New("database failed")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			repository.EXPECT().List(gomock.Any()).Return(test.users, test.repoError)
			results, err := NewUserService(repository).ListUsers(context.Background())
			if test.repoError != nil {
				if !errors.Is(err, test.repoError) {
					t.Errorf("ListUsers() error = %v", err)
				}
				return
			}
			if err != nil || len(results) != test.wantLen || results == nil {
				t.Errorf("ListUsers() = %#v, %v", results, err)
			}
		})
	}
}

func TestUserServiceUpdateUser(t *testing.T) {
	tests := []struct {
		name          string
		input         dto.UpdateUserInput
		expectedName  string
		expectedEmail string
	}{
		{name: "name", input: dto.UpdateUserInput{ID: validUserID, Name: stringPointer(" Grace ")}, expectedName: " Grace "},
		{name: "email", input: dto.UpdateUserInput{ID: validUserID, Email: stringPointer("GRACE@Example.COM")}, expectedEmail: "GRACE@Example.COM"},
		{name: "both", input: dto.UpdateUserInput{ID: validUserID, Name: stringPointer(" Grace "), Email: stringPointer("GRACE@Example.COM")}, expectedName: " Grace ", expectedEmail: "GRACE@Example.COM"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			repository.EXPECT().Update(gomock.Any(), validUserID, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ string, input dto.UpdateUserInput) (domain.User, error) {
					if input.Name != nil && *input.Name != test.expectedName {
						t.Errorf("name = %q, want %q", *input.Name, test.expectedName)
					}
					if input.Email != nil && *input.Email != test.expectedEmail {
						t.Errorf("email = %q, want %q", *input.Email, test.expectedEmail)
					}
					return domain.User{ID: validUserID, Name: "Grace", Email: "grace@example.com"}, nil
				},
			)
			result, err := NewUserService(repository).UpdateUser(context.Background(), test.input)
			if err != nil || result.ID != validUserID {
				t.Errorf("UpdateUser() = %#v, %v", result, err)
			}
		})
	}
}

func TestUserServiceUpdateUserErrors(t *testing.T) {
	tests := []struct {
		name      string
		id        string
		input     dto.UpdateUserInput
		repoError error
		want      error
	}{
		{name: "invalid ID", id: "invalid", input: dto.UpdateUserInput{ID: "invalid", Name: stringPointer("Grace")}},
		{name: "empty update", id: validUserID, input: dto.UpdateUserInput{ID: validUserID}},
		{name: "empty name", id: validUserID, input: dto.UpdateUserInput{ID: validUserID, Name: stringPointer("")}},
		{name: "email with spaces", id: validUserID, input: dto.UpdateUserInput{ID: validUserID, Email: stringPointer(" grace@example.com ")}},
		{name: "invalid email", id: validUserID, input: dto.UpdateUserInput{ID: validUserID, Email: stringPointer("invalid")}},
		{name: "duplicate", id: validUserID, input: dto.UpdateUserInput{ID: validUserID, Email: stringPointer("grace@example.com")}, repoError: domain.ErrEmailAlreadyExists, want: domain.ErrEmailAlreadyExists},
		{name: "not found", id: validUserID, input: dto.UpdateUserInput{ID: validUserID, Name: stringPointer("Grace")}, repoError: domain.ErrUserNotFound, want: domain.ErrUserNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			if test.repoError != nil {
				repository.EXPECT().Update(gomock.Any(), test.id, gomock.Any()).Return(domain.User{}, test.repoError)
			}
			_, err := NewUserService(repository).UpdateUser(context.Background(), test.input)
			if test.want != nil {
				if !errors.Is(err, test.want) {
					t.Errorf("UpdateUser() error = %v, want %v", err, test.want)
				}
				return
			}
			var validationErrors validator.ValidationErrors
			if !errors.As(err, &validationErrors) {
				t.Errorf("UpdateUser() error = %v, want validation error", err)
			}
		})
	}
}

func TestUserServiceDeleteUser(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		repoError  error
		want       error
		validation bool
	}{
		{name: "success", id: validUserID},
		{name: "invalid ID", id: "invalid", validation: true},
		{name: "not found", id: validUserID, repoError: domain.ErrUserNotFound, want: domain.ErrUserNotFound},
		{name: "repository", id: validUserID, repoError: errors.New("database failed")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			if test.id == validUserID {
				repository.EXPECT().Delete(gomock.Any(), test.id).Return(test.repoError)
			}
			err := NewUserService(repository).DeleteUser(context.Background(), test.id)
			if test.validation {
				var validationErrors validator.ValidationErrors
				if !errors.As(err, &validationErrors) {
					t.Errorf("DeleteUser() error = %v, want validation error", err)
				}
				return
			}
			if test.want != nil && !errors.Is(err, test.want) {
				t.Errorf("DeleteUser() error = %v, want %v", err, test.want)
			}
			if test.want == nil && test.repoError != nil && !errors.Is(err, test.repoError) {
				t.Errorf("DeleteUser() error = %v", err)
			}
		})
	}
}

func stringPointer(value string) *string {
	return &value
}
