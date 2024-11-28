CREATE TABLE IF NOT EXISTS books (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    authors VARCHAR(70)[] NOT NULL,
    isbn VARCHAR(20) UNIQUE NOT NULL,
    publication_date DATE,
    genre VARCHAR(50),
    description TEXT,
    average_rating DECIMAL(3,2) CHECK (average_rating BETWEEN 0 AND 5) DEFAULT 0.0
);
