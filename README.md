# Telegram‑бот для учёта отжиманий 🏋️‍♂️

## 📝 Описание

Telegram-бот для отслеживания и учета ежедневных отжиманий с интеллектуальной системой расчета дневной нормы. Бот помогает поддерживать регулярные тренировки, мотивирует к достижению целей и предоставляет детальную статистику прогресса.

**Особенность расчета нормы**: Дневная норма рассчитывается согласно рекомендациям ACSM (American College of Sports Medicine) на основе максимального количества отжиманий за один подход, что обеспечивает индивидуальный и безопасный подход к тренировкам.


## 🎯 Функционал

### Главное меню
- **➕ Добавить отжимания** - Ввод количества выполненных отжиманий с отображением прогресса
- **⚙️ Настройки и статистика** - Доступ к дополнительным функциям

### Настройки
- **🎯 Определить норму** - Автоматический расчет дневной нормы на основе вашего максимума
- **📝 Установить норму** - Ручная установка индивидуальной нормы
- **📊 Статистика** - Просмотр личной статистики и общего рейтинга
- **🔔/🔕 Управление напоминаниями** - Включение/отключение уведомлений
- **⬅️ Назад** - Возврат в главное меню

### Дополнительные команды
- **/reset_norm** - Сброс нормы к значению по умолчанию (40 отжиманий)
- **/start** - Перезапуск бота и отображение главного меню


## 🛠 Технический стек

- **Язык**: Go 1.21+
- **Telegram API**: `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- **База данных**: PostgreSQL 14+
- **Драйвер БД**: `github.com/jackc/pgx/v5/pgxpool`
- **Конфигурация**: `.env` файлы через `github.com/joho/godotenv`

## 🗄 Структура базы данных

```sql
-- Основная таблица пользователей
CREATE TABLE users (
    user_id BIGINT PRIMARY KEY,
    username VARCHAR(100) NOT NULL DEFAULT '',
    max_reps INT NOT NULL DEFAULT 0,
    daily_norm INT NOT NULL DEFAULT 40,
    notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Таблица ежедневной статистики
CREATE TABLE pushups (
    record_id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    count INT NOT NULL DEFAULT 0
);

-- Индексы для ускорения запросов
CREATE INDEX idx_pushups_user_date ON pushups(user_id, date);
CREATE INDEX idx_pushups_date ON pushups(date);
```

## ⚡ Параллелизм и надежность

### 1. Горутины и синхронизация
- Каждый входящий запрос обрабатывается в отдельной горутине
- `sync.Map` для кеширования сегодняшней статистики
- `sync.RWMutex` для защиты разделяемых ресурсов

### 2. Таймауты и контексты
```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
```

### 3. Пул соединений PostgreSQL
```go
dbPool.Config().MaxConns = 10
dbPool.Config().MaxConnIdleTime = 30 * time.Minute
```

### 4. Обработка ошибок
- Грейсфул деградейшн при недоступности БД
- Автоматическое отключение напоминаний для недоступных пользователей
- Детальное логирование ошибок

## 🚀 Дополнительные функции

### 1. Кеширование
- In-memory кеш сегодняшней статистики
- Автоматический сброс в полночь
- Оптимизация запросов к базе данных

### 2. Алгоритм расчета нормы
- Умный расчет на основе максимальных повторений
- Учет уровня подготовки пользователя
- Безопасные ограничения (40-250 отжиманий)

### 3. Система напоминаний
- Интеллектуальная отправка в вечернее время (18:00-21:00)
- Проверка доступности пользователей
- Персонализированные сообщения с прогрессом

### 4. Валидация ввода
- Проверка корректности числовых значений
- Защита от отрицательных значений
- Таймауты ввода (10 секунд)

## 📦 Развертывание

### Docker Compose
```yaml
services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_USER: pushup_user
      POSTGRES_PASSWORD: your_password
      POSTGRES_DB: pushup_tracker
    ports:
      - "5433:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./initdb:/docker-entrypoint-initdb.d

volumes:
  postgres-data:
```

### Переменные окружения
```env
TELEGRAM_BOT_TOKEN=your_bot_token_here
DATABASE_URL=postgresql://pushup_user:password@localhost:5433/pushup_tracker?sslmode=disable
```

## 🧪 Тестирование

- Юнит-тесты бизнес-логики
- Интеграционные тесты работы с БД
- Mock-тесты Telegram API
- Тестирование параллельных access

## 📈 Мониторинг

- Логирование запросов и ошибок
- Метрики производительности
- Статистика использования
- Отслеживание активности пользователей

---

**Примечание**: Бот разработан с учетом лучших практик Go, использует чистую архитектуру и готов к масштабированию.
