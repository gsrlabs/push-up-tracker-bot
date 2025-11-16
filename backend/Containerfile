# ---- Стадия сборки ----
FROM docker.io/library/golang:1.25 AS builder

# Рабочая директория внутри контейнера
WORKDIR /app

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Сборка бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o trackerbot main.go

# ---- Стадия выполнения ----
FROM docker.io/library/alpine:3.22

WORKDIR /app

# Устанавливаем зависимости для работы с PostgreSQL
RUN apk add --no-cache ca-certificates

# Копируем бинарник
COPY --from=builder /app/trackerbot .

# Запускаем приложение
CMD ["./trackerbot"]
