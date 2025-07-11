# --- Build Stage ---
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o tezos-delegation ./cmd/main.go

# --- CA Certificates Stage ---
FROM alpine:latest AS certs
RUN apk --no-cache add ca-certificates

# --- Final Stage ---
FROM scratch AS final
WORKDIR /app
COPY --from=builder /app/tezos-delegation /app/tezos-delegation
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/app/tezos-delegation"] 