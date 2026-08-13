package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	httperrors "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/errors"
	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
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
	case errors.As(err, &validationErrors) && validationErrors[0].Field() == "ID":
		WriteError(w, http.StatusBadRequest, httperrors.CodeInvalidUserID, httperrors.MessageInvalidUserID)
	case errors.As(err, &validationErrors):
		WriteError(w, http.StatusBadRequest, httperrors.CodeValidation, validationMessage(validationErrors[0]))
	case errors.Is(err, domain.ErrInvalidUserID):
		WriteError(w, http.StatusBadRequest, httperrors.CodeInvalidUserID, httperrors.MessageInvalidUserID)
	case errors.Is(err, domain.ErrEmailAlreadyExists):
		WriteError(w, http.StatusConflict, httperrors.CodeEmailAlreadyExists, httperrors.MessageEmailAlreadyExists)
	case errors.Is(err, domain.ErrInvalidCredentials):
		WriteError(w, http.StatusUnauthorized, httperrors.CodeInvalidCredentials, httperrors.MessageInvalidCredentials)
	case errors.Is(err, domain.ErrUserNotFound):
		WriteError(w, http.StatusNotFound, httperrors.CodeUserNotFound, httperrors.MessageUserNotFound)
	default:
		log.Printf("internal HTTP error: %v", err)
		WriteError(w, http.StatusInternalServerError, httperrors.CodeInternal, httperrors.MessageInternal)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, httpdto.ErrorResponse{Code: code, Message: message})
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
