package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationdto "github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"go.uber.org/mock/gomock"
)

const handlerUserID = "507f1f77bcf86cd799439011"

func TestNewUserHandler(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	handler := NewUserHandler(users)
	if handler.users != users {
		t.Errorf("NewUserHandler() = %#v", handler)
	}
}

func TestUserHandlerGetUser(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	want := applicationdto.UserResponse{ID: handlerUserID, Name: "Ada"}
	users.EXPECT().GetUser(gomock.Any(), handlerUserID).Return(want, nil)
	request := httptest.NewRequest(http.MethodGet, "/users/"+handlerUserID, nil)
	request.SetPathValue("id", handlerUserID)
	response := httptest.NewRecorder()

	NewUserHandler(users).GetUser(response, request)

	assertUserResponse(t, response, http.StatusOK, want)
}

func TestUserHandlerGetUserError(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	users.EXPECT().GetUser(gomock.Any(), handlerUserID).Return(applicationdto.UserResponse{}, domain.ErrUserNotFound)
	request := httptest.NewRequest(http.MethodGet, "/users/"+handlerUserID, nil)
	request.SetPathValue("id", handlerUserID)
	response := httptest.NewRecorder()

	NewUserHandler(users).GetUser(response, request)

	assertErrorResponse(t, response, http.StatusNotFound, "USER_NOT_FOUND")
}

func TestUserHandlerListUsers(t *testing.T) {
	tests := []struct {
		name       string
		users      []applicationdto.UserResponse
		err        error
		wantStatus int
	}{
		{name: "success", users: []applicationdto.UserResponse{{ID: handlerUserID}}, wantStatus: http.StatusOK},
		{name: "empty", users: []applicationdto.UserResponse{}, wantStatus: http.StatusOK},
		{name: "error", err: errors.New("database failed"), wantStatus: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			users := mocks.NewMockUserUseCase(controller)
			users.EXPECT().ListUsers(gomock.Any()).Return(test.users, test.err)
			response := httptest.NewRecorder()
			NewUserHandler(users).ListUsers(response, httptest.NewRequest(http.MethodGet, "/users", nil))
			if test.err != nil {
				assertErrorResponse(t, response, test.wantStatus, "INTERNAL_ERROR")
				return
			}
			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var got []applicationdto.UserResponse
			if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(got) != len(test.users) || got == nil {
				t.Errorf("response = %#v", got)
			}
		})
	}
}

func TestUserHandlerUpdateUser(t *testing.T) {
	controller := gomock.NewController(t)
	users := mocks.NewMockUserUseCase(controller)
	want := applicationdto.UserResponse{ID: handlerUserID, Name: "Grace"}
	users.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, input applicationdto.UpdateUserInput) (applicationdto.UserResponse, error) {
			if input.ID != handlerUserID || input.Name == nil || *input.Name != "Grace" || input.Email != nil {
				t.Errorf("update input = %#v", input)
			}
			return want, nil
		},
	)
	request := httptest.NewRequest(http.MethodPatch, "/users/"+handlerUserID, strings.NewReader(`{"name":"Grace"}`))
	request.SetPathValue("id", handlerUserID)
	response := httptest.NewRecorder()

	NewUserHandler(users).UpdateUser(response, request)

	assertUserResponse(t, response, http.StatusOK, want)
}

func TestUserHandlerUpdateUserErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		usecaseErr error
		wantStatus int
		wantCode   string
	}{
		{name: "invalid request", body: `{"name":`, wantStatus: http.StatusBadRequest, wantCode: "INVALID_REQUEST"},
		{name: "duplicate", body: `{"email":"ada@example.com"}`, usecaseErr: domain.ErrEmailAlreadyExists, wantStatus: http.StatusConflict, wantCode: "EMAIL_ALREADY_EXISTS"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			users := mocks.NewMockUserUseCase(controller)
			if test.usecaseErr != nil {
				users.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Return(applicationdto.UserResponse{}, test.usecaseErr)
			}
			request := httptest.NewRequest(http.MethodPatch, "/users/"+handlerUserID, strings.NewReader(test.body))
			request.SetPathValue("id", handlerUserID)
			response := httptest.NewRecorder()
			NewUserHandler(users).UpdateUser(response, request)
			assertErrorResponse(t, response, test.wantStatus, test.wantCode)
		})
	}
}

func TestUserHandlerDeleteUser(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "success", wantStatus: http.StatusNoContent},
		{name: "not found", err: domain.ErrUserNotFound, wantStatus: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			users := mocks.NewMockUserUseCase(controller)
			users.EXPECT().DeleteUser(gomock.Any(), handlerUserID).Return(test.err)
			request := httptest.NewRequest(http.MethodDelete, "/users/"+handlerUserID, nil)
			request.SetPathValue("id", handlerUserID)
			response := httptest.NewRecorder()
			NewUserHandler(users).DeleteUser(response, request)
			if test.err != nil {
				assertErrorResponse(t, response, test.wantStatus, "USER_NOT_FOUND")
				return
			}
			if response.Code != test.wantStatus || response.Body.Len() != 0 {
				t.Errorf("response = %d %q", response.Code, response.Body.String())
			}
		})
	}
}

func assertUserResponse(t *testing.T, response *httptest.ResponseRecorder, status int, want applicationdto.UserResponse) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, status, response.Body)
	}
	var got applicationdto.UserResponse
	if err := json.NewDecoder(response.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got != want {
		t.Errorf("response = %#v, want %#v", got, want)
	}
}
