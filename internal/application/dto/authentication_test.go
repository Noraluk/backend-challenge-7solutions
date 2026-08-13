package dto

import (
	"errors"
	"testing"

	"github.com/go-playground/validator/v10"
)

func TestLoginInputValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   LoginInput
		wantErr bool
	}{
		{name: "valid", input: LoginInput{Email: "user@example.com", Password: "password"}},
		{name: "missing email", input: LoginInput{Password: "password"}, wantErr: true},
		{name: "whitespace email", input: LoginInput{Email: "  ", Password: "password"}, wantErr: true},
		{name: "missing password", input: LoginInput{Email: "user@example.com"}, wantErr: true},
		{name: "whitespace password", input: LoginInput{Email: "user@example.com", Password: "  "}, wantErr: true},
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
