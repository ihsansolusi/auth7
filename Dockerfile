# Dockerfile — multi-stage build

# Stage 1: Build
FROM golang:1.26-alpine AS builder

ARG VERSION=dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o bin/auth7 \
    ./cmd/server/

# Stage 2: Runtime
FROM alpine:3.19

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/bin/auth7 .
COPY configs/config.example.yaml configs/config.yaml
COPY migrations/ migrations/
COPY scripts/ scripts/

RUN apk --no-cache add ca-certificates postgresql-client && \
    chmod +x scripts/docker-entrypoint.sh

USER appuser

EXPOSE 8083

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s \
    CMD wget -qO- http://localhost:8083/health/live || exit 1

ENTRYPOINT ["./scripts/docker-entrypoint.sh", "./auth7"]
CMD ["start"]
