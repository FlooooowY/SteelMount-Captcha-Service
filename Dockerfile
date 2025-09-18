# Build stage
FROM golang:1.21-alpine AS builder

# Install only essential packages
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Skip protobuf generation for faster build (files should be pre-generated)

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o captcha-service ./cmd/server

# Final stage
FROM alpine:latest

# Install minimal packages for runtime
RUN apk --no-cache add ca-certificates wget

# Create non-root user
RUN addgroup -g 1001 -S captcha && \
    adduser -u 1001 -S captcha -G captcha

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/captcha-service .

# Copy configuration files if any
COPY --from=builder /app/config.yaml .

# Change ownership to non-root user
RUN chown -R captcha:captcha /app

# Switch to non-root user
USER captcha

# Expose port range
EXPOSE 38000-40000

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:9090/health || exit 1

# Run the application
CMD ["./captcha-service"]
