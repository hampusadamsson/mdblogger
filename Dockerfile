# multiline ARGS is available since Docker 17.05, so you may need to combine these into a single ARG
ARG GO_VERSION=1.25

# We choose alpine as our base image to minimize the size of the final image
FROM golang:${GO_VERSION}-alpine AS builder

# Enable go modules
ENV GO111MODULE=on

# Install gcc for cgo
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go.mod and go.sum to fetch dependencies efficiently
COPY go.mod go.sum ./

# Download dependencies before building for better caching
RUN go mod download

# Next, copy the entire source code
COPY . .

# Perform the build
RUN CGO_ENABLED=0 go build -o output main.go

RUN ls /app
RUN ls /app/content

# Final stage, only the binary
FROM alpine:latest

WORKDIR /app

# Copy the binary
COPY --from=builder /app/output ./
COPY --from=builder /app/templates/ /app/templates/
COPY --from=builder /app/static/ /app/static/
COPY --from=builder /app/content/ /app/content/

RUN ls /app
RUN ls /app/content

# ENV
ENV MD_CONTENT_PATH = "/app/content"

# Port on which the service will be exposed
EXPOSE 8080

# Command to run the binary
CMD ["./output"]
