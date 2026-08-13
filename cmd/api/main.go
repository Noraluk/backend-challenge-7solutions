package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/auth"
	httpapi "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http"
	httphandlers "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/handlers"
	httproutes "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/routes"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/mongodb"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application"
	"github.com/Noraluk/backend-challenge-7solutions/internal/platform"
	"github.com/Noraluk/backend-challenge-7solutions/internal/ports"
)

const disconnectTimeout = 10 * time.Second

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

var serveHTTP = http.ListenAndServe

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}
	database, err := connectDatabase(context.Background(), config.MongoURI, config.MongoDatabase)
	if err != nil {
		return fmt.Errorf("MongoDB startup error: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
		defer cancel()
		if err := database.Disconnect(ctx); err != nil {
			log.Printf("MongoDB shutdown error: %v", err)
		}
	}()
	repository := database.UserRepository()
	workerContext, stopWorker := context.WithCancel(context.Background())
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

	address := fmt.Sprintf(":%d", config.HTTPPort)
	log.Printf("API server listening on %s", address)
	handler := httpapi.NewHandler(
		httproutes.NewAuthRoutes(authHandler),
		httproutes.NewUserRoutes(userHandler, tokenService),
	)
	if err := serveHTTP(address, handler); err != nil {
		return fmt.Errorf("API server failed: %w", err)
	}
	return nil
}
