package httpapi

import (
	"context"
	"net/http"
	"strings"

	httpdto "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/httpx"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type authenticatedUserIDKey struct{}

func RequireAuthentication(tokens ports.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				httpx.WriteJSON(w, http.StatusUnauthorized, httpdto.ErrorResponse{Error: "unauthorized"})
				return
			}

			claims, err := tokens.Validate(parts[1])
			if err != nil || claims.UserID == "" {
				httpx.WriteJSON(w, http.StatusUnauthorized, httpdto.ErrorResponse{Error: "unauthorized"})
				return
			}

			ctx := context.WithValue(r.Context(), authenticatedUserIDKey{}, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthenticatedUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(authenticatedUserIDKey{}).(string)
	return userID, ok && userID != ""
}
