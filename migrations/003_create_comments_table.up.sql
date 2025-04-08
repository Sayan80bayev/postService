CREATE TABLE IF NOT EXISTS comments (
                          id SERIAL PRIMARY KEY,
                          post_id INT,
                          content TEXT,
                          created_at TIMESTAMP,
                          updated_at TIMESTAMP,
                          deleted_at TIMESTAMP,
                          CONSTRAINT fk_post
                              FOREIGN KEY (post_id)
                                  REFERENCES posts(id)
                                  ON DELETE CASCADE
);