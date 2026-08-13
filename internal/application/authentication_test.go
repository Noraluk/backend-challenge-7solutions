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

func TestAuthenticationServiceLogin(t *testing.T) {
	controller := gomock.NewController(t)
	repository := mocks.NewMockUserRepository(controller)
	hasher := mocks.NewMockPasswordHasher(controller)
	tokens := mocks.NewMockTokenService(controller)
	user := domain.User{ID: "user-id", Email: "ada@example.com", PasswordHash: "hashed-password"}

	repository.EXPECT().GetByEmail(gomock.Any(), " ADA@Example.COM ").Return(user, nil)
	hasher.EXPECT().Compare(user.PasswordHash, "plain-password").Return(nil)
	beforeLogin := time.Now()
	tokens.EXPECT().Generate(gomock.Any()).DoAndReturn(
		func(claims ports.TokenClaims) (string, error) {
			if claims.UserID != user.ID {
				t.Errorf("UserID = %q, want %q", claims.UserID, user.ID)
			}
			minimumExpiration := beforeLogin.Add(time.Hour)
			maximumExpiration := time.Now().Add(time.Hour)
			if claims.ExpiresAt.Before(minimumExpiration) || claims.ExpiresAt.After(maximumExpiration) {
				t.Errorf("ExpiresAt = %s, want between %s and %s", claims.ExpiresAt, minimumExpiration, maximumExpiration)
			}
			return "access-token", nil
		},
	)
	service := NewAuthenticationService(repository, hasher, tokens, time.Hour)

	result, err := service.Login(context.Background(), dto.LoginInput{
		Email:    " ADA@Example.COM ",
		Password: "plain-password",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
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
		{name: "unknown email", findError: domain.ErrUserNotFound},
		{name: "wrong password", compareError: domain.ErrInvalidCredentials},
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
			service := NewAuthenticationService(repository, hasher, tokens, time.Hour)

			if _, err := service.Login(context.Background(), dto.LoginInput{Email: "user@example.com", Password: "password"}); !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
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
	)

	tests := []dto.LoginInput{
		{Password: "password"},
		{Email: "user@example.com"},
		{Email: "user@example.com", Password: "  "},
	}
	for _, input := range tests {
		if _, err := service.Login(context.Background(), input); !errors.Is(err, domain.ErrInvalidCredentials) {
			t.Errorf("Login(%#v) error = %v", input, err)
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
			service := NewAuthenticationService(repository, hasher, tokens, time.Hour)

			if _, err := service.Login(context.Background(), dto.LoginInput{Email: "user@example.com", Password: "password"}); !errors.Is(err, test.want) {
				t.Errorf("Login() error = %v, want %v", err, test.want)
			}
		})
	}
}
