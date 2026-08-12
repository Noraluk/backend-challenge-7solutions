package application

import (
	"strings"

	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

func ValidateRegistration(input dto.RegistrationInput) (dto.RegistrationInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = normalizeEmail(input.Email)
	validationInput := input
	validationInput.Password = strings.TrimSpace(validationInput.Password)

	if err := validate.Struct(validationInput); err != nil {
		return dto.RegistrationInput{}, err
	}

	return input, nil
}

func ValidateUserUpdate(fields map[string]string) (dto.UpdateUserInput, error) {
	validationFields := struct {
		Fields map[string]string `validate:"min=1,dive,keys,oneof=name email,endkeys"`
	}{Fields: fields}
	if err := validate.Struct(validationFields); err != nil {
		return dto.UpdateUserInput{}, err
	}

	var input dto.UpdateUserInput
	if value, ok := fields["name"]; ok {
		name := strings.TrimSpace(value)
		input.Name = &name
	}

	if value, ok := fields["email"]; ok {
		email := normalizeEmail(value)
		input.Email = &email
	}

	if err := validate.Struct(input); err != nil {
		return dto.UpdateUserInput{}, err
	}

	return input, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
