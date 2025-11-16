#!/bin/bash
# Конфигурация
BACKUP_DIR="./backups/db"
DB_NAME="pushup_tracker"
DB_USER="pushup_user"
CONTAINER_NAME="pushup-db"
BOT_SERVICE="bot"

# Цвета
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

error() { echo -e "${RED}❌ $1${NC}"; }
success() { echo -e "${GREEN}✅ $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️ $1${NC}"; }
info() { echo -e "🔹 $1"; }

# Проверка аргументов
if [ -z "$1" ]; then
    error "Укажите файл бекапа"
    echo "Использование: $0 <backup_file.gz>"
    echo ""
    info "Доступные бекапы:"
    find "$BACKUP_DIR" -name "*.gz" -type f -printf "%Tb %Td %TY %TH:%TM | %f\n" 2>/dev/null | sort -r
    echo ""
    info "Пример: $0 pushup_tracker_20251116_114800.sql.gz"
    exit 1
fi

BACKUP_FILE="$1"
[[ "$BACKUP_FILE" != *"/"* ]] && BACKUP_FILE="$BACKUP_DIR/$BACKUP_FILE"

# Проверка файла
if [ ! -f "$BACKUP_FILE" ]; then
    error "Файл не найден: $BACKUP_FILE"
    BASENAME=$(basename "$BACKUP_FILE" .gz)
    if find "$BACKUP_DIR" -name "${BASENAME}*" | grep -q .; then
        info "Возможно вы имели в виду:"
        find "$BACKUP_DIR" -name "${BASENAME}*" -type f | sort -r
    fi
    exit 1
fi

# Проверка целостности
info "Проверка бекапа..."
if ! gzip -t "$BACKUP_FILE" 2>/dev/null; then
    error "Бекап повреждён"
    exit 1
fi
success "Бекап в порядке"

# Проверка контейнера
if ! podman ps --format "table {{.Names}}" | grep -q "^$CONTAINER_NAME$"; then
    error "Контейнер $CONTAINER_NAME не запущен"
    info "Запустите: podman-compose up -d postgres"
    exit 1
fi

# Подтверждение
echo ""
warning "╔════════════════════════════════════╗"
warning "║ ВНИМАНИЕ! ДАННЫЕ БУДУТ УДАЛЕНЫ!    ║"
warning "╚════════════════════════════════════╝"
echo ""
read -p "Продолжить? (y/N): " -n 1 -r
echo
[[ ! $REPLY =~ ^[Yy]$ ]] && { info "Отменено"; exit 0; }

# Остановка бота
info "Останавливаем бота..."
podman-compose stop "$BOT_SERVICE" > /dev/null && success "Бот остановлен" || warning "Бот не запущен"

# Пересоздание БД
info "Подготовка БД..."
podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d postgres -c "DROP DATABASE IF EXISTS $DB_NAME;" > /dev/null
podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME;" > /dev/null
success "База пересоздана"

# Восстановление
info "Восстановление данных..."
START_TIME=$(date +%s)
if gunzip -c "$BACKUP_FILE" | podman exec -i "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" > /dev/null; then
    DURATION=$(( $(date +%s) - START_TIME ))
    success "Восстановлено за ${DURATION}с"

    # Проверка
    TABLE_COUNT=$(podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';" | tr -d ' \n')
    [ -n "$TABLE_COUNT" ] && [ "$TABLE_COUNT" -gt 0 ] && success "Таблиц: $TABLE_COUNT"

    USER_COUNT=$(podman exec "$CONTAINER_NAME" psql -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM users;" 2>/dev/null | tr -d ' \n')
    [ -n "$USER_COUNT" ] && success "Пользователей: $USER_COUNT"
else
    error "Ошибка восстановления"
    podman-compose start "$BOT_SERVICE" > /dev/null 2>&1
    exit 1
fi

# Запуск бота
info "Запускаем бота..."
podman-compose up -d "$BOT_SERVICE" > /dev/null && success "Бот запущен" || error "Не удалось запустить бота"

# Финал
echo ""
success "╔══════════════════════════╗"
success "║ ВОССТАНОВЛЕНИЕ ЗАВЕРШЕНО ║"
success "╚══════════════════════════╝"
echo ""
info "📁 Файл: $(basename "$BACKUP_FILE")"
info "📏 Размер: $(du -h "$BACKUP_FILE" | cut -f1)"
info "⏱️ Время: ${DURATION}с"
info "🗃️ Таблиц: $TABLE_COUNT"
[ -n "$USER_COUNT" ] && info "👥 Пользователей: $USER_COUNT"
echo ""
info "🌐 Бот будет готов через несколько секунд"
