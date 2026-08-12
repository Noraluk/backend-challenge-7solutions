package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type testRouteGroup struct {
	registered bool
}

func (group *testRouteGroup) Register(mux *http.ServeMux) {
	group.registered = true
	mux.HandleFunc("GET /test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestNewHandlerRegistersHealthAndRouteGroups(t *testing.T) {
	group := &testRouteGroup{}
	handler := NewHandler(group)

	healthResponse := httptest.NewRecorder()
	handler.ServeHTTP(healthResponse, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthResponse.Code != http.StatusOK || healthResponse.Body.String() != "ok\n" {
		t.Errorf("health response = %d %q", healthResponse.Code, healthResponse.Body.String())
	}
	if contentType := healthResponse.Header().Get("Content-Type"); contentType != "text/plain; charset=utf-8" {
		t.Errorf("health Content-Type = %q", contentType)
	}

	groupResponse := httptest.NewRecorder()
	handler.ServeHTTP(groupResponse, httptest.NewRequest(http.MethodGet, "/test", nil))
	if !group.registered || groupResponse.Code != http.StatusNoContent {
		t.Errorf("group registered = %v, status = %d", group.registered, groupResponse.Code)
	}
}
