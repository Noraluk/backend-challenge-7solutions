package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLogRequests(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	nextCalls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalls++
		w.Header().Set("X-Test", "preserved")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("response"))
	})
	handler := LogRequests(logger)(next)
	request := httptest.NewRequest(http.MethodPost, "/users?role=admin", strings.NewReader(`{"password":"secret"}`))
	request.Header.Set("Authorization", "Bearer sensitive-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if nextCalls != 1 {
		t.Errorf("next handler calls = %d, want 1", nextCalls)
	}
	if response.Code != http.StatusCreated || response.Body.String() != "response" || response.Header().Get("X-Test") != "preserved" {
		t.Errorf("response = status %d, header %q, body %q", response.Code, response.Header().Get("X-Test"), response.Body.String())
	}
	logEntry := output.String()
	for _, field := range []string{`msg="HTTP request"`, "method=POST", "path=/users", "elapsed="} {
		if !strings.Contains(logEntry, field) {
			t.Errorf("log entry %q does not contain %q", logEntry, field)
		}
	}
	if strings.Count(logEntry, `msg="HTTP request"`) != 1 {
		t.Errorf("request log count = %d, want 1", strings.Count(logEntry, `msg="HTTP request"`))
	}
	for _, sensitiveValue := range []string{"sensitive-token", "secret", "Authorization"} {
		if strings.Contains(logEntry, sensitiveValue) {
			t.Errorf("log entry contains sensitive value %q", sensitiveValue)
		}
	}
}
