CREATE TABLE IF NOT EXISTS lists_names (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_by INT REFERENCES users(id) ON DELETE SET NULL,
    created_at timestamp(0) WITH TIME ZONE NOT NULL DEFAULT NOW(),
    version INT NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS book_lists (
    list_name INT REFERENCES lists_names(id) ON DELETE CASCADE,
    book_id INT REFERENCES books(id) ON DELETE CASCADE,
    status VARCHAR(20) CHECK (status IN ('currently reading', 'completed')) NOT NULL DEFAULT 'currently reading',
    version INT NOT NULL DEFAULT 1,
    PRIMARY KEY (list_name, book_id)
);