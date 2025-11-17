-- Основная таблица пользователей
CREATE TABLE users (
    user_id BIGINT PRIMARY KEY,
    username VARCHAR(100) NOT NULL DEFAULT '',
    max_reps INT NOT NULL DEFAULT 0,
    daily_norm INT NOT NULL DEFAULT 40,
    notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated_max_reps TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_notification TIMESTAMP DEFAULT NULL
);

-- Таблица ежедневной статистики
CREATE TABLE pushups (
    record_id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    count INT NOT NULL DEFAULT 0
);

-- История максимальных отжиманий за подход
CREATE TABLE max_reps_history (
    record_id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    max_reps INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, date)
);

-- Индексы для ускорения запросов
CREATE INDEX idx_pushups_user_date ON pushups(user_id, date);
CREATE INDEX idx_pushups_date ON pushups(date);
CREATE INDEX idx_max_reps_history_user_date ON max_reps_history(user_id, date); -- Для быстрого получения истории пользователя
CREATE INDEX idx_max_reps_history_date ON max_reps_history(date); -- Для общих запросов по дате
CREATE INDEX idx_users_max_reps_reminder ON users(last_updated_max_reps)
WHERE notifications_enabled = TRUE AND max_reps > 0 AND max_reps < 100;



