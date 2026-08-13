package grpcapi

import (
	"context"
	"errors"
	"testing"

	userv1 "github.com/Noraluk/backend-challenge-7solutions/gen/user/v1"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestRequireAuthenticationAllowsCreateUser(t *testing.T) {
	tokens := mocks.NewMockTokenService(gomock.NewController(t))
	called := false
	_, err := RequireAuthentication(tokens)(context.Background(), nil, &grpc.UnaryServerInfo{
		FullMethod: userv1.UserService_CreateUser_FullMethodName,
	}, func(context.Context, any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil || !called {
		t.Fatalf("public interceptor error = %v, called = %t", err, called)
	}
}

func TestRequireAuthenticationRejectsMissingOrMalformedMetadata(t *testing.T) {
	tests := []struct {
		name    string
		context context.Context
	}{
		{name: "missing", context: context.Background()},
		{name: "wrong scheme", context: incomingAuthorization("Basic abc")},
		{name: "missing token", context: incomingAuthorization("Bearer")},
		{name: "multiple values", context: metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer one", "authorization", "Bearer two"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			_, err := RequireAuthentication(mocks.NewMockTokenService(gomock.NewController(t)))(test.context, nil, &grpc.UnaryServerInfo{
				FullMethod: userv1.UserService_GetUser_FullMethodName,
			}, func(context.Context, any) (any, error) {
				called = true
				return nil, nil
			})
			if status.Code(err) != codes.Unauthenticated || called {
				t.Fatalf("interceptor code = %s, called = %t", status.Code(err), called)
			}
		})
	}
}

func TestRequireAuthenticationRejectsInvalidToken(t *testing.T) {
	tokens := mocks.NewMockTokenService(gomock.NewController(t))
	tokens.EXPECT().Validate("invalid").Return(ports.TokenClaims{}, errors.New("invalid token"))
	_, err := RequireAuthentication(tokens)(incomingAuthorization("Bearer invalid"), nil, &grpc.UnaryServerInfo{
		FullMethod: userv1.UserService_GetUser_FullMethodName,
	}, func(context.Context, any) (any, error) {
		t.Fatal("handler called for invalid token")
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("interceptor code = %s, want %s", status.Code(err), codes.Unauthenticated)
	}
}

func TestRequireAuthenticationInjectsValidSubject(t *testing.T) {
	tokens := mocks.NewMockTokenService(gomock.NewController(t))
	tokens.EXPECT().Validate("valid").Return(ports.TokenClaims{UserID: "user-123"}, nil)
	_, err := RequireAuthentication(tokens)(incomingAuthorization("bearer valid"), nil, &grpc.UnaryServerInfo{
		FullMethod: userv1.UserService_GetUser_FullMethodName,
	}, func(ctx context.Context, _ any) (any, error) {
		userID, ok := AuthenticatedUserID(ctx)
		if !ok || userID != "user-123" {
			t.Fatalf("authenticated subject = %q, %t", userID, ok)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor error = %v", err)
	}
}

func TestAuthenticatedUserIDWithoutSubject(t *testing.T) {
	if userID, ok := AuthenticatedUserID(context.Background()); ok || userID != "" {
		t.Fatalf("AuthenticatedUserID() = %q, %t", userID, ok)
	}
}

func incomingAuthorization(value string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", value))
}
