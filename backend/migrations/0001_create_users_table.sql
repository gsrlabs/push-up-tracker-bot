-- migrations/0001_create_users_table.sql
-- +goose Up

CREATE TABLE users (
    user_id BIGINT PRIMARY KEY,
    username VARCHAR(100) NOT NULL DEFAULT '',
    max_reps INT NOT NULL DEFAULT 0,
    daily_norm INT NOT NULL DEFAULT 40,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_updated_max_reps TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS users;






