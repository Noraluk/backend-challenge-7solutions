package application

import (
	"errors"
	"strings"
	"testing"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/go-playground/validator/v10"
)

func TestValidateRegistrationNormalizesInput(t *testing.T) {
	input := dto.RegistrationInput{
		Name:     "  Ada Lovelace  ",
		Email:    "  ADA+TEST@Example.COM  ",
		Password: " password with spaces ",
	}

	got, err := ValidateRegistration(input)
	if err != nil {
		t.Fatalf("ValidateRegistration() error = %v", err)
	}

	if got.Name != "Ada Lovelace" {
		t.Errorf("Name = %q, want %q", got.Name, "Ada Lovelace")
	}
	if got.Email != "ada+test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "ada+test@example.com")
	}
	if got.Password != input.Password {
		t.Error("Password was modified during normalization")
	}
}

func TestValidateRegistrationAcceptsValidEmails(t *testing.T) {
	tests := []struct {
		email string
		want  string
	}{
		{email: "user@example.com", want: "user@example.com"},
		{email: "USER+tag@Example.COM", want: "user+tag@example.com"},
		{email: " first.last@sub.example.com ", want: "first.last@sub.example.com"},
	}

	for _, test := range tests {
		t.Run(test.email, func(t *testing.T) {
			got, err := ValidateRegistration(dto.RegistrationInput{
				Name:     "Valid User",
				Email:    test.email,
				Password: "password",
			})
			if err != nil {
				t.Fatalf("ValidateRegistration() error = %v", err)
			}
			if got.Email != test.want {
				t.Errorf("Email = %q, want %q", got.Email, test.want)
			}
		})
	}
}

func TestValidateRegistrationRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		input     dto.RegistrationInput
		wantField string
		wantCode  string
	}{
		{
			name:      "missing name",
			input:     dto.RegistrationInput{Email: "user@example.com", Password: "password"},
			wantField: "name",
			wantCode:  "required",
		},
		{
			name:      "whitespace name",
			input:     dto.RegistrationInput{Name: "  ", Email: "user@example.com", Password: "password"},
			wantField: "name",
			wantCode:  "required",
		},
		{
			name:      "missing email",
			input:     dto.RegistrationInput{Name: "Ada", Password: "password"},
			wantField: "email",
			wantCode:  "required",
		},
		{
			name:      "invalid email",
			input:     dto.RegistrationInput{Name: "Ada", Email: "not-an-email", Password: "password"},
			wantField: "email",
			wantCode:  "email",
		},
		{
			name:      "email display name",
			input:     dto.RegistrationInput{Name: "Ada", Email: "Ada <ada@example.com>", Password: "password"},
			wantField: "email",
			wantCode:  "email",
		},
		{
			name:      "missing password",
			input:     dto.RegistrationInput{Name: "Ada", Email: "ada@example.com"},
			wantField: "password",
			wantCode:  "required",
		},
		{
			name:      "whitespace password",
			input:     dto.RegistrationInput{Name: "Ada", Email: "ada@example.com", Password: "  "},
			wantField: "password",
			wantCode:  "required",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateRegistration(test.input)
			assertValidationError(t, err, test.wantField, test.wantCode)
		})
	}
}

func TestValidateUserUpdate(t *testing.T) {
	input, err := ValidateUserUpdate(map[string]string{
		"name":  "  Grace Hopper ",
		"email": " GRACE@Example.COM ",
	})
	if err != nil {
		t.Fatalf("ValidateUserUpdate() error = %v", err)
	}

	if input.Name == nil || *input.Name != "Grace Hopper" {
		t.Errorf("Name = %v, want %q", input.Name, "Grace Hopper")
	}
	if input.Email == nil || *input.Email != "grace@example.com" {
		t.Errorf("Email = %v, want %q", input.Email, "grace@example.com")
	}
}

func TestValidateUserUpdateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name      string
		fields    map[string]string
		wantField string
		wantCode  string
	}{
		{name: "empty update", fields: map[string]string{}, wantField: "fields", wantCode: "min"},
		{name: "unsupported field", fields: map[string]string{"password": "new-password"}, wantField: "fields[password]", wantCode: "oneof"},
		{name: "empty name", fields: map[string]string{"name": "  "}, wantField: "name", wantCode: "min"},
		{name: "empty email", fields: map[string]string{"email": "  "}, wantField: "email", wantCode: "min"},
		{name: "invalid email", fields: map[string]string{"email": "invalid"}, wantField: "email", wantCode: "email"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateUserUpdate(test.fields)
			assertValidationError(t, err, test.wantField, test.wantCode)
		})
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
