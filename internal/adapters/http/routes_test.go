package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type testRouteGroup struct {
	registered bool
	pattern    string
}

func (group *testRouteGroup) Register(mux *http.ServeMux) {
	group.registered = true
	mux.HandleFunc(group.pattern, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestNewHandlerRegistersHealthAndRouteGroups(t *testing.T) {
	group := &testRouteGroup{pattern: "GET /test"}
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

func TestNewHandlerWithoutRouteGroups(t *testing.T) {
	handler := NewHandler()

	healthResponse := httptest.NewRecorder()
	handler.ServeHTTP(healthResponse, httptest.NewRequest(http.MethodGet, "/health", nil))
	if healthResponse.Code != http.StatusOK {
		t.Errorf("health status = %d, want %d", healthResponse.Code, http.StatusOK)
	}

	notFoundResponse := httptest.NewRecorder()
	handler.ServeHTTP(notFoundResponse, httptest.NewRequest(http.MethodGet, "/unknown", nil))
	if notFoundResponse.Code != http.StatusNotFound {
		t.Errorf("unknown route status = %d, want %d", notFoundResponse.Code, http.StatusNotFound)
	}
}

func TestNewHandlerRegistersMultipleRouteGroups(t *testing.T) {
	groups := []*testRouteGroup{
		{pattern: "GET /first"},
		{pattern: "GET /second"},
	}
	handler := NewHandler(groups[0], groups[1])

	for index, group := range groups {
		if !group.registered {
			t.Errorf("group %d was not registered", index)
		}
	}
	for _, path := range []string{"/first", "/second"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusNoContent {
			t.Errorf("%s status = %d, want %d", path, response.Code, http.StatusNoContent)
		}
	}
}
