package routes

import (
	"net/http"

	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/handlers"
)

type AuthRoutes struct {
	handler *handlers.AuthHandler
}

func NewAuthRoutes(handler *handlers.AuthHandler) *AuthRoutes {
	return &AuthRoutes{handler: handler}
}

func (routes *AuthRoutes) Register(mux *http.ServeMux) {
	auth := http.NewServeMux()
	auth.HandleFunc("POST /register", routes.handler.RegisterUser)
	auth.HandleFunc("POST /login", routes.handler.Login)
	mux.Handle("/auth/", http.StripPrefix("/auth", auth))
}
