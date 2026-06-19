# ==============================================
# webdata-service - Multi-stage Dockerfile
# ==============================================

# Stage 1: Dependencies
FROM golang:1.25-alpine AS deps
WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

ARG GITHUB_TOKEN
ENV GOPRIVATE=github.com/mercadocercano/*
RUN if [ -n "$GITHUB_TOKEN" ]; then git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; fi

COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Stage 2: Builder
FROM deps AS builder

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -trimpath \
    -o webdata-service ./cmd/api/main.go

# Stage 3: Development (with Air hot reload)
FROM mercado-cercano/go-dev:1.24 AS development

WORKDIR /app

ARG GITHUB_TOKEN
RUN if [ -n "$GITHUB_TOKEN" ]; then git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; fi

COPY --chown=appuser:appgroup go.mod go.sum ./
RUN go mod download

COPY --chown=appuser:appgroup . .

RUN mkdir -p tmp migrations logs /go/pkg/mod && \
    chmod -R 777 /go/pkg && \
    chown -R appuser:appgroup /app tmp migrations logs

USER appuser

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD curl -f http://localhost:8150/health || exit 1

EXPOSE 8150 2114

CMD sh -c 'if [ -n "$GITHUB_TOKEN" ]; then git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"; fi && air -c .air.toml'

# Stage 4: Production
FROM gcr.io/distroless/static-debian12:nonroot AS production

LABEL org.opencontainers.image.title="webdata-service" \
      org.opencontainers.image.description="Web Data Layer — Firecrawl-powered product intelligence" \
      org.opencontainers.image.vendor="Mercado Cercano"

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /app/webdata-service ./

USER nonroot

EXPOSE 8150

ENTRYPOINT ["./webdata-service"]

# Default: Development
FROM development
