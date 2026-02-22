# ---- Стадия сборки ----
FROM golang:1.25.5-alpine AS builder
# Рабочая директория внутри контейнера
WORKDIR /app
# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download
# Копируем весь проект
COPY . .
# Сборка бинарника
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o app

# ---- Стадия выполнения ----
FROM alpine:latest
ARG TIME_ZONE=UTC
ENV TZ=${TIME_ZONE}
RUN apk --no-cache add ca-certificates tzdata && \
	cp /usr/share/zoneinfo/${TZ} /etc/localtime && \
	echo ${TZ} > /etc/timezone
WORKDIR /app
# Копируем бинарник
COPY --from=builder /app/app .
COPY --from=builder /app/migrations ./migrations
# Запускаем приложение
CMD ["./app"]
