#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups/db"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="pushup_tracker"
DB_USER="pushup_user"
CONTAINER_NAME="pushup-db"

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

success() { echo -e "${GREEN}✅ $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }
info() { echo -e "${BLUE}🔹 $1${NC}"; }

# Создаём директорию
mkdir -p "$BACKUP_DIR"

echo "🔄 Создание бекапа базы данных..."

# Проверяем что контейнер БД запущен
if ! docker ps --format "table {{.Names}}" | grep -q "^$CONTAINER_NAME$"; then
    error "Контейнер БД $CONTAINER_NAME не запущен"
    info "Запустите: docker compose up -d postgres"
    exit 1
fi

# 1. Бекап БД через pg_dump
info "Создание дампа базы данных..."
if docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" > "$BACKUP_DIR/${DB_NAME}_${DATE}.sql"; then
    # Сжимаем
    gzip "$BACKUP_DIR/${DB_NAME}_${DATE}.sql"
    success "Бекап БД создан: $BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz"
else
    error "Ошибка при создании бекапа"
    # Удаляем частично созданный файл если есть
    rm -f "$BACKUP_DIR/${DB_NAME}_${DATE}.sql"
    exit 1
fi

# Проверяем целостность созданного бекапа
info "Проверка целостности бекапа..."
if gzip -t "$BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz" 2>/dev/null; then
    success "Бекап прошел проверку целостности"
else
    error "Созданный бекап поврежден"
    rm -f "$BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz"
    exit 1
fi

# Статистика
echo ""
success "📊 Бекап успешно создан:"
info "   🗄️  База данных: $BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz"
info "   📏 Размер: $(du -h "$BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz" | cut -f1)"
info "   📅 Дата: $(date +"%d.%m.%Y %H:%M")"

# Дополнительная информация о количестве бекапов
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "*.gz" -type f | wc -l)
TOTAL_SIZE=$(find "$BACKUP_DIR" -name "*.gz" -type f -exec du -cb {} + | tail -1 | cut -f1)

# Конвертируем размер в человеко-читаемый формат
if command -v numfmt >/dev/null 2>&1; then
    TOTAL_SIZE_HR=$(numfmt --to=iec $TOTAL_SIZE)
else
    TOTAL_SIZE_HR="${TOTAL_SIZE} bytes"
fi

echo ""
info "📈 Статистика бекапов:"
info "   Всего бекапов: $BACKUP_COUNT"
info "   Общий размер: $TOTAL_SIZE_HR"