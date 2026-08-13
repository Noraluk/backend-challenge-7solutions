package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/testutil"
	"github.com/go-playground/validator/v10"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{name: "valid", body: `{"name":"Ada"}`},
		{name: "malformed", body: `{"name":`, wantErr: true},
		{name: "unknown field", body: `{"name":"Ada","role":"admin"}`, wantErr: true},
		{name: "multiple values", body: `{"name":"Ada"} {}`, wantErr: true},
		{name: "trailing invalid value", body: `{"name":"Ada"} x`, wantErr: true},
		{name: "too large", body: `{"name":"` + strings.Repeat("a", maximumRequestBodySize) + `"}`, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			var destination struct {
				Name string `json:"name"`
			}

			err := DecodeJSON(response, request, &destination)
			if (err != nil) != test.wantErr {
				t.Errorf("DecodeJSON() error = %v, wantErr %v", err, test.wantErr)
			}
			if !test.wantErr && destination.Name != "Ada" {
				t.Errorf("decoded name = %q", destination.Name)
			}
		})
	}
}

func TestWriteJSON(t *testing.T) {
	response := httptest.NewRecorder()
	WriteJSON(response, http.StatusCreated, map[string]string{"status": "created"})

	if response.Code != http.StatusCreated {
		t.Errorf("status = %d", response.Code)
	}
	testutil.AssertJSONContentType(t, response)
	if response.Body.String() != "{\"status\":\"created\"}\n" {
		t.Errorf("body = %q", response.Body.String())
	}
}

func TestWriteApplicationError(t *testing.T) {
	validate := validator.New()
	validationError := func(value any) error {
		err := validate.Struct(value)
		if err == nil {
			t.Fatal("expected validation error")
		}
		return err
	}

	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{name: "required", err: validationError(struct {
			Name string `validate:"required"`
		}{}), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantMessage: "name is required"},
		{name: "invalid", err: validationError(struct {
			Email string `validate:"email"`
		}{Email: "invalid"}), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantMessage: "email is invalid"},
		{name: "unsupported update", err: validationError(struct {
			Fields map[string]string `validate:"dive,keys,oneof=name email,endkeys"`
		}{Fields: map[string]string{"password": "value"}}), wantStatus: http.StatusBadRequest, wantCode: "VALIDATION_ERROR", wantMessage: "update contains an unsupported field"},
		{name: "invalid ID input", err: validationError(struct {
			ID string `validate:"len=24,hexadecimal"`
		}{ID: "invalid"}), wantStatus: http.StatusBadRequest, wantCode: "INVALID_USER_ID", wantMessage: "user ID is invalid"},
		{name: "invalid ID", err: domain.ErrInvalidUserID, wantStatus: http.StatusBadRequest, wantCode: "INVALID_USER_ID", wantMessage: "user ID is invalid"},
		{name: "duplicate", err: domain.ErrEmailAlreadyExists, wantStatus: http.StatusConflict, wantCode: "EMAIL_ALREADY_EXISTS", wantMessage: "email already exists"},
		{name: "credentials", err: domain.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantCode: "INVALID_CREDENTIALS", wantMessage: "invalid credentials"},
		{name: "not found", err: domain.ErrUserNotFound, wantStatus: http.StatusNotFound, wantCode: "USER_NOT_FOUND", wantMessage: "user not found"},
		{name: "internal", err: errors.New("database details"), wantStatus: http.StatusInternalServerError, wantCode: "INTERNAL_ERROR", wantMessage: "internal server error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			WriteApplicationError(response, test.err)
			var body httpdto.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.Code != test.wantStatus || body.Code != test.wantCode || body.Message != test.wantMessage {
				t.Errorf("response = %d %#v", response.Code, body)
			}
		})
	}
}
