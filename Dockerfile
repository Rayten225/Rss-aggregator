# Этап 1: Сборка приложения
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Копируем go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь исходный код
COPY . .

# Компилируем приложение, указав правильный путь к main.go
RUN go build -o goNews ./src/main.go

# Этап 2: Создание финального образа
FROM alpine:latest

WORKDIR /app

# Копируем скомпилированный бинарник и статические файлы
COPY --from=builder /app/goNews .
COPY ./src/webapp ./src/webapp
COPY ./src/config.json ./src/config.json

# Устанавливаем переменные окружения
ENV dbpass=admin

# Открываем порт 8000
EXPOSE 8000

# Запускаем приложение
CMD ["./goNews"]