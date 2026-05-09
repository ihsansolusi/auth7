# Dockerfile — multi-stage build

# Stage 1: Build
FROM golang:1.26-alpine AS builder

ARG VERSION=dev
ARG GITHUB_TOKEN

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Authenticate to GitHub for private module download
RUN git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

COPY go.mod go.sum ./
RUN GONOSUMDB="github.com/ihsansolusi" GOPRIVATE="github.com/ihsansolusi/*" go mod download

# Clear token from git config (not baked into image)
RUN git config --global --unset url."https://${GITHUB_TOKEN}@github.com/".insteadOf 2>/dev/null || true

COPY . .
RUN echo "=== /app contents ===" && ls -la /app && echo "=== cmd/ ===" && ls -la /app/cmd || true && echo "=== cmd/server/ ===" && ls /app/cmd/server || true
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION}" \
    -o bin/auth7 \
    ./cmd/server/ \
    || (go build ./... 2>&1 | head -40 && exit 1)

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
