# Простой Dockerfile для использования с пре-собранным бинарником
# Используйте: make build-local && docker compose -f docker-compose.simple.yml up -d

FROM alpine:3.23

WORKDIR /app

# Копируем готовый бинарник из контекста сборки
COPY exe /app/exe

# Копируем публичные файлы
COPY public /app/public

ENV PROJECT_ROOT=/app

CMD ["/app/exe"]
