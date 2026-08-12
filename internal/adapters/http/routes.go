package httpapi

import (
	"fmt"
	"net/http"
)

type RouteGroup interface {
	Register(*http.ServeMux)
}

func NewHandler(groups ...RouteGroup) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", health)
	for _, group := range groups {
		group.Register(mux)
	}

	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}
