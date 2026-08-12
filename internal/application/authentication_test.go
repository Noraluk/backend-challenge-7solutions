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
	"go.uber.org/mock/gomock"
)

func TestAuthenticationServiceAuthenticate(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	hasher := mocks.NewMockPasswordHasher(controller)
	tokens := mocks.NewMockTokenService(controller)
	now := time.Date(2026, time.August, 11, 12, 0, 0, 0, time.UTC)
	user := domain.User{ID: "user-id", Email: "ada@example.com", PasswordHash: "hashed-password"}

	repository.EXPECT().GetByEmail(gomock.Any(), user.Email).Return(user, nil)
	hasher.EXPECT().Compare(user.PasswordHash, "plain-password").Return(nil)
	tokens.EXPECT().Generate(ports.TokenClaims{UserID: user.ID, ExpiresAt: now.Add(time.Hour)}).Return("access-token", nil)
	service := NewAuthenticationService(repository, hasher, tokens, time.Hour, func() time.Time { return now })

	result, err := service.Authenticate(context.Background(), dto.LoginInput{
		Email:    " ADA@Example.COM ",
		Password: "plain-password",
	})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if result.AccessToken != "access-token" || result.ExpiresIn != time.Hour {
		t.Errorf("result = %#v", result)
	}
}

func TestAuthenticationServiceCredentialFailuresAreEquivalent(t *testing.T) {
	tests := []struct {
		name         string
		findError    error
		compareError error
	}{
		{name: "unknown email", findError: ports.ErrUserNotFound},
		{name: "wrong password", compareError: ports.ErrInvalidCredentials},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			hasher := mocks.NewMockPasswordHasher(controller)
			tokens := mocks.NewMockTokenService(controller)
			user := domain.User{PasswordHash: "hash"}

			repository.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(user, test.findError)
			if test.findError == nil {
				hasher.EXPECT().Compare(user.PasswordHash, "password").Return(test.compareError)
			}
			service := NewAuthenticationService(repository, hasher, tokens, time.Hour, time.Now)

			if _, err := service.Authenticate(context.Background(), dto.LoginInput{Email: "user@example.com", Password: "password"}); !errors.Is(err, ErrInvalidCredentials) {
				t.Errorf("Authenticate() error = %v, want ErrInvalidCredentials", err)
			}
		})
	}
}

func TestAuthenticationServiceRejectsBlankCredentialsBeforeDependencies(t *testing.T) {
	controller := gomock.NewController(t)
	service := NewAuthenticationService(
		mocks.NewMockUserRepository(controller),
		mocks.NewMockPasswordHasher(controller),
		mocks.NewMockTokenService(controller),
		time.Hour,
		nil,
	)

	tests := []dto.LoginInput{
		{Password: "password"},
		{Email: "user@example.com"},
		{Email: "user@example.com", Password: "  "},
	}
	for _, input := range tests {
		if _, err := service.Authenticate(context.Background(), input); !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("Authenticate(%#v) error = %v", input, err)
		}
	}
}

func TestAuthenticationServicePropagatesInternalErrors(t *testing.T) {
	repositoryError := errors.New("database failed")
	compareError := errors.New("hash invalid")
	tokenError := errors.New("token failed")

	tests := []struct {
		name         string
		findError    error
		compareError error
		tokenError   error
		want         error
	}{
		{name: "repository", findError: repositoryError, want: repositoryError},
		{name: "password adapter", compareError: compareError, want: compareError},
		{name: "token adapter", tokenError: tokenError, want: tokenError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			repository := mocks.NewMockUserRepository(controller)
			hasher := mocks.NewMockPasswordHasher(controller)
			tokens := mocks.NewMockTokenService(controller)
			user := domain.User{ID: "user-id", PasswordHash: "hash"}

			repository.EXPECT().GetByEmail(gomock.Any(), "user@example.com").Return(user, test.findError)
			if test.findError == nil {
				hasher.EXPECT().Compare(user.PasswordHash, "password").Return(test.compareError)
			}
			if test.findError == nil && test.compareError == nil {
				tokens.EXPECT().Generate(gomock.Any()).Return("token", test.tokenError)
			}
			service := NewAuthenticationService(repository, hasher, tokens, time.Hour, time.Now)

			if _, err := service.Authenticate(context.Background(), dto.LoginInput{Email: "user@example.com", Password: "password"}); !errors.Is(err, test.want) {
				t.Errorf("Authenticate() error = %v, want %v", err, test.want)
			}
		})
	}
}
