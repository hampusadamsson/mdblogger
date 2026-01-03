# Use the Go version from your project
ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 1. Fetch dependencies - this layer is cached unless go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# 2. Copy source and build
COPY . .
# CGO_ENABLED=0 creates a statically linked binary that runs anywhere
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o output .

# Final stage: Use a tiny alpine image
FROM alpine:latest

# Install CA certificates for secure connections
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary and all required assets from the builder
COPY --from=builder /app/output ./
COPY --from=builder /app/templates/ ./templates/
COPY --from=builder /app/static/ ./static/
COPY --from=builder /app/content/ ./content/

# Set Environment Variables (Note: no spaces around '=')
ENV BLOG_PATH="/app/content"
ENV BLOG_PORT="8080"
ENV BLOG_HOST="0.0.0.0"

# Expose the port defined in your config
EXPOSE 8080

# Run the app
ENTRYPOINT ["./output"]
