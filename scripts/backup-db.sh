#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups/db"
DATE=$(date +%Y%m%d_%H%M%S)
DB_NAME="pushup_tracker"
BINARY_NAME="trackerbot"

# Создаем директории
mkdir -p $BACKUP_DIR
mkdir -p $BINARY_BACKUP_DIR

echo "🔄 Создание бекапа..."

# 1. Бекап базы данных
echo "📦 Бекап базы данных..."
docker-compose exec -T postgres pg_dump -U pushup_user -d $DB_NAME > $BACKUP_DIR/${DB_NAME}_${DATE}.sql

# Сжимаем бекап БД
gzip $BACKUP_DIR/${DB_NAME}_${DATE}.sql
echo "✅ Бекап БД создан: $BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz"

# Статистика
echo ""
echo "📊 Бекап создан:"
echo "   🗄️  База данных: $BACKUP_DIR/${DB_NAME}_${DATE}.sql.gz"
