SHELL := /bin/sh

BINARY ?= bin/api
COMPOSE ?= docker compose

.PHONY: help fmt tidy test test-race vet build run check compose-config compose-up compose-down compose-reset

help:
	@printf '%s\n' \
		'fmt             Format Go source files' \
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
	gofmt -w $$(find cmd internal -type f -name '*.go')

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
