.PHONY: help test test-coverage fmt vet run build dev dev-down dev-rebuild dev-logs docker-build docker-build-dev docker-push docker-build-push docker-clean

SHELL := /bin/bash

# =============================================================================
# Version Information
# =============================================================================
VERSION    := $(shell cat VERSION 2>/dev/null || echo "1.0.0")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
TAG        ?= $(VERSION)

VERSION_PKG := main
GO_LDFLAGS  := -s -w -X $(VERSION_PKG).Version=$(VERSION) \
               -X $(VERSION_PKG).GitCommit=$(GIT_COMMIT)

export GOPROXY ?= https://goproxy.cn,direct

# Colors for help command
CYAN  := \033[36m
GREEN := \033[32m
RESET := \033[0m

# =============================================================================
# Help
# =============================================================================

## Show this help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")-1); \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			printf "  $(CYAN)%-20s$(RESET) $(GREEN)%s$(RESET)\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

# =============================================================================
# Build & Run
# =============================================================================

## Build the Go application
build:
	go build -ldflags "$(GO_LDFLAGS)" -o ./bin/main ./cmd/main.go

## Run the application locally
run:
	go run cmd/main.go

# =============================================================================
# Testing
# =============================================================================

## Run all tests
test:
	go test ./...

## Run tests with coverage report
test-coverage:
	go test -coverprofile=coverage.out -covermode=atomic ./...

# =============================================================================
# Code Quality
# =============================================================================

## Format Go code
fmt:
	go fmt ./...

## Run go vet to catch potential issues
vet:
	go vet ./...

# =============================================================================
# Local Development (Docker)
# =============================================================================

## Start local dev environment (Hot-Reloading with air)
dev:
	docker compose --project-directory . -f docker/docker-compose.dev.yaml up --build -d

## Stop local dev environment
dev-down:
	docker compose --project-directory . -f docker/docker-compose.dev.yaml down

## Rebuild and restart local dev environment
dev-rebuild: dev-down
	docker compose --project-directory . -f docker/docker-compose.dev.yaml up --build -d

## Follow dev environment logs
dev-logs:
	docker compose --project-directory . -f docker/docker-compose.dev.yaml logs -f

# =============================================================================
# Docker Build Rules
# =============================================================================

include docker/docker.mk
