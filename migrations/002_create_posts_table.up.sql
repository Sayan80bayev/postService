CREATE TABLE IF NOT EXISTS posts (
                       id SERIAL PRIMARY KEY,
                       title TEXT,
                       content TEXT,
                       category_id INT NOT NULL DEFAULT -1,
                       user_id INT,
                       image_url TEXT,
                       like_count BIGINT DEFAULT 0,
                       created_at TIMESTAMP,
                       updated_at TIMESTAMP,
                       deleted_at TIMESTAMP,
                       CONSTRAINT fk_category
                           FOREIGN KEY (category_id)
                               REFERENCES categories(id)
                               ON DELETE CASCADE
);