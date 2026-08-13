package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/handlers"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/middleware"
)

type RouteGroup interface {
	Register(*http.ServeMux)
}

func NewHandler(groups ...RouteGroup) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health)
	for _, group := range groups {
		group.Register(mux)
	}

	return middleware.LogRequests(slog.Default())(mux)
}
