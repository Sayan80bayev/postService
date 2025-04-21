-- 004_user_comments.down.sql

-- Drop foreign key constraint for user_id
ALTER TABLE comments
DROP CONSTRAINT IF EXISTS fk_user;

-- Remove the user_id column from comments table
ALTER TABLE comments
DROP COLUMN IF EXISTS user_id;