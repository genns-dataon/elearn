# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy api source code
COPY api/ ./

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install SQLite
RUN apk --no-cache add sqlite-libs ca-certificates

# Copy binary from builder
COPY --from=builder /app/main .

# Copy prompts directory
COPY --from=builder /app/prompts ./prompts

# Create storage directory
RUN mkdir -p /app/storage/uploads /app/storage/audio

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]
