# Multi-stage build for production
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build both binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/analytics ./cmd/analytics

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates postgresql-client

WORKDIR /root/

# Copy binaries from builder
COPY --from=builder /app/server .
COPY --from=builder /app/analytics .
COPY --from=builder /app/migrations ./migrations

# Expose port
EXPOSE 8080

# Default command (can be overridden)
CMD ["./server"]
