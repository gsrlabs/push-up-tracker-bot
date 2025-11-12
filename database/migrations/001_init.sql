-- Миграция для существующих данных
UPDATE users 
SET last_updated_max_reps = COALESCE(
    (SELECT MAX(date) FROM max_reps_history WHERE max_reps_history.user_id = users.user_id),
    last_updated
) 
WHERE last_updated_max_reps IS NULL;
