package routes

import (
	"net/http"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/httpx"
	applicationdto "github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type AuthRoutes struct {
	registration   ports.RegistrationUseCase
	authentication ports.AuthenticationUseCase
}

func NewAuthRoutes(registration ports.RegistrationUseCase, authentication ports.AuthenticationUseCase) *AuthRoutes {
	return &AuthRoutes{registration: registration, authentication: authentication}
}

func (routes *AuthRoutes) Register(mux *http.ServeMux) {
	auth := http.NewServeMux()
	auth.HandleFunc("POST /register", routes.registerUser)
	auth.HandleFunc("POST /login", routes.login)
	mux.Handle("/auth/", http.StripPrefix("/auth", auth))
}

func (routes *AuthRoutes) registerUser(w http.ResponseWriter, r *http.Request) {
	var request httpdto.RegistrationRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpdto.ErrorResponse{Error: "invalid request body"})
		return
	}

	user, err := routes.registration.Register(r.Context(), applicationdto.RegistrationInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		httpx.WriteApplicationError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, httpdto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
	})
}

func (routes *AuthRoutes) login(w http.ResponseWriter, r *http.Request) {
	var request httpdto.LoginRequest
	if err := httpx.DecodeJSON(w, r, &request); err != nil {
		httpx.WriteJSON(w, http.StatusBadRequest, httpdto.ErrorResponse{Error: "invalid request body"})
		return
	}

	result, err := routes.authentication.Authenticate(r.Context(), applicationdto.LoginInput{
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
