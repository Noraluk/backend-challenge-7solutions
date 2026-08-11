package application

import (
	"errors"
	"fmt"
	"net/mail"
	"sort"
	"strings"
)

const (
	ValidationRequired    = "required"
	ValidationInvalid     = "invalid"
	ValidationUnsupported = "unsupported"
	ValidationEmpty       = "empty"
)

var ErrValidation = errors.New("validation failed")

type ValidationError struct {
	Field string
	Code  string
}

func (e ValidationError) Error() string {
	switch e.Code {
	case ValidationRequired:
		return fmt.Sprintf("%s is required", e.Field)
	case ValidationUnsupported:
		return fmt.Sprintf("%s is not supported", e.Field)
	case ValidationEmpty:
		return "update must include at least one supported field"
	default:
		return fmt.Sprintf("%s is invalid", e.Field)
	}
}

func (e ValidationError) Unwrap() error {
	return ErrValidation
}

type RegistrationInput struct {
	Name     string
	Email    string
	Password string
}

type UpdateUserInput struct {
	Name  *string
	Email *string
}

func ValidateRegistration(input RegistrationInput) (RegistrationInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = normalizeEmail(input.Email)

	if input.Name == "" {
		return RegistrationInput{}, ValidationError{Field: "name", Code: ValidationRequired}
	}
	if input.Email == "" {
		return RegistrationInput{}, ValidationError{Field: "email", Code: ValidationRequired}
	}
	if !validEmail(input.Email) {
		return RegistrationInput{}, ValidationError{Field: "email", Code: ValidationInvalid}
	}
	if strings.TrimSpace(input.Password) == "" {
		return RegistrationInput{}, ValidationError{Field: "password", Code: ValidationRequired}
	}

	return input, nil
}

func ValidateUserUpdate(fields map[string]string) (UpdateUserInput, error) {
	if len(fields) == 0 {
		return UpdateUserInput{}, ValidationError{Field: "update", Code: ValidationEmpty}
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		if key != "name" && key != "email" {
			return UpdateUserInput{}, ValidationError{Field: key, Code: ValidationUnsupported}
		}
	}

	var input UpdateUserInput
	if value, ok := fields["name"]; ok {
		name := strings.TrimSpace(value)
		if name == "" {
			return UpdateUserInput{}, ValidationError{Field: "name", Code: ValidationRequired}
		}
		input.Name = &name
	}

	if value, ok := fields["email"]; ok {
		email := normalizeEmail(value)
		if email == "" {
			return UpdateUserInput{}, ValidationError{Field: "email", Code: ValidationRequired}
		}
		if !validEmail(email) {
			return UpdateUserInput{}, ValidationError{Field: "email", Code: ValidationInvalid}
		}
		input.Email = &email
	}

	return input, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	address, err := mail.ParseAddress(email)
	return err == nil && address.Address == email && strings.Contains(email, "@")
}
