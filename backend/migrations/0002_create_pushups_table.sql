-- migrations/0002_create_pushups_table.sql
-- +goose Up
CREATE TABLE pushups (
    record_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    count INT NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS pushups;