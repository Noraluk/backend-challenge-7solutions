package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	applicationdto "github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"github.com/Noraluk/backend-challenge-7solutions/internal/testutil"
	"github.com/go-playground/validator/v10"
	"go.uber.org/mock/gomock"
)

func TestNewAuthHandler(t *testing.T) {
	controller := gomock.NewController(t)
	registration := mocks.NewMockRegistrationUseCase(controller)
	authentication := mocks.NewMockAuthenticationUseCase(controller)
	handler := NewAuthHandler(registration, authentication)
	if handler.registration != registration || handler.authentication != authentication {
		t.Errorf("NewAuthHandler() = %#v", handler)
	}
}

func TestAuthHandlerRegisterUser(t *testing.T) {
	controller := gomock.NewController(t)
	registration := mocks.NewMockRegistrationUseCase(controller)
	input := applicationdto.RegistrationInput{Name: "Ada", Email: "ada@example.com", Password: "password"}
	want := applicationdto.UserResponse{ID: "user-id", Name: input.Name, Email: input.Email}
	registration.EXPECT().Register(gomock.Any(), input).Return(want, nil)

	response := httptest.NewRecorder()
	NewAuthHandler(registration, nil).RegisterUser(response, httptest.NewRequest(
		http.MethodPost,
		"/auth/register",
		strings.NewReader(`{"name":"Ada","email":"ada@example.com","password":"password"}`),
	))

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	testutil.AssertJSONContentType(t, response)
	var got applicationdto.UserResponse
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got != want {
		t.Errorf("response = %#v, want %#v", got, want)
	}
}

func TestAuthHandlerRegisterUserErrors(t *testing.T) {
	validationError := validator.New().Struct(struct {
		Name string `validate:"required"`
	}{})
	tests := []struct {
		name       string
		body       string
		usecaseErr error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid request", body: `{"name":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "validation", body: `{"name":"","email":"ada@example.com","password":"password"}`, usecaseErr: validationError, wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR"},
		{name: "duplicate", body: `{"name":"Ada","email":"ada@example.com","password":"password"}`, usecaseErr: domain.ErrEmailAlreadyExists, wantStatus: http.StatusConflict, wantCode: "EMAIL_ALREADY_EXISTS"},
		{name: "internal", body: `{"name":"Ada","email":"ada@example.com","password":"password"}`, usecaseErr: errors.New("database secret"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			registration := mocks.NewMockRegistrationUseCase(controller)
			if test.usecaseErr != nil {
				registration.EXPECT().Register(gomock.Any(), gomock.Any()).Return(applicationdto.UserResponse{}, test.usecaseErr)
			}
			response := httptest.NewRecorder()
			NewAuthHandler(registration, nil).RegisterUser(response, httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(test.body)))
			testutil.AssertAPIError(t, response, test.wantStatus, test.wantCode)
			if strings.Contains(response.Body.String(), "database secret") {
				t.Errorf("response leaks internal error: %s", response.Body)
			}
		})
	}
}

func TestAuthHandlerLogin(t *testing.T) {
	controller := gomock.NewController(t)
	authentication := mocks.NewMockAuthenticationUseCase(controller)
	input := applicationdto.LoginInput{Email: "ada@example.com", Password: "password"}
	authentication.EXPECT().Login(gomock.Any(), input).Return(applicationdto.LoginResponse{
		AccessToken: "token", ExpiresIn: time.Hour,
	}, nil)

	response := httptest.NewRecorder()
	NewAuthHandler(nil, authentication).Login(response, httptest.NewRequest(
		http.MethodPost,
		"/auth/login",
		strings.NewReader(`{"email":"ada@example.com","password":"password"}`),
	))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	testutil.AssertJSONContentType(t, response)
	var got httpdto.LoginResponse
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.AccessToken != "token" || got.TokenType != "Bearer" || got.ExpiresIn != 3600 {
		t.Errorf("response = %#v", got)
	}
}

func TestAuthHandlerLoginErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		usecaseErr error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid request", body: `{"email":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "credentials", body: `{"email":"ada@example.com","password":"wrong"}`, usecaseErr: domain.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantCode: "INVALID_CREDENTIALS"},
		{name: "internal", body: `{"email":"ada@example.com","password":"password"}`, usecaseErr: errors.New("JWT secret"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			authentication := mocks.NewMockAuthenticationUseCase(controller)
			if test.usecaseErr != nil {
				authentication.EXPECT().Login(gomock.Any(), gomock.Any()).Return(applicationdto.LoginResponse{}, test.usecaseErr)
			}
			response := httptest.NewRecorder()
			NewAuthHandler(nil, authentication).Login(response, httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(test.body)))
			testutil.AssertAPIError(t, response, test.wantStatus, test.wantCode)
			if strings.Contains(response.Body.String(), "JWT secret") {
				t.Errorf("response leaks internal error: %s", response.Body)
			}
		})
	}
}
