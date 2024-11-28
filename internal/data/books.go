package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	//"strings"
	"time"

	"github.com/RayMC17/bookclub-api/internal/validator"
	"github.com/lib/pq" // Update the path according to your module path
)

var ErrRecordNotFound = errors.New("record not found")

// Book model definition
type Book struct {
	ID              int       `json:"id"`
	Title           string    `json:"title"`
	Authors         []string  `json:"authors"`
	ISBN            string    `json:"isbn"`
	PublicationDate time.Time `json:"publication_date"`
	Genre           string    `json:"genre"`
	Description     string    `json:"description"`
	AverageRating   float64   `json:"average_rating"`
}

// // ReadingList model definition
// type ReadingList struct {
//     ID          int      `json:"id"`
//     Name        string   `json:"name"`
//     Description string   `json:"description"`
//     Books       []int    `json:"books"`
//     Status      string   `json:"status"`
// }

// BookModel struct and methods
type BookModel struct {
	DB *sql.DB
}

func ValidateBook(v *validator.Validator, book *Book) {
	v.Check(book.Title != "", "title", "must be provided")
	v.Check(len(book.Title) <= 255, "title", "must not be more than 255 characters long")
	v.Check(len(book.Authors) > 0, "authors", "must have at least one author")
	v.Check(book.ISBN != "", "isbn", "must be provided")
	v.Check(len(book.ISBN) == 13, "isbn", "must be exactly 13 characters long")
	v.Check(book.PublicationDate.Before(time.Now()), "publication_date", "must be in the past")
	v.Check(book.Genre != "", "genre", "must be provided")
	v.Check(len(book.Genre) <= 50, "genre", "must not be more than 50 characters long")
	v.Check(len(book.Description) <= 1000, "description", "must not be more than 1000 characters long")
	v.Check(book.AverageRating >= 0 && book.AverageRating <= 5, "average_rating", "must be between 0 and 5")
}

// BookModel methods (Insert, Get, Update, Delete, GetAll) as defined in your code
// Insert a new book
func (m *BookModel) Insert(book *Book) error {
	//authors := strings.Join(book.Authors, ",")
	query := `
        INSERT INTO books (title, authors, isbn, publication_date, genre, description, average_rating)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id`
	args := []interface{}{book.Title, pq.Array(book.Authors), book.ISBN, book.PublicationDate, book.Genre, book.Description, book.AverageRating}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&book.ID)
}

// Get a single book by ID
func (m *BookModel) Get(id int) (*Book, error) {
	query := `
        SELECT id, title, authors, isbn, publication_date, genre, description, average_rating
        FROM books
        WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var book Book
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&book.ID, &book.Title, pq.Array(&book.Authors), &book.ISBN,
		&book.PublicationDate, &book.Genre, &book.Description, &book.AverageRating,
	)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	}
	return &book, err
}

// Update a book
func (m *BookModel) Update(book *Book) error {
	query := `
        UPDATE books
        SET title = $1, authors = $2, isbn = $3, publication_date = $4, genre = $5, description = $6, average_rating = $7
        WHERE id = $8`
	args := []interface{}{book.Title, pq.Array(book.Authors), book.ISBN, book.PublicationDate, book.Genre, book.Description, book.AverageRating, book.ID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...)
	return err.Err()
}

// Delete a book by ID
func (m *BookModel) Delete(id int) error {
	query := `DELETE FROM books WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id)
	return err.Err()
}

// GetAll retrieves all books with optional filters and pagination.
func (m *BookModel) GetAll(filters Filters) ([]*Book, Metadata, error) {
	query := fmt.Sprintf(`
        SELECT COUNT(*) OVER(), id, title, authors, isbn, publication_date, genre, description, average_rating
        FROM books
        ORDER BY %s %s, id ASC
        LIMIT $1 OFFSET $2`, filters.SortColumn(), filters.SortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, filters.Limit(), filters.Offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	books := []*Book{}

	for rows.Next() {
		var book Book
		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.Title,
			pq.Array(&book.Authors),
			&book.ISBN,
			&book.PublicationDate,
			&book.Genre,
			&book.Description,
			&book.AverageRating,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		books = append(books, &book)
	}

	err = rows.Err()
	if err != nil {
		return nil, Metadata{}, err
	}

	metadata := CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}

// GetAll retrieves all books with optional filters and pagination.
func (m *BookModel) GetAllFilters(title string, author string, genre string, filters Filters) ([]*Book, Metadata, error) {
	query := fmt.Sprintf(`
	SELECT COUNT(*) OVER(), id, title, authors, isbn, publication_date, genre, description, average_rating
	FROM books
	WHERE (title ILIKE '%%' || $1 || '%%' OR $1 = '')
  		OR (genre ILIKE '%%' || $2 || '%%' OR $2 = '')
  		OR ($3 = '' OR EXISTS (
        	SELECT 1 FROM unnest(authors) author WHERE author ILIKE '%%' || $3 || '%%'
      	))
	ORDER BY %s %s, id ASC
	LIMIT $4 OFFSET $5
`, filters.SortColumn(), filters.SortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, title, genre, author, filters.Limit(), filters.Offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	books := []*Book{}

	for rows.Next() {
		var book Book
		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.Title,
			pq.Array(&book.Authors),
			&book.ISBN,
			&book.PublicationDate,
			&book.Genre,
			&book.Description,
			&book.AverageRating,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		books = append(books, &book)
	}

	err = rows.Err()
	if err != nil {
		return nil, Metadata{}, err
	}

	metadata := CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return books, metadata, nil
}

func (m *BookModel) BookExists(id int) error {
	query := `
        SELECT id
        FROM books
        WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var book Book
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&book.ID,
	)
	if err != nil {
		return err
	}
	return nil
}
