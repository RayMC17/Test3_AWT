package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/RayMC17/bookclub-api/internal/validator"
)

var ErrNoRecord = errors.New("record not found")

// Review represents a review for a book.
type Review struct {
	ID        int64     `json:"id"`
	BookID    int64     `json:"book_id"`
	AuthorID  int       `json:"user_id"`
	Rating    int       `json:"rating"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Version   int       `json:"version"`
}

// ReviewModel wraps a SQL database connection pool.
type ReviewModel struct {
	DB *sql.DB
}

// ValidateReview validates the review data.
func ValidateReview(v *validator.Validator, review *Review) {

	v.Check(review.Rating >= 1 && review.Rating <= 5, "rating", "must be between 1 and 5")

	v.Check(review.Content != "", "content", "must be provided")
	v.Check(len(review.Content) <= 1000, "content", "must not be more than 1000 characters long")
}

// Insert adds a new review to the database.
func (m *ReviewModel) Insert(review *Review) error {
	query := `
        INSERT INTO boo_reviews (book_id, user_id, rating, review_text)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, version`

	args := []interface{}{review.BookID, review.AuthorID, review.Rating, review.Content}

	return m.DB.QueryRow(query, args...).Scan(&review.ID, &review.CreatedAt, &review.Version)
}

// Get retrieves a specific review by ID.
func (m *ReviewModel) Get(id int64) (*Review, error) {
	query := `
        SELECT id, book_id, user_id, rating, review_text, created_at, version
        FROM boo_reviews
        WHERE id = $1`

	var review Review

	err := m.DB.QueryRow(query, id).Scan(
		&review.ID,
		&review.BookID,
		&review.AuthorID,
		&review.Rating,
		&review.Content,
		&review.CreatedAt,
		&review.Version,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNoRecord
	} else if err != nil {
		return nil, err
	}

	return &review, nil
}

// Update modifies the data of a specific review.
func (m *ReviewModel) Update(review *Review) error {
	query := `
        UPDATE boo_reviews
        SET rating = $1, review_text = $2, version = version+1
        WHERE id = $3
        RETURNING version`

	args := []interface{}{review.Rating, review.Content, review.ID}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&review.Version)
}

// Delete removes a specific review from the database.
func (m *ReviewModel) Delete(id int64) error {
	query := `
        DELETE FROM boo_reviews
        WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	//check if any rows affected
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound //no rows affected
	}
	return nil
}

// GetAll retrieves all reviews for a specific book with optional filters for pagination and sorting.
func (m *ReviewModel) GetAll(bookID int64) ([]*Review, error) {
	query := `
        SELECT id, book_id, user_id, rating, review_text, created_at, version
        FROM boo_reviews
        WHERE book_id = $1
    `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []*Review{}

	for rows.Next() {
		var review Review
		err := rows.Scan(
			&review.ID,
			&review.BookID,
			&review.AuthorID,
			&review.Rating,
			&review.Content,
			&review.CreatedAt,
			&review.Version,
		)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, &review)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return reviews, nil
}

// Helper function to safely format the SQL query with sort options.
// func formatQuery(query, sortColumn, sortDirection string) string {
// 	return fmt.Sprintf(query, sortColumn, sortDirection)
// }

func (m *ReviewModel) GetAllByUser(userID int64) ([]*Review, error) {
	query := `
        SELECT id, book_id, user_id, review_text, rating, version
        FROM boo_reviews
        WHERE user_id = $1
    `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := []*Review{}

	for rows.Next() {
		var review Review
		err := rows.Scan(
			&review.ID,
			&review.BookID,
			&review.AuthorID,
			&review.Content,
			&review.Rating,
			&review.Version,
		)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, &review)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return reviews, nil
}
