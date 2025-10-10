# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o rapidrtmp .

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 rapidrtmp && \
    adduser -D -u 1000 -G rapidrtmp rapidrtmp

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/rapidrtmp .

# Copy test player (optional)
COPY test-player.html ./

# Create data directory
RUN mkdir -p /app/data/streams && \
    chown -R rapidrtmp:rapidrtmp /app

# Switch to non-root user
USER rapidrtmp

# Expose ports
EXPOSE 8080 1935

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./rapidrtmp"]

