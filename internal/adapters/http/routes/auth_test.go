package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application"
	applicationdto "github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"go.uber.org/mock/gomock"
)

func authHandler(routes *AuthRoutes) http.Handler {
	mux := http.NewServeMux()
	routes.Register(mux)
	return mux
}

func TestRegisterUser(t *testing.T) {
	controller := gomock.NewController(t)
	registration := mocks.NewMockRegistrationUseCase(controller)
	createdAt := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	input := applicationdto.RegistrationInput{Name: "Ada", Email: "ada@example.com", Password: "password"}
	registration.EXPECT().Register(gomock.Any(), input).Return(applicationdto.UserResult{
		ID: "user-id", Name: input.Name, Email: input.Email, CreatedAt: createdAt,
	}, nil)

	response := httptest.NewRecorder()
	authHandler(NewAuthRoutes(registration, nil)).ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"password"}`)),
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	if strings.Contains(response.Body.String(), "password") {
		t.Errorf("response exposes password: %s", response.Body)
	}
	var body httpdto.UserResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ID != "user-id" || body.Email != input.Email || !body.CreatedAt.Equal(createdAt) {
		t.Errorf("response = %#v", body)
	}
}

func TestRegisterUserErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		usecaseErr error
		wantStatus int
	}{
		{name: "malformed", body: `{"name":`, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"name":"Ada","email":"ada@example.com","password":"password","role":"admin"}`, wantStatus: http.StatusBadRequest},
		{name: "duplicate", body: `{"name":"Ada","email":"ada@example.com","password":"password"}`, usecaseErr: application.ErrEmailAlreadyExists, wantStatus: http.StatusConflict},
		{name: "internal", body: `{"name":"Ada","email":"ada@example.com","password":"password"}`, usecaseErr: errors.New("database failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			registration := mocks.NewMockRegistrationUseCase(controller)
			if test.usecaseErr != nil {
				registration.EXPECT().Register(gomock.Any(), gomock.Any()).Return(applicationdto.UserResult{}, test.usecaseErr)
			}
			response := httptest.NewRecorder()
			authHandler(NewAuthRoutes(registration, nil)).ServeHTTP(response,
				httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(test.body)),
			)
			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	controller := gomock.NewController(t)
	authentication := mocks.NewMockAuthenticationUseCase(controller)
	input := applicationdto.LoginInput{Email: "ada@example.com", Password: "password"}
	authentication.EXPECT().Authenticate(gomock.Any(), input).Return(
		applicationdto.AuthenticationResult{AccessToken: "signed-token", ExpiresIn: time.Hour}, nil,
	)

	response := httptest.NewRecorder()
	authHandler(NewAuthRoutes(nil, authentication)).ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ada@example.com","password":"password"}`)),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	var body httpdto.LoginResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.AccessToken != "signed-token" || body.TokenType != "Bearer" || body.ExpiresIn != 3600 {
		t.Errorf("response = %#v", body)
	}
}

func TestLoginErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		usecaseErr error
		wantStatus int
	}{
		{name: "malformed", body: `{"email":`, wantStatus: http.StatusBadRequest},
		{name: "invalid credentials", body: `{"email":"ada@example.com","password":"wrong"}`, usecaseErr: application.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized},
		{name: "internal", body: `{"email":"ada@example.com","password":"password"}`, usecaseErr: errors.New("token failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			authentication := mocks.NewMockAuthenticationUseCase(controller)
			if test.usecaseErr != nil {
				authentication.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(applicationdto.AuthenticationResult{}, test.usecaseErr)
			}
			response := httptest.NewRecorder()
			authHandler(NewAuthRoutes(nil, authentication)).ServeHTTP(response,
				httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(test.body)),
			)
			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d; body = %s", response.Code, test.wantStatus, response.Body)
			}
		})
	}
}
