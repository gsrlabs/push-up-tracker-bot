#!/bin/bash

# Цвета
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Конфигурация
COMPOSE="docker compose"
BOT="bot"
DB="postgres"
NETWORK="pushup-network"

# Функции
success() { echo -e "${GREEN} $1${NC}"; }
info()    { echo -e "${BLUE} $1${NC}"; }
warning() { echo -e "${YELLOW} $1${NC}"; }
error()   { echo -e "${RED} $1${NC}"; }

# Проверка: запущен ли compose
check_running() {
    if ! $COMPOSE ps &>/dev/null; then
        return 1
    fi
}

# Показать статус
show_status() {
    echo ""
    info "Статус контейнеров:"
    $COMPOSE ps
    echo ""
}

# Показать подсказку
show_help() {
    echo ""
    info "Управление проектом PushUp Tracker"
    echo ""
    echo -e "  ${YELLOW}run.sh start${NC}          — Запустить все контейнеры (БД + бот)"
    echo -e "  ${YELLOW}run.sh stop${NC}           — Остановить все контейнеры (БЕЗ удаления volume)"
    echo -e "  ${YELLOW}run.sh restart${NC}        — Перезапустить все"
    echo -e "  ${YELLOW}run.sh rebuild${NC}        — Пересобрать и запустить (go build)"
    echo -e "  ${YELLOW}run.sh logs${NC}           — Логи бота в реальном времени"
    echo -e "  ${YELLOW}run.sh status${NC}         — Показать статус"
    echo ""
    echo -e "  ${YELLOW}run.sh start-bot${NC}      — Запустить только бота"
    echo -e "  ${YELLOW}run.sh stop-bot${NC}       — Остановить только бота"
    echo -e "  ${YELLOW}run.sh restart-bot${NC}    — Перезапустить бота"
    echo ""
    echo -e "  ${YELLOW}run.sh start-db${NC}       — Запустить только БД"
    echo -e "  ${YELLOW}run.sh stop-db${NC}        — Остановить только БД"
    echo -e "  ${YELLOW}run.sh restart-db${NC}     — Перезапустить БД"
    echo ""
    echo -e "  ${YELLOW}run.sh db${NC}             — Подключиться к БД (psql)"
    echo -e "  ${YELLOW}run.sh down-clean${NC}     — УДАЛИТЬ volume (ОСТОРОЖНО!)"
    echo ""
    info "Пример: ./scripts/run.sh start"
    echo ""
}

# === ОСНОВНАЯ ЛОГИКА ===
case "$1" in

    "start")
        echo "Запуск контейнеров..."
        $COMPOSE up -d
        sleep 2
        show_status
        success "Контейнеры запущены"
        ;;

    "stop")
        echo "Остановка контейнеров (volume сохранён)..."
        $COMPOSE stop
        success "Контейнеры остановлены"
        ;;

    "restart")
        echo "Перезапуск всех контейнеров..."
        $COMPOSE restart
        sleep 2
        show_status
        success "Перезапуск завершён"
        ;;

    "rebuild")
        echo "Пересборка и запуск..."
        $COMPOSE down
        $COMPOSE up --build -d
        sleep 2
        show_status
        success "Контейнеры пересобраны и запущены"
        ;;

    "logs")
        echo "Логи бота (Ctrl+C для выхода)..."
        $COMPOSE logs -f "$BOT"
        ;;

    "status")
        show_status
        ;;

    # === Только бот ===
    "start-bot")
        echo "Запуск бота..."
        $COMPOSE up -d "$BOT"
        sleep 1
        $COMPOSE ps "$BOT"
        success "Бот запущен"
        ;;

    "stop-bot")
        echo "Остановка бота..."
        $COMPOSE stop "$BOT"
        success "Бот остановлен"
        ;;

    "restart-bot")
        echo "Перезапуск бота..."
        $COMPOSE restart "$BOT"
        sleep 1
        $COMPOSE ps "$BOT"
        success "Бот перезапущен"
        ;;

    # === Только БД ===
    "start-db")
        echo "Запуск PostgreSQL..."
        $COMPOSE up -d "$DB"
        sleep 3
        $COMPOSE ps "$DB"
        success "БД запущена"
        ;;

    "stop-db")
        echo "Остановка PostgreSQL..."
        $COMPOSE stop "$DB"
        success "БД остановлена"
        ;;

    "restart-db")
        echo "Перезапуск PostgreSQL..."
        $COMPOSE restart "$DB"
        sleep 3
        $COMPOSE ps "$DB"
        success "БД перезапущена"
        ;;
        
     "db")
        echo "Подключение к базе данных..."
        if docker ps --format "table {{.Names}}" | grep -q "$DB"; then
            $COMPOSE exec "$DB" psql -U pushup_user -d pushup_tracker
        else
            error "Контейнер БД ($DB) не запущен!"
            info "Запустите: ./scripts/run.sh start-db"
            exit 1
        fi
        ;;

    # === ОПАСНАЯ КОМАНДА ===
    "down-clean")
        warning "ВНИМАНИЕ! Это УДАЛИТ ВСЕ ДАННЫЕ (volume)!"
        warning "Все пользователи и статистика будут потеряны!"
        read -p "Введите 'YES' для подтверждения: " confirm
        if [[ "$confirm" == "YES" ]]; then
            echo "Удаление volume..."
            $COMPOSE down -v
            success "Проект полностью очищен"
        else
            error "Отменено"
        fi
        ;;

    "")
        show_help
        ;;

    *)
        error "Неизвестная команда: $1"
        show_help
        exit 1
        ;;

esac
