#!/bin/bash

BACKUP_DIR="./backups"

# Цвета
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

if [ ! -d "$BACKUP_DIR" ]; then
    echo "❌ Директория бекапов не найдена: $BACKUP_DIR"
    exit 1
fi

echo "📋 Список всех бекапов:"
echo ""

# Счетчики по типам
full_count=0
data_count=0
schema_count=0
volume_count=0

# Вывод в табличном формате
printf "${BLUE}%-25s %-15s %-10s %s${NC}\n" "Имя файла" "Тип" "Размер" "Дата создания"
printf "%s\n" "----------------------------------------------------------------------"

for file in "$BACKUP_DIR"/*.gz; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        size=$(du -h "$file" | cut -f1)
        
        # Определяем тип
        case "$filename" in
            *full*)
                type="Полный"
                ((full_count++))
                ;;
            *data*)
                type="Данные"
                ((data_count++))
                ;;
            *schema*)
                type="Схема"
                ((schema_count++))
                ;;
            *volume*)
                type="Volume"
                ((volume_count++))
                ;;
            *)
                type="Другое"
                ;;
        esac
        
        # Извлекаем дату из имени файла
        if [[ $filename =~ ([0-9]{8}_[0-9]{6}) ]]; then
            date_str="${BASH_REMATCH[1]}"
            formatted_date=$(date -d "${date_str:0:8} ${date_str:9:2}:${date_str:11:2}:${date_str:13:2}" +"%d.%m.%Y %H:%M" 2>/dev/null || echo "неизвестно")
        else
            formatted_date="неизвестно"
        fi
        
        printf "%-25s %-15s %-10s %s\n" "$filename" "$type" "$size" "$formatted_date"
    fi
done

echo ""
echo "📊 Статистика по типам:"
echo "   🗄️  Полные бекапы: $full_count"
echo "   📊 Бекапы данных: $data_count"
echo "   📐 Бекапы схемы: $schema_count"
echo "   💾 Volume бекапы: $volume_count"

# Проверяем наличие символических ссылок
echo ""
if [ -L "$BACKUP_DIR/latest_full.sql.gz" ]; then
    echo -e "${GREEN}✅ Последний полный бекап: $(readlink $BACKUP_DIR/latest_full.sql.gz)${NC}"
fi
if [ -L "$BACKUP_DIR/latest_volume.tar.gz" ]; then
    echo -e "${GREEN}✅ Последний volume бекап: $(readlink $BACKUP_DIR/latest_volume.tar.gz)${NC}"
fi