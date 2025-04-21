-- 004_user_comments.up.sql

-- Add user_id column to comments table
ALTER TABLE comments
    ADD COLUMN user_id INT;

-- Add foreign key constraint to reference users(id)
ALTER TABLE comments
    ADD CONSTRAINT fk_user
        FOREIGN KEY (user_id)
            REFERENCES users(id)
            ON DELETE SET NULL;