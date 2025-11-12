#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups/db"
DB_NAME="pushup_tracker"
CONTAINER_NAME="pushup-db"
DB_USER="pushup_user"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функции для цветного вывода
error() { echo -e "${RED}❌ $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️ $1${NC}"; }
info() { echo -e "🔹 $1"; }

# Проверка аргументов
if [ -z "$1" ]; then
    error "Укажите файл бекапа для восстановления"
    echo "Использование: $0 <backup_file.gz>"
    echo ""
    info "Доступные бекапы:"
    find $BACKUP_DIR -name "*.gz" -type f -printf "%Tb %Td %TY %TH:%TM | %p\n" 2>/dev/null | sort -r
    echo ""
    info "Пример: $0 backups/db/pushup_tracker_20241215_143022.sql.gz"
    info "Или: $0 pushup_tracker_20241215_143022.sql.gz (если файл в backups/db/)"
    exit 1
fi

BACKUP_FILE=$1

# Если указано только имя файла, добавляем путь
if [[ "$BACKUP_FILE" != *"/"* ]]; then
    BACKUP_FILE="$BACKUP_DIR/$BACKUP_FILE"
fi

# Проверка существования файла
if [ ! -f "$BACKUP_FILE" ]; then
    error "Файл бекапа не найден: $BACKUP_FILE"
    
    # Предлагаем похожие файлы
    BASENAME=$(basename "$BACKUP_FILE" .gz)
    if find $BACKUP_DIR -name "${BASENAME}*" | grep -q .; then
        info "Возможно вы имели в виду:"
        find $BACKUP_DIR -name "${BASENAME}*" -type f | sort -r
    else
        info "Доступные бекапы:"
        find $BACKUP_DIR -name "*.gz" -type f -printf "%Tb %Td %TY %TH:%TM | %p\n" 2>/dev/null | sort -r | head -10
    fi
    exit 1
fi

# Проверка целостности бекапа
info "Проверка целостности бекапа..."
if ! gzip -t "$BACKUP_FILE" 2>/dev/null; then
    error "Бекап поврежден или имеет неверный формат"
    exit 1
fi
success "Бекап прошел проверку целостности"

# Проверка что контейнер БД запущен
if ! docker ps --format "table {{.Names}}" | grep -q "$CONTAINER_NAME"; then
    error "Контейнер БД $CONTAINER_NAME не запущен"
    info "Запустите: docker-compose up -d postgres"
    exit 1
fi

# Предупреждение о потере данных
echo ""
warning "╔══════════════════════════════════════════════════╗"
warning "║               ВНИМАНИЕ! ОПАСНО!                 ║"
warning "║    Это перезапишет текущую базу данных!         ║"
warning "║     Все текущие данные будут потеряны!          ║"
warning "╚══════════════════════════════════════════════════╝"
echo ""
read -p "Вы уверены, что хотите продолжить? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    info "Восстановление отменено"
    exit 0
fi

info "Начинаем восстановление из: $BACKUP_FILE"

# Останавливаем бота чтобы избежать конфликтов
info "Останавливаем бота..."
if docker-compose stop bot 2>/dev/null; then
    success "Бот остановлен"
else
    warning "Не удалось остановить бота (возможно не запущен)"
fi

# Дропаем и создаем базу заново (чистое восстановление)
info "Подготовка базы данных..."
if ! docker exec $CONTAINER_NAME psql -U $DB_USER -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null; then
    error "Не удалось удалить старую базу данных"
    docker-compose start bot 2>/dev/null || true
    exit 1
fi

if ! docker exec $CONTAINER_NAME psql -U $DB_USER -d postgres -c "CREATE DATABASE $DB_NAME;" 2>/dev/null; then
    error "Не удалось создать новую базу данных"
    docker-compose start bot 2>/dev/null || true
    exit 1
fi
success "База данных подготовлена"

# Восстанавливаем базу
info "Восстановление данных..."
START_TIME=$(date +%s)

if gunzip -c "$BACKUP_FILE" | docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME 2>/dev/null; then
    END_TIME=$(date +%s)
    DURATION=$((END_TIME - START_TIME))
    success "База успешно восстановлена за ${DURATION} секунд"
    
    # Проверяем что данные загрузились
    TABLE_COUNT=$(docker exec $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';" 2>/dev/null | tr -d ' \n')
    if [ -n "$TABLE_COUNT" ] && [ "$TABLE_COUNT" -gt 0 ]; then
        success "Загружено таблиц: $TABLE_COUNT"
        
        # Дополнительная проверка - считаем пользователей
        USER_COUNT=$(docker exec $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME -t -c "SELECT count(*) FROM users;" 2>/dev/null | tr -d ' \n')
        if [ -n "$USER_COUNT" ]; then
            success "Загружено пользователей: $USER_COUNT"
        fi
    else
        warning "Не удалось проверить количество таблиц"
    fi
else
    error "Ошибка восстановления базы"
    
    # Пытаемся запустить бота обратно
    warning "Пытаемся запустить бота..."
    docker-compose start bot 2>/dev/null || true
    exit 1
fi

# Запускаем бота обратно
info "Запускаем бота..."
if docker-compose start bot 2>/dev/null || docker-compose up -d bot 2>/dev/null; then
    success "Бот перезапущен"
else
    error "Не удалось запустить бота"
    info "Запустите вручную: docker-compose up -d bot"
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
info "⏱️  Время восстановления: ${DURATION} секунд"
info "🗃️  Таблиц в БД: $TABLE_COUNT"
if [ -n "$USER_COUNT" ]; then
    info "👥 Пользователей: $USER_COUNT"
fi
echo ""
info "🌐 Бот должен быть доступен через несколько секунд"