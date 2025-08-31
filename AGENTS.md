# AGENTS.md

This file provides instructions for AI coding agents to effectively work on this project.

## Project Overview

This is a backend service written in Go, following the Clean Architecture principles. The goal is to maintain a clear separation of concerns between business logic and infrastructure details.

-   **`internal/domain`**: Contains core business logic, entities, and interfaces for repositories and services. This is the most independent layer.
-   **`internal/usecase`**: Implements the business logic by orchestrating domain entities and repository interfaces.
-   **`internal/infrastructure`**: Provides concrete implementations for domain interfaces, such as database repositories (`persistence`) and external services (`auth`).
-   **`internal/interfaces`**: Exposes the application to the outside world, primarily through an HTTP API (`http`). This layer handles request/response logic.
-   **`cmd`**: The main entry point of the application.

When adding features, respect this structure. For instance, a new business rule should be in the `domain` or `usecase` layer, not in an HTTP controller.

## Development Environment Setup

There are two primary ways to set up the development environment. Docker is the recommended approach.

### Recommended: Docker Setup

This is the simplest way to get the application and its dependencies (like a database) running.

1.  **Copy Environment File**:
    ```bash
    cp .env.example .env
    ```
2.  **Fill in `.env`**: Populate the `.env` file with the necessary configuration values.
3.  **Build and Run**:
    ```bash
    docker-compose up --build
    ```

### Local Go Setup

1.  **Prerequisites**: Ensure you have Go (version 1.25.0 or newer) installed.
2.  **Copy Environment File**:
    ```bash
    cp .env.example .env
    ```
3.  **Fill in `.env`**: Populate the `.env` file with the necessary configuration for a local setup.
4.  **Install Dependencies**:
    ```bash
    go mod tidy
    ```
5.  **Run the Application**:
    ```bash
    go run cmd/main.go
    ```

## Testing Instructions

-   **Run all tests**: To run all tests across all packages, use the following command from the project root:
    ```bash
    go test ./...
    ```
-   **Run tests with coverage**: To see a test coverage report:
    ```bash
    go test -cover ./...
    ```
-   **Run tests for a specific package**: For example, to test only the `usecase` package:
    ```bash
    go test ./internal/usecase/...
    ```
-   Always add or update tests for any code you change. Test files should be named `*_test.go` and reside in the same package as the code they are testing.

## Code Style and Conventions

-   **Formatting**: All Go code must be formatted with `gofmt`. Before committing, run:
    ```bash
    go fmt ./...
    ```
-   **Static Analysis**: Use `go vet` to catch potential issues in the code:
    ```bash
    go vet ./...
    ```
-   **Error Handling**: Handle errors explicitly. Do not ignore them using `_`.
-   **Zero Dependencies in `domain`**: The `domain` package should not have any dependencies on other layers of this application. It should only contain pure Go code and business logic.

## Pull Request (PR) Instructions

-   **Title Format**: Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification. The format is `<type>(<scope>): <subject>`.
    -   **type**: `feat` (new feature), `fix` (bug fix), `docs`, `style`, `refactor`, `test`, `chore`, etc.
    -   **scope**: Optional, indicating the part of the codebase affected (e.g., `domain`, `http`, `persistence`).
    -   **Examples**:
        -   `feat(auth): implement user registration endpoint`
        -   `fix: correct calculation error in billing`
        -   `docs: update README with new setup instructions`
        -   `chore: update dependencies`
-   **Pre-PR Checklist**: Before submitting a pull request, ensure you have:
    1.  Formatted the code: `go fmt ./...`
    2.  Run all tests and confirmed they pass: `go test ./...`
    3.  Updated or added tests for your changes.
