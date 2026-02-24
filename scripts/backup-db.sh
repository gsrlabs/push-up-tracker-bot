#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="pushup_tracker"
DB_USER="pushup_user"
CONTAINER_NAME="pushup-db"
VOLUME_NAME="push-up-tracker-bot_postgres_data"  # Имя volume (обычно <project>_postgres_data)

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

success() { echo -e "${GREEN}✅ $1${NC}"; }
error() { echo -e "${RED}❌ $1${NC}"; }
info() { echo -e "${BLUE}🔹 $1${NC}"; }
warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }

# Создаём директорию для бекапов
mkdir -p "$BACKUP_DIR"

echo "🔄 Создание бекапа базы данных..."

# Проверяем что контейнер БД запущен
if ! docker ps --format "table {{.Names}}" | grep -q "^$CONTAINER_NAME$"; then
    error "Контейнер БД $CONTAINER_NAME не запущен"
    info "Запустите: docker compose up -d postgres"
    exit 1
fi

# Проверяем существование volume
if ! docker volume ls --format "{{.Name}}" | grep -q "^$VOLUME_NAME$"; then
    warning "Volume $VOLUME_NAME не найден, но контейнер работает"
    info "Продолжаем создание бекапа через pg_dump..."
fi

# 1. Бекап БД через pg_dump (структура и данные)
info "Создание дампа базы данных (структура + данные)..."
BACKUP_FILE="$BACKUP_DIR/${DB_NAME}_full_${DATE}.sql"

if docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" > "$BACKUP_FILE"; then
    # Сжимаем
    gzip "$BACKUP_FILE"
    success "Полный бекап БД создан: $BACKUP_DIR/${DB_NAME}_full_${DATE}.sql.gz"
else
    error "Ошибка при создании полного бекапа"
    rm -f "$BACKUP_FILE"
    exit 1
fi

# 2. Бекап только данных (без схемы) - полезно для переноса
info "Создание дампа только данных (без схемы)..."
DATA_FILE="$BACKUP_DIR/${DB_NAME}_data_${DATE}.sql"

if docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" --data-only > "$DATA_FILE"; then
    gzip "$DATA_FILE"
    success "Бекап данных создан: $BACKUP_DIR/${DB_NAME}_data_${DATE}.sql.gz"
else
    warning "Не удалось создать бекап только данных (продолжаем)"
    rm -f "$DATA_FILE"
fi

# 3. Бекап только схемы (структуры)
info "Создание дампа только схемы..."
SCHEMA_FILE="$BACKUP_DIR/${DB_NAME}_schema_${DATE}.sql"

if docker exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" --schema-only > "$SCHEMA_FILE"; then
    gzip "$SCHEMA_FILE"
    success "Бекап схемы создан: $BACKUP_DIR/${DB_NAME}_schema_${DATE}.sql.gz"
else
    warning "Не удалось создать бекап только схемы (продолжаем)"
    rm -f "$SCHEMA_FILE"
fi

# 4. Создание бекапа самого volume (опционально)
info "Создание бекапа Docker volume..."
if docker run --rm -v "$VOLUME_NAME":/volume -v "$BACKUP_DIR":/backup alpine tar czf "/backup/volume_${DATE}.tar.gz" -C /volume ./; then
    success "Бекап volume создан: $BACKUP_DIR/volume_${DATE}.tar.gz"
else
    warning "Не удалось создать бекап volume (продолжаем)"
fi

# Проверяем целостность основного бекапа
info "Проверка целостности основного бекапа..."
if gzip -t "$BACKUP_DIR/${DB_NAME}_full_${DATE}.sql.gz" 2>/dev/null; then
    success "Основной бекап прошел проверку целостности"
else
    error "Основной бекап поврежден"
    rm -f "$BACKUP_DIR/${DB_NAME}_full_${DATE}.sql.gz"
    exit 1
fi

# Статистика
echo ""
success "📊 Бекапы успешно созданы:"

# Показываем информацию о каждом созданном бекапе
for file in "$BACKUP_DIR"/*"${DATE}"*.gz; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        size=$(du -h "$file" | cut -f1)
        
        case "$filename" in
            *full*)
                info "   🗄️  Полный бекап: $filename ($size)"
                ;;
            *data*)
                info "   📊 Только данные: $filename ($size)"
                ;;
            *schema*)
                info "   📐 Только схема: $filename ($size)"
                ;;
            *volume*)
                info "   💾 Volume бекап: $filename ($size)"
                ;;
        esac
    fi
done

info "   📅 Дата: $(date +"%d.%m.%Y %H:%M")"

# Дополнительная информация о всех бекапах
BACKUP_COUNT=$(find "$BACKUP_DIR" -name "*.gz" -type f | wc -l)
TOTAL_SIZE=$(find "$BACKUP_DIR" -name "*.gz" -type f -exec du -cb {} + | tail -1 | cut -f1)

# Конвертируем размер в человеко-читаемый формат
if command -v numfmt >/dev/null 2>&1; then
    TOTAL_SIZE_HR=$(numfmt --to=iec $TOTAL_SIZE)
else
    TOTAL_SIZE_HR="${TOTAL_SIZE} bytes"
fi

echo ""
info "📈 Статистика всех бекапов:"
info "   Всего бекапов: $BACKUP_COUNT"
info "   Общий размер: $TOTAL_SIZE_HR"

# Создаем символическую ссылку на последний бекап
cd "$BACKUP_DIR"
ln -sf "${DB_NAME}_full_${DATE}.sql.gz" latest_full.sql.gz
ln -sf "volume_${DATE}.tar.gz" latest_volume.tar.gz 2>/dev/null
cd - > /dev/null

success "Ссылки на последние бекапы обновлены:"
info "   latest_full.sql.gz -> последний полный бекап"
info "   latest_volume.tar.gz -> последний бекап volume"

# Информация о восстановлении
echo ""
info "📋 Для восстановления используйте:"
echo ""
echo "   # Восстановить полный бекап:"
echo "   docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME < backup.sql"
echo ""
echo "   # Восстановить из gz:"
echo "   gunzip -c $BACKUP_DIR/${DB_NAME}_full_${DATE}.sql.gz | docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $DB_NAME"
echo ""
echo "   # Восстановить volume:"
echo "   docker run --rm -v $VOLUME_NAME:/volume -v $BACKUP_DIR:/backup alpine sh -c 'rm -rf /volume/* && tar xzf /backup/volume_${DATE}.tar.gz -C /volume'"