package grpcapi

import (
	"context"

	userv1 "github.com/Noraluk/backend-challenge-7solutions/gen/user/v1"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

type UserServer struct {
	userv1.UnimplementedUserServiceServer
	registration ports.RegistrationUseCase
	users        ports.UserUseCase
}

func NewUserServer(registration ports.RegistrationUseCase, users ports.UserUseCase) *UserServer {
	return &UserServer{registration: registration, users: users}
}

func (s *UserServer) CreateUser(ctx context.Context, request *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	user, err := s.registration.Register(ctx, dto.RegistrationInput{
		Name:     request.GetName(),
		Email:    request.GetEmail(),
		Password: request.GetPassword(),
	})
	if err != nil {
		return nil, err
	}
	return &userv1.CreateUserResponse{User: user.ToProto()}, nil
}

func (s *UserServer) GetUser(ctx context.Context, request *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	user, err := s.users.GetUser(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	return &userv1.GetUserResponse{User: user.ToProto()}, nil
}
