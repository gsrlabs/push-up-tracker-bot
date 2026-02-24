-- migrations/0003_create_max_reps_history_table.sql
-- +goose Up
CREATE TABLE max_reps_history (
    record_id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
    date DATE NOT NULL,
    max_reps INT NOT NULL DEFAULT 0,
    UNIQUE(user_id, date)
);

-- +goose Down
DROP TABLE IF EXISTS max_reps_history;
