package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/auth"
	httpapi "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http"
	httproutes "github.com/Noraluk/backend-challenge-7solutions/internal/adapters/http/routes"
	"github.com/Noraluk/backend-challenge-7solutions/internal/adapters/mongodb"
	"github.com/Noraluk/backend-challenge-7solutions/internal/application"
	"github.com/Noraluk/backend-challenge-7solutions/internal/platform"
)

const disconnectTimeout = 10 * time.Second

func main() {
	config, err := platform.LoadConfig()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}
	database, err := mongodb.Connect(context.Background(), config.MongoURI, config.MongoDatabase)
	if err != nil {
		log.Fatalf("MongoDB startup error: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
		defer cancel()
		if err := database.Disconnect(ctx); err != nil {
			log.Printf("MongoDB shutdown error: %v", err)
		}
	}()
	repository := database.UserRepository()
	passwordHasher := auth.NewBcryptPasswordHasher()
	tokenService := auth.NewJWTService(config.JWTSecret, time.Now)
	registrationService := application.NewRegistrationService(repository, passwordHasher, time.Now)
	authenticationService := application.NewAuthenticationService(repository, passwordHasher, tokenService, config.JWTTTL, time.Now)

	address := fmt.Sprintf(":%d", config.HTTPPort)
	log.Printf("API server listening on %s", address)
	handler := httpapi.NewHandler(
		httproutes.NewAuthRoutes(registrationService, authenticationService),
	)
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}
