#!/bin/bash

BACKUP_DIR="./backups"
CONTAINER_NAME="pushup-db"
DB_NAME="pushup_tracker"
DB_USER="pushup_user"

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

success() { echo -e "${GREEN}✅ $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }
info() { echo -e "${BLUE}🔹 $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }

# Проверка наличия аргумента
if [ -z "$1" ]; then
    error "Укажите файл бекапа для восстановления"
    info "Использование: $0 <файл_бекапа.sql.gz>"
    info "Доступные бекапы:"
    ls -lh "$BACKUP_DIR"/*.gz 2>/dev/null || echo "   Нет бекапов"
    exit 1
fi

BACKUP_FILE="$1"

# Проверяем существование файла
if [ ! -f "$BACKUP_FILE" ]; then
    # Пробуем найти в директории бекапов
    if [ -f "$BACKUP_DIR/$1" ]; then
        BACKUP_FILE="$BACKUP_DIR/$1"
    else
        error "Файл бекапа не найден: $1"
        exit 1
    fi
fi

# Определяем тип бекапа
if [[ "$BACKUP_FILE" == *volume*.tar.gz ]]; then
    warning "Это бекап Docker volume, а не SQL бекап"
    read -p "Восстановить volume? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        VOLUME_NAME="push-up-tracker-bot_postgres_data"
        info "Останавливаем контейнер БД..."
        docker stop "$CONTAINER_NAME"
        
        info "Восстанавливаем volume из бекапа..."
        docker run --rm -v "$VOLUME_NAME":/volume -v "$(dirname "$BACKUP_FILE")":/backup alpine sh -c "rm -rf /volume/* && tar xzf /backup/$(basename "$BACKUP_FILE") -C /volume"
        
        info "Запускаем контейнер БД..."
        docker start "$CONTAINER_NAME"
        
        success "Volume восстановлен"
    fi
    exit 0
fi

# Проверяем что контейнер запущен
if ! docker ps | grep -q "$CONTAINER_NAME"; then
    error "Контейнер $CONTAINER_NAME не запущен"
    exit 1
fi

echo "🔄 Восстановление базы данных из: $BACKUP_FILE"

# Подтверждение
warning "ВНИМАНИЕ: Это удалит текущие данные в базе!"
read -p "Продолжить? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    info "Восстановление отменено"
    exit 0
fi

# Определяем тип бекапа по имени файла
if [[ "$BACKUP_FILE" == *schema* ]]; then
    warning "Это бекап только схемы (без данных)"
    info "Восстановление схемы..."
    gunzip -c "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME"
elif [[ "$BACKUP_FILE" == *data* ]]; then
    warning "Это бекап только данных (без схемы)"
    info "Восстановление данных..."
    gunzip -c "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME"
else
    info "Полное восстановление базы данных..."
    
    # Очищаем существующие данные
    info "Очистка существующих данных..."
    docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
    
    # Восстанавливаем из бекапа
    info "Восстановление из бекапа..."
    gunzip -c "$BACKUP_FILE" | docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME"
fi

if [ $? -eq 0 ]; then
    success "База данных успешно восстановлена"
    
    # Показываем статистику после восстановления
    echo ""
    info "📊 Статистика после восстановления:"
    docker exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT 
            (SELECT COUNT(*) FROM users) as users_count,
            (SELECT COUNT(*) FROM pushups) as pushups_count,
            (SELECT COUNT(*) FROM max_reps_history) as history_count;
    "
else
    error "Ошибка при восстановлении базы данных"
    exit 1
fi