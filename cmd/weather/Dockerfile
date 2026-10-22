FROM golang:1.26.1-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /app/cmd/weather/exe /app/cmd/weather

FROM alpine:3.23

WORKDIR /app

COPY --from=builder /app/cmd/weather/exe /app

COPY --from=builder /app/public /app/public

ENV PROJECT_ROOT=/app

CMD ["/app/exe"]