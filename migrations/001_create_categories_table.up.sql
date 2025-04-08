CREATE TABLE IF NOT EXISTS categories (
                                          id SERIAL PRIMARY KEY,
                                          name VARCHAR(255) NOT NULL UNIQUE,
                                          created_at TIMESTAMP,
                                          updated_at TIMESTAMP,
                                          deleted_at TIMESTAMP
);