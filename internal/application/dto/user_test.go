package dto

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/go-playground/validator/v10"
)

func TestRegistrationInputValidatePreservesEmail(t *testing.T) {
	input := RegistrationInput{
		Name:     "  Ada Lovelace  ",
		Email:    "ADA+TEST@Example.COM",
		Password: " password with spaces ",
	}

	if err := input.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if input.Name != "Ada Lovelace" {
		t.Errorf("Name = %q, want %q", input.Name, "Ada Lovelace")
	}
	if input.Email != "ADA+TEST@Example.COM" {
		t.Errorf("Email = %q, want %q", input.Email, "ADA+TEST@Example.COM")
	}
	if input.Password != " password with spaces " {
		t.Error("Password was modified during validation")
	}
}

func TestRegistrationInputValidateAcceptsValidEmails(t *testing.T) {
	emails := []string{
		"user@example.com",
		"USER+tag@Example.COM",
		"first.last@sub.example.com",
	}

	for _, email := range emails {
		t.Run(email, func(t *testing.T) {
			input := RegistrationInput{Name: "Valid User", Email: email, Password: "password"}
			if err := input.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestRegistrationInputValidateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		input     RegistrationInput
		wantField string
		wantCode  string
	}{
		{name: "missing name", input: RegistrationInput{Email: "user@example.com", Password: "password"}, wantField: "name", wantCode: "required"},
		{name: "whitespace name", input: RegistrationInput{Name: "  ", Email: "user@example.com", Password: "password"}, wantField: "name", wantCode: "required"},
		{name: "missing email", input: RegistrationInput{Name: "Ada", Password: "password"}, wantField: "email", wantCode: "required"},
		{name: "invalid email", input: RegistrationInput{Name: "Ada", Email: "not-an-email", Password: "password"}, wantField: "email", wantCode: "email"},
		{name: "email with spaces", input: RegistrationInput{Name: "Ada", Email: " ada@example.com ", Password: "password"}, wantField: "email", wantCode: "email"},
		{name: "email display name", input: RegistrationInput{Name: "Ada", Email: "Ada <ada@example.com>", Password: "password"}, wantField: "email", wantCode: "email"},
		{name: "missing password", input: RegistrationInput{Name: "Ada", Email: "ada@example.com"}, wantField: "password", wantCode: "required"},
		{name: "whitespace password", input: RegistrationInput{Name: "Ada", Email: "ada@example.com", Password: "  "}, wantField: "password", wantCode: "required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.input.Validate()
			assertValidationError(t, err, test.wantField, test.wantCode)
		})
	}
}

func TestUserIDInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{name: "valid", id: "507f1f77bcf86cd799439011"},
		{name: "invalid length", id: "invalid", wantErr: true},
		{name: "invalid hexadecimal", id: "zzzzzzzzzzzzzzzzzzzzzzzz", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := (UserIDInput{ID: test.id}).Validate()
			var validationErrors validator.ValidationErrors
			if test.wantErr && !errors.As(err, &validationErrors) {
				t.Errorf("Validate() error = %v, want validation error", err)
			}
			if !test.wantErr && err != nil {
				t.Errorf("Validate() error = %v", err)
			}
		})
	}
}

func TestUpdateUserInputValidate(t *testing.T) {
	name := "Grace"
	email := "grace@example.com"
	empty := ""
	tests := []struct {
		name    string
		input   UpdateUserInput
		wantErr bool
	}{
		{name: "name", input: UpdateUserInput{ID: "507f1f77bcf86cd799439011", Name: &name}},
		{name: "email", input: UpdateUserInput{ID: "507f1f77bcf86cd799439011", Email: &email}},
		{name: "both", input: UpdateUserInput{ID: "507f1f77bcf86cd799439011", Name: &name, Email: &email}},
		{name: "missing ID", input: UpdateUserInput{Name: &name}, wantErr: true},
		{name: "invalid ID", input: UpdateUserInput{ID: "invalid", Name: &name}, wantErr: true},
		{name: "missing fields", input: UpdateUserInput{ID: "507f1f77bcf86cd799439011"}, wantErr: true},
		{name: "empty name", input: UpdateUserInput{ID: "507f1f77bcf86cd799439011", Name: &empty}, wantErr: true},
		{name: "invalid email", input: UpdateUserInput{ID: "507f1f77bcf86cd799439011", Email: &name}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.input.Validate()
			var validationErrors validator.ValidationErrors
			if test.wantErr && !errors.As(err, &validationErrors) {
				t.Errorf("Validate() error = %v, want validation error", err)
			}
			if !test.wantErr && err != nil {
				t.Errorf("Validate() error = %v", err)
			}
		})
	}
}

func TestNewUserResult(t *testing.T) {
	createdAt := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)
	user := domain.User{
		ID: "507f1f77bcf86cd799439011", Name: "Ada", Email: "ada@example.com", PasswordHash: "secret", CreatedAt: createdAt,
	}
	result := NewUserResult(user)
	if result.ID != user.ID || result.Name != user.Name || result.Email != user.Email || !result.CreatedAt.Equal(createdAt) {
		t.Errorf("NewUserResult() = %#v", result)
	}
}

func assertValidationError(t *testing.T, err error, field, code string) {
	t.Helper()
	if err == nil {
		t.Fatal("validation error = nil")
	}
	var validationErrors validator.ValidationErrors
	if !errors.As(err, &validationErrors) || len(validationErrors) == 0 {
		t.Fatalf("error type = %T, want validator.ValidationErrors", err)
	}
	validationError := validationErrors[0]
	if got := strings.ToLower(validationError.Field()); got != field {
		t.Errorf("field = %q, want %q", got, field)
	}
	if validationError.Tag() != code {
		t.Errorf("tag = %q, want %q", validationError.Tag(), code)
	}
}
