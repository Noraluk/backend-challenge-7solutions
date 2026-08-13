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
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/handlers"
	applicationdto "github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"github.com/Noraluk/backend-challenge-7solutions/internal/testutil"
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
	registration.EXPECT().Register(gomock.Any(), input).Return(applicationdto.UserResponse{
		ID: "user-id", Name: input.Name, Email: input.Email, CreatedAt: createdAt,
	}, nil)

	response := httptest.NewRecorder()
	authHandler(NewAuthRoutes(handlers.NewAuthHandler(registration, nil))).ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"password"}`)),
	)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	testutil.AssertJSONContentType(t, response)
	if strings.Contains(response.Body.String(), "password") {
		t.Errorf("response exposes password: %s", response.Body)
	}
	var body applicationdto.UserResponse
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
		wantCode   string
	}{
		{name: "malformed", body: `{"name":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "unknown field", body: `{"name":"Ada","email":"ada@example.com","password":"password","role":"admin"}`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "duplicate", body: `{"name":"Ada","email":"ada@example.com","password":"password"}`, usecaseErr: domain.ErrEmailAlreadyExists, wantStatus: http.StatusConflict, wantCode: "EMAIL_ALREADY_EXISTS"},
		{name: "internal", body: `{"name":"Ada","email":"ada@example.com","password":"password"}`, usecaseErr: errors.New("database failed"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			registration := mocks.NewMockRegistrationUseCase(controller)
			if test.usecaseErr != nil {
				registration.EXPECT().Register(gomock.Any(), gomock.Any()).Return(applicationdto.UserResponse{}, test.usecaseErr)
			}
			response := httptest.NewRecorder()
			authHandler(NewAuthRoutes(handlers.NewAuthHandler(registration, nil))).ServeHTTP(response,
				httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(test.body)),
			)
			testutil.AssertAPIError(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestLogin(t *testing.T) {
	controller := gomock.NewController(t)
	authentication := mocks.NewMockAuthenticationUseCase(controller)
	input := applicationdto.LoginInput{Email: "ada@example.com", Password: "password"}
	authentication.EXPECT().Login(gomock.Any(), input).Return(
		applicationdto.LoginResponse{AccessToken: "signed-token", ExpiresIn: time.Hour}, nil,
	)

	response := httptest.NewRecorder()
	authHandler(NewAuthRoutes(handlers.NewAuthHandler(nil, authentication))).ServeHTTP(response,
		httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(`{"email":"ada@example.com","password":"password"}`)),
	)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	testutil.AssertJSONContentType(t, response)
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
		wantCode   string
	}{
		{name: "malformed", body: `{"email":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "invalid credentials", body: `{"email":"ada@example.com","password":"wrong"}`, usecaseErr: domain.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantCode: "INVALID_CREDENTIALS"},
		{name: "internal", body: `{"email":"ada@example.com","password":"password"}`, usecaseErr: errors.New("token failed"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			authentication := mocks.NewMockAuthenticationUseCase(controller)
			if test.usecaseErr != nil {
				authentication.EXPECT().Login(gomock.Any(), gomock.Any()).Return(applicationdto.LoginResponse{}, test.usecaseErr)
			}
			response := httptest.NewRecorder()
			authHandler(NewAuthRoutes(handlers.NewAuthHandler(nil, authentication))).ServeHTTP(response,
				httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(test.body)),
			)
			testutil.AssertAPIError(t, response, test.wantStatus, test.wantCode)
		})
	}
}
