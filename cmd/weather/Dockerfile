# syntax=docker/dockerfile:1.7
ARG GO_VERSION=1.26
ARG ALPINE_VERSION=3.23

# ---------- build ----------
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download && go mod verify

COPY . .

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -trimpath -ldflags="-s -w" -o /out/weather ./cmd/weather

# ---------- runtime ----------
FROM alpine:${ALPINE_VERSION} AS runtime

ARG VERSION=dev
LABEL org.opencontainers.image.title="weather" \
    org.opencontainers.image.version="${VERSION}"

# ca-certificates — HTTPS OpenWeather, tzdata — TIME_ZONE (time.LoadLocation)
RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S -g 10001 app \
    && adduser -S -u 10001 -G app -H -s /sbin/nologin app \
    && mkdir -p /app/out/logs \
    && chown -R 10001:10001 /app

WORKDIR /app
COPY --from=build --chown=10001:10001 /out/weather /app/weather
COPY --chown=10001:10001 public /app/public

ENV PROJECT_ROOT=/app \
    HTTP_ADDR=:5050

USER 10001:10001
EXPOSE 5050

HEALTHCHECK --interval=10s --timeout=3s --start-period=15s --retries=3 \
    CMD wget -q -O /dev/null http://127.0.0.1:5050/healthz || exit 1

ENTRYPOINT ["/app/weather"]