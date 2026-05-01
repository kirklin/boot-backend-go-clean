# =============================================================================
# Boot Backend Go Clean — Production Build (Multi-stage Distroless)
# =============================================================================

# ---- Stage 1: Build ----
FROM golang:1.26-alpine AS builder

# Set proxy (can be overridden by --build-arg)
ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=${GOPROXY}

# Version info
ARG VERSION=unknown
ARG GIT_COMMIT=unknown

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY pkg/ ./pkg/

# Build application with extreme static linking and size reduction
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -extldflags '-static'" \
    -o main ./cmd/main.go

# ---- Stage 2: Runtime (Distroless) ----
# We use the static-debian12 distroless image to ensure zero attack surface.
# It has no shell, no package manager, making it incredibly secure.
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the compiled binary from the builder
COPY --from=builder /app/main /main

# The application port is defined via environment variables
# EXPOSE 8888

# Run as nonroot user
USER nonroot:nonroot

ENTRYPOINT ["/main"]
