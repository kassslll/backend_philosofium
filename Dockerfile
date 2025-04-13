# Базовый образ для сборки
FROM golang:1.21-alpine AS builder

# Установка зависимостей
RUN apk add --no-cache git gcc musl-dev

# Создание рабочей директории
WORKDIR /app

# Копирование go.mod и go.sum для скачивания зависимостей
COPY go.mod go.sum ./

# Скачивание зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения
RUN CGO_ENABLED=0 GOOS=linux go build -o backend .

# Финальный образ
FROM alpine:latest

# Установка зависимостей для работы с PostgreSQL
RUN apk add --no-cache libc6-compat

# Создание рабочей директории
WORKDIR /app

# Копирование бинарного файла из builder
COPY --from=builder /app/backend .

# Копирование конфигурационных файлов
COPY config ./config

# Копирование миграций
COPY migrations ./migrations

# Порт, который будет слушать приложение
EXPOSE 8080

# Команда для запуска приложения
CMD ["./backend"]