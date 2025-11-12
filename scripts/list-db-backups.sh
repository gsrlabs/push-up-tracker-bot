#!/bin/bash

# Конфигурация
BACKUP_DIR="./backups/db"

echo "🗄️  Доступные бекапы базы данных:"
echo ""

# Проверяем есть ли бекапы
if [ ! -d "$BACKUP_DIR" ] || [ -z "$(ls -A "$BACKUP_DIR" 2>/dev/null)" ]; then
    echo "❌ Бекапы не найдены в директории: $BACKUP_DIR"
    echo "💡 Создайте бекап: ./scripts/backup.sh"
    exit 1
fi

# Простой вывод через ls
echo "Бекапы (последние 20):"
ls -lt "$BACKUP_DIR"/*.gz 2>/dev/null | head -20 | awk '{ 
    if(NR>1) {
        date = $6 " " $7 " " $8
        file = $9
        printf "📅 %s | 📁 %s\n", date, file
    }
}'

total_count=$(ls "$BACKUP_DIR"/*.gz 2>/dev/null | wc -l)

# Получаем последний бекап для примера
latest_backup=$(ls -t "$BACKUP_DIR"/*.gz 2>/dev/null | head -1)
latest_filename=$(basename "$latest_backup" 2>/dev/null)

echo ""
echo "📊 Всего бекапов: $total_count"
echo ""
echo "💡 Для восстановления: ./scripts/restore-db.sh <имя_файла>"

if [ -n "$latest_filename" ]; then
    echo "💡 Пример: ./scripts/restore-db.sh $latest_filename"
else
	echo "💡 Пример: "
    echo "./scripts/restore-db.sh pushup_tracker_20241215_143022.sql.gz"
fi