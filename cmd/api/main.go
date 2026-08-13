package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	userv1 "github.com/Noraluk/backend-challenge-7solutions/gen/user/v1"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/auth"
	grpcapi "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/grpc"
	httpapi "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http"
	httphandlers "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/handlers"
	httproutes "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/routes"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/mongodb"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application"
	"github.com/Noraluk/backend-challenge-7solutions/internal/platform"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
	"google.golang.org/grpc"
)

const (
	shutdownTimeout   = 10 * time.Second
	disconnectTimeout = 10 * time.Second
)

type databaseConnection interface {
	UserRepository() ports.UserRepository
	Disconnect(context.Context) error
}

var loadConfiguration = platform.LoadConfig

var connectDatabase = func(ctx context.Context, uri, databaseName string) (databaseConnection, error) {
	connection, err := mongodb.Connect(ctx, uri, databaseName)
	if err != nil {
		return nil, err
	}
	return connection, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	if err := run(ctx); err != nil {
		stop()
		log.Print(err)
		os.Exit(1)
	}
	stop()
}

func run(ctx context.Context) error {
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	database, err := connectDatabase(ctx, config.MongoURI, config.MongoDatabase)
	if err != nil {
		return fmt.Errorf("MongoDB startup error: %w", err)
	}
	defer func() {
		disconnectContext, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
		defer cancel()
		if err := database.Disconnect(disconnectContext); err != nil {
			log.Printf("MongoDB shutdown error: %v", err)
		}
	}()

	repository := database.UserRepository()
	workerContext, stopWorker := context.WithCancel(ctx)
	workerStopped := make(chan struct{})
	go func() {
		application.RunUserCountWorker(workerContext, repository, slog.Default())
		close(workerStopped)
	}()
	defer func() {
		stopWorker()
		<-workerStopped
	}()

	passwordHasher := auth.NewBcryptPasswordHasher()
	tokenService := auth.NewJWTService(config.JWTSecret, time.Now)
	registrationService := application.NewRegistrationService(repository, passwordHasher)
	authenticationService := application.NewAuthenticationService(repository, passwordHasher, tokenService, config.JWTTTL)
	userService := application.NewUserService(repository)
	authHandler := httphandlers.NewAuthHandler(registrationService, authenticationService)
	userHandler := httphandlers.NewUserHandler(userService)
	grpcAddress := fmt.Sprintf(":%d", config.GRPCPort)
	grpcListener, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		return fmt.Errorf("gRPC startup error: %w", err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcapi.RequireAuthentication(tokenService)))
	userv1.RegisterUserServiceServer(grpcServer, grpcapi.NewUserServer(registrationService, userService))
	defer grpcServer.Stop()
	grpcStopped := make(chan error, 1)
	go func() {
		grpcStopped <- grpcServer.Serve(grpcListener)
	}()
	log.Printf("gRPC server listening on %s", grpcAddress)

	address := fmt.Sprintf(":%d", config.HTTPPort)
	log.Printf("API server listening on %s", address)
	handler := httpapi.NewHandler(
		httproutes.NewAuthRoutes(authHandler),
		httproutes.NewUserRoutes(userHandler, tokenService),
	)
	server := &http.Server{Addr: address, Handler: handler}
	serverStopped := make(chan error, 1)
	go func() {
		serverStopped <- server.ListenAndServe()
	}()

	select {
	case err := <-serverStopped:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("API server failed: %w", err)
		}
		return nil
	case err := <-grpcStopped:
		if err != nil {
			return fmt.Errorf("gRPC server failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Print("servers shutting down")
	}

	grpcServer.GracefulStop()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		_ = server.Close()
		return fmt.Errorf("HTTP shutdown error: %w", err)
	}
	return nil
}
