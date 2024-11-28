CREATE TABLE IF NOT EXISTS boo_reviews (
    id bigserial PRIMARY KEY,
    book_id INT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE, 
    rating INT CHECK(rating BETWEEN 1 AND 5),
    review_text TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, 
    version integer NOT NULL DEFAULT 1
);