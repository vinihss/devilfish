# Dockerfile
# Production build - minimal image using distroless

FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=1 GOOS=linux go build -o /devilfishd ./cmd/devilfishd

# Final stage - minimal
FROM gcr.io/distroless/base-debian12:nonroot

# Copy binary from builder
COPY --from=builder /devilfishd /devilfishd

# Use non-root user
USER nonroot:nonroot

# Expose ports
EXPOSE 8080 8081

# Set entrypoint
ENTRYPOINT ["/devilfishd"]