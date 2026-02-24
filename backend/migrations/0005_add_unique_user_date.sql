-- migrations/0005_add_unique_user_date.sql
-- +goose Up

ALTER TABLE pushups
ADD CONSTRAINT unique_user_date UNIQUE (user_id, date);

-- +goose Down

ALTER TABLE pushups
DROP CONSTRAINT IF EXISTS unique_user_date;