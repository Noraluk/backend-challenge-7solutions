package handlers

import (
	"net/http"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	httperrors "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/errors"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/httpx"
	applicationdto "github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type AuthHandler struct {
	registration   ports.RegistrationUseCase
	authentication ports.AuthenticationUseCase
}

func NewAuthHandler(registration ports.RegistrationUseCase, authentication ports.AuthenticationUseCase) *AuthHandler {
	return &AuthHandler{registration: registration, authentication: authentication}
}

func (handler *AuthHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var request httpdto.RegistrationRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httperrors.CodeInvalidRequest, httperrors.MessageInvalidRequest)
		return
	}

	user, err := handler.registration.Register(r.Context(), applicationdto.RegistrationInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		httpx.WriteApplicationError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, user)
}

func (handler *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var request httpdto.LoginRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httperrors.CodeInvalidRequest, httperrors.MessageInvalidRequest)
		return
	}

	result, err := handler.authentication.Login(r.Context(), applicationdto.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		httpx.WriteApplicationError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, httpdto.LoginResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(result.ExpiresIn.Seconds()),
	})
}
