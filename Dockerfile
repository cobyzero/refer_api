# syntax=docker/dockerfile:1

# ---- Go Build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install dependencies for cgo-free build
RUN apk add --no-cache git

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/server .

# ---- Runtime stage ----
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /app/server ./server

# Copy necessary runtime dirs (uploads)
RUN mkdir -p /app/uploads && chown -R appuser:appgroup /app

USER appuser

EXPOSE 3000

# Healthcheck hits the root endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:3000/ || exit 1

CMD ["./server"]
