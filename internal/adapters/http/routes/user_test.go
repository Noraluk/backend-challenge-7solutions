package routes

import (
	"context"
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
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"github.com/go-playground/validator/v10"
	"go.uber.org/mock/gomock"
)

const validUserID = "507f1f77bcf86cd799439011"

func userHandler(t *testing.T, users *mocks.MockUserUseCase, tokens *mocks.MockTokenService) http.Handler {
	t.Helper()
	tokens.EXPECT().Validate("valid-token").Return(ports.TokenClaims{UserID: "authenticated-user"}, nil).AnyTimes()
	mux := http.NewServeMux()
	NewUserRoutes(handlers.NewUserHandler(users), tokens).Register(mux)
	return mux
}

func authenticatedRequest(method, target, body string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer valid-token")
	return request
}

func TestUserRoutesGet(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	tokens := mocks.NewMockTokenService(controller)
	result := applicationdto.UserResponse{ID: validUserID, Name: "Ada", Email: "ada@example.com", CreatedAt: time.Now()}
	users.EXPECT().GetUser(gomock.Any(), validUserID).Return(result, nil)
	response := httptest.NewRecorder()

	userHandler(t, users, tokens).ServeHTTP(response, authenticatedRequest(http.MethodGet, "/users/"+validUserID, ""))

	if response.Code != http.StatusOK || strings.Contains(response.Body.String(), "password") {
		t.Fatalf("response = %d %s", response.Code, response.Body)
	}
	var body applicationdto.UserResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil || body.ID != validUserID {
		t.Errorf("response = %#v, %v", body, err)
	}
}

func TestUserRoutesList(t *testing.T) {
	tests := []struct {
		name    string
		results []applicationdto.UserResponse
	}{
		{name: "users", results: []applicationdto.UserResponse{{ID: validUserID, Name: "Ada"}}},
		{name: "empty", results: []applicationdto.UserResponse{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			users := mocks.NewMockUserUseCase(controller)
			tokens := mocks.NewMockTokenService(controller)
			users.EXPECT().ListUsers(gomock.Any()).Return(test.results, nil)
			response := httptest.NewRecorder()
			userHandler(t, users, tokens).ServeHTTP(response, authenticatedRequest(http.MethodGet, "/users", ""))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d", response.Code)
			}
			var body []applicationdto.UserResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil || len(body) != len(test.results) || body == nil {
				t.Errorf("response = %#v, %v", body, err)
			}
		})
	}
}

func TestUserRoutesUpdate(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	tokens := mocks.NewMockTokenService(controller)
	users.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, input applicationdto.UpdateUserInput) (applicationdto.UserResponse, error) {
			if input.ID != validUserID || input.Name == nil || *input.Name != "Grace" || input.Email != nil {
				t.Errorf("update input = %#v", input)
			}
			return applicationdto.UserResponse{ID: validUserID, Name: *input.Name}, nil
		},
	)
	response := httptest.NewRecorder()
	userHandler(t, users, tokens).ServeHTTP(response,
		authenticatedRequest(http.MethodPatch, "/users/"+validUserID, `{"name":"Grace"}`),
	)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Grace") {
		t.Errorf("response = %d %s", response.Code, response.Body)
	}
}

func TestUserRoutesDelete(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	tokens := mocks.NewMockTokenService(controller)
	users.EXPECT().DeleteUser(gomock.Any(), validUserID).Return(nil)
	response := httptest.NewRecorder()
	userHandler(t, users, tokens).ServeHTTP(response, authenticatedRequest(http.MethodDelete, "/users/"+validUserID, ""))
	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Errorf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestUserRoutesRequireAuthentication(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	tokens := mocks.NewMockTokenService(controller)
	mux := http.NewServeMux()
	NewUserRoutes(handlers.NewUserHandler(users), tokens).Register(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/users", nil))
	assertAPIError(t, response, http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestUserRouteErrors(t *testing.T) {
	validation := validator.New()
	validationError := validation.Struct(struct {
		Name string `validate:"required"`
	}{})
	tests := []struct {
		name       string
		method     string
		target     string
		body       string
		operation  string
		err        error
		wantStatus int
		wantCode   string
	}{
		{name: "get invalid ID", method: http.MethodGet, target: "/users/invalid", operation: "get", err: domain.ErrInvalidUserID, wantStatus: 400, wantCode: "INVALID_USER_ID"},
		{name: "get not found", method: http.MethodGet, target: "/users/" + validUserID, operation: "get", err: domain.ErrUserNotFound, wantStatus: 404, wantCode: "USER_NOT_FOUND"},
		{name: "list internal", method: http.MethodGet, target: "/users", operation: "list", err: errors.New("database URI secret"), wantStatus: 500, wantCode: "INTERNAL_ERROR"},
		{name: "update malformed", method: http.MethodPatch, target: "/users/" + validUserID, body: `{"name":`, operation: "decode", wantStatus: 400, wantCode: "INVALID_REQUEST"},
		{name: "update validation", method: http.MethodPatch, target: "/users/" + validUserID, body: `{}`, operation: "update", err: validationError, wantStatus: 400, wantCode: "VALIDATION_ERROR"},
		{name: "update duplicate", method: http.MethodPatch, target: "/users/" + validUserID, body: `{"email":"ada@example.com"}`, operation: "update", err: domain.ErrEmailAlreadyExists, wantStatus: 409, wantCode: "EMAIL_ALREADY_EXISTS"},
		{name: "delete not found", method: http.MethodDelete, target: "/users/" + validUserID, operation: "delete", err: domain.ErrUserNotFound, wantStatus: 404, wantCode: "USER_NOT_FOUND"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			users := mocks.NewMockUserUseCase(controller)
			tokens := mocks.NewMockTokenService(controller)
			switch test.operation {
			case "get":
				users.EXPECT().GetUser(gomock.Any(), strings.TrimPrefix(test.target, "/users/")).Return(applicationdto.UserResponse{}, test.err)
			case "list":
				users.EXPECT().ListUsers(gomock.Any()).Return(nil, test.err)
			case "update":
				users.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(applicationdto.UserResponse{}, test.err)
			case "delete":
				users.EXPECT().DeleteUser(gomock.Any(), validUserID).Return(test.err)
			}
			response := httptest.NewRecorder()
			userHandler(t, users, tokens).ServeHTTP(response, authenticatedRequest(test.method, test.target, test.body))
			assertAPIError(t, response, test.wantStatus, test.wantCode)
			if strings.Contains(response.Body.String(), "database URI") {
				t.Errorf("response leaks internal error: %s", response.Body)
			}
		})
	}
}

func assertAPIError(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body)
	}
	var body httpdto.ErrorResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != code || body.Message == "" {
		t.Errorf("error response = %#v", body)
	}
}
