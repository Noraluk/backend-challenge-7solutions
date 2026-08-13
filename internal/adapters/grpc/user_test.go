package grpcapi

import (
	"context"
	"errors"
	"testing"
	"time"

	userv1 "github.com/Noraluk/backend-challenge-7solutions/gen/user/v1"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application/dto"
	"github.com/Noraluk/backend-challenge-7solutions/internal/mocks"
	"go.uber.org/mock/gomock"
)

func TestUserServerCreateUser(t *testing.T) {
	registration := mocks.NewMockRegistrationUseCase(gomock.NewController(t))
	users := mocks.NewMockUserUseCase(gomock.NewController(t))
	createdAt := time.Date(2026, time.August, 13, 10, 30, 0, 0, time.UTC)
	input := dto.RegistrationInput{Name: "Ada", Email: "ada@example.com", Password: "secret"}
	registration.EXPECT().Register(gomock.Any(), input).Return(dto.UserResponse{
		ID: "507f1f77bcf86cd799439011", Name: input.Name, Email: input.Email, CreatedAt: createdAt,
	}, nil)

	response, err := NewUserServer(registration, users).CreateUser(context.Background(), &userv1.CreateUserRequest{
		Name: input.Name, Email: input.Email, Password: input.Password,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if response.GetUser().GetId() != "507f1f77bcf86cd799439011" || response.GetUser().GetEmail() != input.Email {
		t.Fatalf("CreateUser() user = %v", response.GetUser())
	}
	if !response.GetUser().GetCreatedAt().AsTime().Equal(createdAt) {
		t.Errorf("created_at = %v, want %v", response.GetUser().GetCreatedAt(), createdAt)
	}
	if response.GetUser().ProtoReflect().Descriptor().Fields().ByName("password_hash") != nil {
		t.Error("protobuf User exposes password_hash")
	}
}

func TestUserServerCreateUserErrors(t *testing.T) {
	wantErr := errors.New("registration failed")
	registration := mocks.NewMockRegistrationUseCase(gomock.NewController(t))
	registration.EXPECT().Register(gomock.Any(), gomock.Any()).Return(dto.UserResponse{}, wantErr)
	_, err := NewUserServer(registration, mocks.NewMockUserUseCase(gomock.NewController(t))).CreateUser(
		context.Background(), &userv1.CreateUserRequest{},
	)
	if !errors.Is(err, wantErr) {
		t.Errorf("CreateUser() error = %v, want %v", err, wantErr)
	}
}

func TestUserServerGetUser(t *testing.T) {
	users := mocks.NewMockUserUseCase(gomock.NewController(t))
	const id = "507f1f77bcf86cd799439011"
	users.EXPECT().GetUser(gomock.Any(), id).Return(dto.UserResponse{ID: id, Name: "Ada", Email: "ada@example.com"}, nil)

	response, err := NewUserServer(mocks.NewMockRegistrationUseCase(gomock.NewController(t)), users).GetUser(
		context.Background(), &userv1.GetUserRequest{Id: id},
	)
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}
	if response.GetUser().GetId() != id {
		t.Errorf("GetUser() id = %q, want %q", response.GetUser().GetId(), id)
	}
}

func TestUserServerGetUserErrors(t *testing.T) {
	wantErr := errors.New("get user failed")
	users := mocks.NewMockUserUseCase(gomock.NewController(t))
	users.EXPECT().GetUser(gomock.Any(), gomock.Any()).Return(dto.UserResponse{}, wantErr)
	_, err := NewUserServer(mocks.NewMockRegistrationUseCase(gomock.NewController(t)), users).GetUser(
		context.Background(), &userv1.GetUserRequest{},
	)
	if !errors.Is(err, wantErr) {
		t.Errorf("GetUser() error = %v, want %v", err, wantErr)
	}
}
