# Build stage
FROM golang:1.23.6-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o fulcrum ./cmd/fulcrum

# Final stage
FROM alpine:latest

# Install CA certificates for HTTPS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/fulcrum .

# Expose the application port
EXPOSE 3000

# Run the binary
CMD ["./fulcrum"]
