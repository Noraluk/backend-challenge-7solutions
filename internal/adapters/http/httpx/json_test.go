package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application"
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
	if response.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", response.Header().Get("Content-Type"))
	}
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
		name       string
		err        error
		wantStatus int
		wantBody   string
	}{
		{name: "required", err: validationError(struct {
			Name string `validate:"required"`
		}{}), wantStatus: http.StatusBadRequest, wantBody: "name is required"},
		{name: "invalid", err: validationError(struct {
			Email string `validate:"email"`
		}{Email: "invalid"}), wantStatus: http.StatusBadRequest, wantBody: "email is invalid"},
		{name: "unsupported update", err: validationError(struct {
			Fields map[string]string `validate:"dive,keys,oneof=name email,endkeys"`
		}{Fields: map[string]string{"password": "value"}}), wantStatus: http.StatusBadRequest, wantBody: "update contains an unsupported field"},
		{name: "duplicate", err: application.ErrEmailAlreadyExists, wantStatus: http.StatusConflict, wantBody: "email already exists"},
		{name: "credentials", err: application.ErrInvalidCredentials, wantStatus: http.StatusUnauthorized, wantBody: "invalid credentials"},
		{name: "internal", err: errors.New("database details"), wantStatus: http.StatusInternalServerError, wantBody: "internal server error"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			WriteApplicationError(response, test.err)
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), test.wantBody) {
				t.Errorf("response = %d %q", response.Code, response.Body.String())
			}
		})
	}
}
