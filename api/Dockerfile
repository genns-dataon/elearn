# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy api source code
COPY api/ ./api/

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o main ./api

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install SQLite
RUN apk --no-cache add sqlite-libs ca-certificates

# Copy binary from builder
COPY --from=builder /app/main .

# Copy prompts directory
COPY --from=builder /app/api/prompts ./prompts

# Create storage directory
RUN mkdir -p /app/storage/uploads /app/storage/audio

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
