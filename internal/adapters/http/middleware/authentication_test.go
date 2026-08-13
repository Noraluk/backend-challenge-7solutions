package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Noraluk/backend-challenge-7solutions/internal/domain"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"go.uber.org/mock/gomock"
)

func TestRequireAuthentication(t *testing.T) {
	tests := []struct {
		name          string
		header        string
		claims        ports.TokenClaims
		validateError error
		wantStatus    int
		wantCalled    bool
	}{
		{name: "missing header", wantStatus: http.StatusUnauthorized},
		{name: "wrong scheme", header: "Basic token", wantStatus: http.StatusUnauthorized},
		{name: "malformed bearer", header: "Bearer", wantStatus: http.StatusUnauthorized},
		{name: "invalid token", header: "Bearer invalid", validateError: domain.ErrInvalidToken, wantStatus: http.StatusUnauthorized},
		{name: "expired token", header: "Bearer expired", validateError: domain.ErrInvalidToken, wantStatus: http.StatusUnauthorized},
		{name: "empty subject", header: "Bearer token", claims: ports.TokenClaims{}, wantStatus: http.StatusUnauthorized},
		{name: "valid token", header: "Bearer valid", claims: ports.TokenClaims{UserID: "user-id"}, wantStatus: http.StatusNoContent, wantCalled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			controller := gomock.NewController(t)
			tokens := mocks.NewMockTokenService(controller)
			called := false
			if test.header != "" && test.header != "Basic token" && test.header != "Bearer" {
				token := test.header[len("Bearer "):]
				tokens.EXPECT().Validate(token).Return(test.claims, test.validateError)
			}
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				userID, ok := AuthenticatedUserID(r.Context())
				if !ok || userID != "user-id" {
					t.Errorf("authenticated user = %q, %v", userID, ok)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			handler := RequireAuthentication(tokens)(next)
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.Header.Set("Authorization", test.header)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if called != test.wantCalled {
				t.Errorf("next called = %v, want %v", called, test.wantCalled)
			}
		})
	}
}
