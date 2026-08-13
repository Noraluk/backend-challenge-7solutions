package middleware

import (
	"context"
	"net/http"
	"strings"

	httperrors "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/errors"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/httpx"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type authenticatedUserIDKey struct{}

func RequireAuthentication(tokens ports.TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Fields(r.Header.Get("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				httpx.WriteError(w, http.StatusUnauthorized, httperrors.CodeUnauthorized, httperrors.MessageUnauthorized)
				return
			}

			claims, err := tokens.Validate(parts[1])
			if err != nil || claims.UserID == "" {
				httpx.WriteError(w, http.StatusUnauthorized, httperrors.CodeUnauthorized, httperrors.MessageUnauthorized)
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
