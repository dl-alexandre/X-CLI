# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /x ./cmd/x

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /x /usr/local/bin/x

# Create config directory
RUN mkdir -p /root/.config/x-cli

# Set environment
ENV XDG_CONFIG_HOME=/root/.config
ENV HOME=/root

# Default command
ENTRYPOINT ["x"]
CMD ["--help"]
