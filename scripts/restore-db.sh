#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups/db"
DB_NAME="pushup_tracker"
DB_USER="pushup_user"
CONTAINER_NAME="pushup-db"
BOT_SERVICE="bot"
COMPOSE_CMD="docker compose"

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

error() { echo -e "${RED}❌ $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️ $1${NC}"; }
info() { echo -e "${BLUE}🔹 $1${NC}"; }

# Проверка аргументов
if [ -z "$1" ]; then
    error "Укажите файл бекапа"
    echo "Использование: $0 <backup_file.gz>"
    echo ""
    info "Доступные бекапы:"
    find "$BACKUP_DIR" -name "*.gz" -type f -printf "%Tb %Td %TY %TH:%TM | %f\n" 2>/dev/null | sort -r | head -10
    echo ""
    
    # Автоматически показываем последний бекап для примера
    LATEST_BACKUP=$(find "$BACKUP_DIR" -name "*.gz" -type f -printf "%T@ %f\n" 2>/dev/null | sort -nr | head -1 | cut -d' ' -f2-)
    if [ -n "$LATEST_BACKUP" ]; then
        info "Пример: $0 $LATEST_BACKUP"
    else
        info "Пример: $0 pushup_tracker_20251116_114800.sql.gz"
    fi
    exit 1
fi

BACKUP_FILE="$1"
[[ "$BACKUP_FILE" != *"/"* ]] && BACKUP_FILE="$BACKUP_DIR/$BACKUP_FILE"

# Проверка файла
if [ ! -f "$BACKUP_FILE" ]; then
    error "Файл не найден: $BACKUP_FILE"
    
    # Проверяем существование директории
    if [ ! -d "$BACKUP_DIR" ]; then
        error "Директория бекапов не существует: $BACKUP_DIR"
        info "Создайте бекап: ./scripts/backup.sh"
        exit 1
    fi
    
    BASENAME=$(basename "$BACKUP_FILE" .gz)
    SIMILAR_FILES=$(find "$BACKUP_DIR" -name "${BASENAME}*" -type f 2>/dev/null | head -5)
    if [ -n "$SIMILAR_FILES" ]; then
        info "Возможно вы имели в виду:"
        echo "$SIMILAR_FILES"
    else
        info "Доступные бекапы:"
        find "$BACKUP_DIR" -name "*.gz" -type f -printf "  %f\n" 2>/dev/null | sort -r | head -5
    fi
    exit 1
fi

# Проверка целостности
info "Проверка целостности бекапа..."
if ! gzip -t "$BACKUP_FILE" 2>/dev/null; then
    error "Бекап повреждён или имеет неверный формат"
    exit 1
fi
success "Бекап прошел проверку целостности"

# Проверка контейнера БД
if ! docker ps --format "table {{.Names}}" | grep -q "^$CONTAINER_NAME$"; then
    error "Контейнер БД $CONTAINER_NAME не запущен"
    info "Запустите: $COMPOSE_CMD up -d postgres"
    exit 1
fi

# Подтверждение
echo ""
warning "╔══════════════════════════════════════════════════╗"
warning "║               ВНИМАНИЕ! ОПАСНО!                 ║"
warning "║    Все текущие данные БД будут удалены!         ║"
warning "║     Пользователи, статистика - всё уйдёт!       ║"
warning "╚══════════════════════════════════════════════════╝"
echo ""
read -p "Вы уверены, что хотите продолжить? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    info "Восстановление отменено"
    exit 0
fi

# Остановка бота
info "Останавливаем бота..."
if $COMPOSE_CMD stop "$BOT_SERVICE" > /dev/null 2>&1; then
    success "Бот остановлен"
else
    warning "Бот не был запущен или не удалось остановить"
fi

# Пересоздание БД
info "Подготовка базы данных..."
if ! docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" > /dev/null 2>&1; then
    error "Не удалось удалить базу данных"
    $COMPOSE_CMD start "$BOT_SERVICE" > /dev/null 2>&1 || true
    exit 1
fi

if ! docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME;" > /dev/null 2>&1; then
    error "Не удалось создать базу данных"
    $COMPOSE_CMD start "$BOT_SERVICE" > /dev/null 2>&1 || true
    exit 1
fi
success "База данных подготовлена"

# Восстановление
info "Восстановление данных из бекапа..."
START_TIME=$(date +%s)

if gunzip -c "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" > /dev/null 2>&1; then
    DURATION=$(( $(date +%s) - START_TIME ))
    success "Данные восстановлены за ${DURATION}с"

    # Проверка восстановленных данных
    info "Проверка восстановленных данных..."
    TABLE_COUNT=$(docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';" 2>/dev/null | tr -d ' \n')
    if [ -n "$TABLE_COUNT" ] && [ "$TABLE_COUNT" -gt 0 ]; then
        success "Таблиц восстановлено: $TABLE_COUNT"
    else
        warning "Не удалось проверить количество таблиц"
    fi

    USER_COUNT=$(docker exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM users;" 2>/dev/null | tr -d ' \n')
    if [ -n "$USER_COUNT" ]; then
        success "Пользователей восстановлено: $USER_COUNT"
    fi
else
    error "Ошибка при восстановлении данных из бекапа"
    warning "Пытаемся запустить бота..."
    $COMPOSE_CMD start "$BOT_SERVICE" > /dev/null 2>&1 || true
    exit 1
fi

# Запуск бота
info "Запускаем бота..."
if $COMPOSE_CMD up -d "$BOT_SERVICE" > /dev/null 2>&1; then
    success "Бот запущен"
else
    error "Не удалось запустить бота"
    info "Запустите вручную: $COMPOSE_CMD up -d $BOT_SERVICE"
fi

# Финальная информация
echo ""
success "╔══════════════════════════════════════════════════╗"
success "║           ВОССТАНОВЛЕНИЕ ЗАВЕРШЕНО!             ║"
success "╚══════════════════════════════════════════════════╝"
echo ""
info "📁 Файл: $(basename "$BACKUP_FILE")"
info "📏 Размер: $(du -h "$BACKUP_FILE" | cut -f1)"
info "📅 Дата создания: $(stat -c %y "$BACKUP_FILE" 2>/dev/null | cut -d'.' -f1 || echo "неизвестно")"
info "⏱️ Время восстановления: ${DURATION}с"
info "🗃️ Таблиц в БД: $TABLE_COUNT"
if [ -n "$USER_COUNT" ]; then
    info "👥 Пользователей: $USER_COUNT"
fi
echo ""
info "🌐 Бот должен быть доступен через несколько секунд"