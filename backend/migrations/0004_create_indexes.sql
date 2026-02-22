-- migrations/0003_create_max_reps_history_table.sql
-- +goose Up
-- users
CREATE INDEX idx_pushups_user_date ON pushups(user_id, date);
CREATE INDEX idx_pushups_date ON pushups(date);
-- max_reps_history
CREATE INDEX idx_max_reps_history_user_date ON max_reps_history(user_id, date); 
CREATE INDEX idx_max_reps_history_date ON max_reps_history(date);

-- +goose Down
DROP INDEX IF EXISTS idx_pushups_user_date;
DROP INDEX IF EXISTS idx_pushups_date;
DROP INDEX IF EXISTS idx_max_reps_history_user_date;
DROP INDEX IF EXISTS idx_max_reps_history_date;