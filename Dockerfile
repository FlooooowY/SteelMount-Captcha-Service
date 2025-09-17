# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies in single layer
RUN apk add --no-cache --virtual .build-deps \
    git \
    ca-certificates \
    tzdata \
    netcat-openbsd \
    upx

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags="-w -s -X main.version=1.0.0 -X main.buildTime=$(date -u +%Y%m%d.%H%M%S)" \
    -o captcha-service ./cmd/server

# Compress binary with UPX
RUN upx --best --lzma captcha-service

# Final stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy binary from builder stage
COPY --from=builder /app/captcha-service /captcha-service

# Copy configuration files
COPY --from=builder /app/config.yaml /config.yaml

# Expose port range
EXPOSE 38000-40000 9090

# Set resource limits
ENV GOMEMLIMIT=512MiB
ENV GOGC=100

# Run the application as non-root user
ENTRYPOINT ["/captcha-service"]
