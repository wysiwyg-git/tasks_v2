# Этап 1: сборка бинарника
FROM golang:1.25-alpine AS builder

# Устанавливаем git для скачивания зависимостей (если нужно)
RUN apk add --no-cache git

WORKDIR /app

# Копируем файлы модулей и загружаем зависимости (кешируется)
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Собираем статический бинарник
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/tasks

# Этап 2: минимальный образ для запуска
FROM alpine:3.21

# Устанавливаем ca-certificates для HTTPS-запросов (если будут внешние API)
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Копируем бинарник из первого этапа
COPY --from=builder /app/server .

# Копируем миграции
COPY --from=builder /app/db/migrations /app/db/migrations

# Порт, который слушает приложение (можно переопределить через ENV)
EXPOSE 8080

# Запускаем
CMD ["./server"]