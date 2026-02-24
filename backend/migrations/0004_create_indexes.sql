-- migrations/0004_create_indexes.sql
-- +goose Up

-- pushups
CREATE INDEX idx_pushups_user ON pushups(user_id);
CREATE INDEX idx_pushups_date ON pushups(date);
CREATE INDEX idx_pushups_user_date ON pushups(user_id, date);
CREATE INDEX idx_pushups_date_user ON pushups(date, user_id);

-- частичный индекс для первой тренировки
CREATE INDEX idx_pushups_user_date_positive
ON pushups(user_id, date)
WHERE count > 0;

-- max_reps_history
CREATE INDEX idx_max_reps_history_user_date
ON max_reps_history(user_id, date DESC);

CREATE INDEX idx_max_reps_history_user_max_reps
ON max_reps_history(user_id, max_reps DESC, date DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_pushups_user;
DROP INDEX IF EXISTS idx_pushups_date;
DROP INDEX IF EXISTS idx_pushups_user_date;
DROP INDEX IF EXISTS idx_pushups_date_user;
DROP INDEX IF EXISTS idx_pushups_user_date_positive;

DROP INDEX IF EXISTS idx_max_reps_history_user_date;
DROP INDEX IF EXISTS idx_max_reps_history_user_max_reps;