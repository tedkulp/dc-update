# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY ./cmd/ cmd
COPY ./internal/ internal

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o dc-update cmd/dc-update/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates docker-cli docker-compose

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /build/dc-update .

ENTRYPOINT ["./dc-update"]