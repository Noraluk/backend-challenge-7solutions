package routes

import (
	"net/http"

	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/handlers"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/middleware"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type UserRoutes struct {
	handler *handlers.UserHandler
	tokens  ports.TokenService
}

func NewUserRoutes(handler *handlers.UserHandler, tokens ports.TokenService) *UserRoutes {
	return &UserRoutes{handler: handler, tokens: tokens}
}

func (routes *UserRoutes) Register(mux *http.ServeMux) {
	requireAuthentication := middleware.RequireAuthentication(routes.tokens)
	mux.Handle("GET /users", requireAuthentication(http.HandlerFunc(routes.handler.ListUsers)))
	mux.Handle("GET /users/{id}", requireAuthentication(http.HandlerFunc(routes.handler.GetUser)))
	mux.Handle("PATCH /users/{id}", requireAuthentication(http.HandlerFunc(routes.handler.UpdateUser)))
	mux.Handle("DELETE /users/{id}", requireAuthentication(http.HandlerFunc(routes.handler.DeleteUser)))
}
