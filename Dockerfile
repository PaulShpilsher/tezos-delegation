# --- Build Stage ---
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o tezoz-delegation ./cmd/main.go

# # --- Test Stage ---
# FROM builder AS tester
# RUN go test ./...

# --- Final Stage ---
FROM alpine:latest AS final
WORKDIR /app

# Install CA certificates if needed
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN adduser -D -g '' appuser

COPY --from=builder /app/tezoz-delegation ./tezoz-delegation

# Set permissions
RUN chown -R appuser:appuser /app

USER appuser

ENTRYPOINT ["/app/tezoz-delegation"] 