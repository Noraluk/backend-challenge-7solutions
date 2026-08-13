package grpcapi

import (
	"context"
	"strings"

	userv1 "github.com/Noraluk/backend-challenge-7solutions/gen/user/v1"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type authenticatedUserIDKey struct{}

func RequireAuthentication(tokens ports.TokenService) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == userv1.UserService_CreateUser_FullMethodName {
			return handler(ctx, request)
		}

		values := metadata.ValueFromIncomingContext(ctx, "authorization")
		if len(values) != 1 {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		parts := strings.Fields(values[0])
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}
		claims, err := tokens.Validate(parts[1])
		if err != nil || claims.UserID == "" {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}
		return handler(context.WithValue(ctx, authenticatedUserIDKey{}, claims.UserID), request)
	}
}

func AuthenticatedUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(authenticatedUserIDKey{}).(string)
	return userID, ok && userID != ""
}
