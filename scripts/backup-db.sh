#!/bin/bash
# Конфигурация
BACKUP_DIR="./backups/db"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="pushup_tracker"
DB_USER="pushup_user"
CONTAINER_NAME="pushup-db"

# Создаём директорию
mkdir -p "$BACKUP_DIR"

echo "🔄 Создание бекапа..."

# 1. Бекап БД через pg_dump
echo "📦 Бекап базы данных..."
if podman exec "$CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" > "$BACKUP_DIR/${DB_NAME}_${DATE}.sql"; then
    # Сжимаем
    gzip "$BACKUP_DIR/${DB_NAME}_${DATE}.sql"
    echo "✅ Бекап БД создан: $BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz"
else
    echo "❌ Ошибка при создании бекапа"
    exit 1
fi

# Статистика
echo ""
echo "📊 Бекап создан:"
echo " 🗄️ База данных: $BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz"
echo " 📏 Размер: $(du -h "$BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz" | cut -f1)"
