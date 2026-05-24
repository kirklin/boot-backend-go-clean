# =============================================================================
# Boot Backend Go Clean — Production Build (Multi-stage Distroless)
# =============================================================================

# ---- Stage 0: Download GeoIP Data ----
FROM alpine:latest AS geoip
RUN apk --no-cache add git
RUN git clone --depth 1 --filter=blob:none --sparse \
      https://github.com/lionsoul2014/ip2region.git /tmp/ip2region && \
    cd /tmp/ip2region && git sparse-checkout set data && \
    mkdir -p /geoip && \
    cp data/ip2region_v4.xdb /geoip/ && \
    cp data/ip2region_v6.xdb /geoip/

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
    -ldflags="-s -w \
      -X github.com/kirklin/boot-backend-go-clean/pkg/version.Version=${VERSION} \
      -X github.com/kirklin/boot-backend-go-clean/pkg/version.GitCommit=${GIT_COMMIT} \
      -X github.com/kirklin/boot-backend-go-clean/pkg/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
      -extldflags '-static'" \
    -o main ./cmd/main.go

# Build lightweight health check binary for Docker HEALTHCHECK
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-s -w -extldflags '-static'" \
    -o healthcheck ./cmd/healthcheck/main.go

# ---- Stage 2: Runtime (Distroless) ----
# We use the static-debian12 distroless image to ensure zero attack surface.
# It has no shell, no package manager, making it incredibly secure.
FROM gcr.io/distroless/static-debian12:nonroot

# Copy the compiled binary from the builder
COPY --from=builder /app/main /main
COPY --from=builder /app/healthcheck /healthcheck

# Copy GeoIP data files (owned by nonroot via --chown)
COPY --from=geoip --chown=nonroot:nonroot /geoip/ /data/

# The application port is defined via environment variables
# EXPOSE 8888

# Run as nonroot user
USER nonroot:nonroot

ENTRYPOINT ["/main"]
