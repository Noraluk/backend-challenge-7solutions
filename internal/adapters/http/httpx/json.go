package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application"
	"github.com/go-playground/validator/v10"
)

const maximumRequestBodySize = 1 << 20

func DecodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maximumRequestBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("request body must contain one JSON object")
		}
		return err
	}

	return nil
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func WriteApplicationError(w http.ResponseWriter, err error) {
	var validationErrors validator.ValidationErrors
	switch {
	case errors.As(err, &validationErrors):
		WriteJSON(w, http.StatusBadRequest, httpdto.ErrorResponse{Error: validationMessage(validationErrors[0])})
	case errors.Is(err, application.ErrEmailAlreadyExists):
		WriteJSON(w, http.StatusConflict, httpdto.ErrorResponse{Error: "email already exists"})
	case errors.Is(err, application.ErrInvalidCredentials):
		WriteJSON(w, http.StatusUnauthorized, httpdto.ErrorResponse{Error: "invalid credentials"})
	default:
		WriteJSON(w, http.StatusInternalServerError, httpdto.ErrorResponse{Error: "internal server error"})
	}
}

func validationMessage(err validator.FieldError) string {
	field := strings.ToLower(err.Field())
	if strings.HasPrefix(field, "fields") {
		field = "update"
	}
	if err.Tag() == "required" || err.Tag() == "min" {
		return field + " is required"
	}
	if err.Tag() == "oneof" {
		return "update contains an unsupported field"
	}
	return field + " is invalid"
}
