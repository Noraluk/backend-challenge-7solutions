SHELL := /bin/sh

BINARY ?= bin/api
COMPOSE ?= docker compose
BUF_VERSION ?= v1.65.0

.PHONY: help fmt mocks proto proto-generate tidy test test-race vet build run check compose-config compose-up compose-down compose-reset

help:
	@printf '%s\n' \
		'fmt             Format Go source files' \
		'mocks           Generate gomock implementations with mockgen' \
		'proto           Validate Protobuf contracts' \
		'proto-generate  Validate contracts and generate Go code' \
		'tidy            Synchronize Go module dependencies' \
		'test            Run unit tests' \
		'test-race       Run unit tests with the race detector' \
		'vet             Run Go static analysis' \
		'build           Build the API binary' \
		'run             Run the API locally' \
		'check           Run tests, race detection, vet, and build' \
		'compose-config  Validate Docker Compose configuration' \
		'compose-up      Build and start the local stack' \
		'compose-down    Stop the local stack and preserve data' \
		'compose-reset   Stop the local stack and remove its data'

fmt:
	gofmt -w $$(find cmd gen internal -type f -name '*.go')

mocks:
	mkdir -p internal/mocks
	go tool mockgen -destination=internal/mocks/mock_user_repository.go -package=mocks -write_package_comment=false -write_source_comment=false github.com/Noraluk/backend-challenge-7solutions/internal/ports UserRepository
	go tool mockgen -destination=internal/mocks/mock_password_hasher.go -package=mocks -write_package_comment=false -write_source_comment=false github.com/Noraluk/backend-challenge-7solutions/internal/ports PasswordHasher
	go tool mockgen -destination=internal/mocks/mock_token_service.go -package=mocks -write_package_comment=false -write_source_comment=false github.com/Noraluk/backend-challenge-7solutions/internal/ports TokenService
	go tool mockgen -destination=internal/mocks/mock_registration_usecase.go -package=mocks -write_package_comment=false -write_source_comment=false github.com/Noraluk/backend-challenge-7solutions/internal/ports RegistrationUseCase
	go tool mockgen -destination=internal/mocks/mock_authentication_usecase.go -package=mocks -write_package_comment=false -write_source_comment=false github.com/Noraluk/backend-challenge-7solutions/internal/ports AuthenticationUseCase
	go tool mockgen -destination=internal/mocks/mock_user_usecase.go -package=mocks -write_package_comment=false -write_source_comment=false github.com/Noraluk/backend-challenge-7solutions/internal/ports UserUseCase
	go tool mockgen -source=internal/adapters/mongodb/user_repository.go -destination=internal/mocks/mock_user_collection.go -package=mocks -mock_names=userCollection=MockUserCollection -write_package_comment=false -write_source_comment=false
	gofmt -w internal/mocks

proto:
	go run github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION) lint
	go run github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION) build

proto-generate: proto
	go run github.com/bufbuild/buf/cmd/buf@$(BUF_VERSION) generate

tidy:
	go mod tidy

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

build:
	mkdir -p $$(dirname $(BINARY))
	CGO_ENABLED=0 go build -trimpath -o $(BINARY) ./cmd/api

run:
	go run ./cmd/api

check: test test-race vet build

compose-config:
	$(COMPOSE) config --quiet

compose-up:
	$(COMPOSE) up --build

compose-down:
	$(COMPOSE) down

compose-reset:
	$(COMPOSE) down --volumes
