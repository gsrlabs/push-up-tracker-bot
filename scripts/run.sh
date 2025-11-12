#!/bin/bash

case "$1" in
    "start")
        docker-compose up -d
        echo "✅ Контейнеры запущены в фоновом режиме"
        sleep 2
        docker-compose ps
        ;;
    "stop")
        docker-compose down
        echo "✅ Контейнеры остановлены"
        ;;
    "rebuild")
        docker-compose down
        echo "⏳ Пересборка контейнеров..."
        docker-compose up --build -d
        sleep 2
        docker-compose ps
        echo "✅ Контейнеры пересобраны и запущены"
        ;;
    "logs")
        docker-compose logs -f bot
        # Эта команда будет работать до принудительного завершения (Ctrl+C)
        ;;
    "status")
        docker-compose ps
        ;;
    *)
        echo "Использование: $0 {start|stop|restart|logs|status}"
        echo "  start   - запустить контейнеры"
        echo "  stop    - остановить контейнеры"
        echo "  rebuild - пересобрать и запустить контейнеры"
        echo "  logs    - показать логи в реальном времени"
        echo "  status  - показать статус контейнеров"
        exit 1
        ;;
esac

