package data

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/RayMC17/bookclub-api/internal/validator"
)

// ReadingList represents a reading list in the book club system.
type ReadingList struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   int       `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	Version     int       `json:"version"`
}

type BookINlist struct {
	ListNameID int    `json:"list_name_id"`
	BookID     int    `json:"book_id"`
	Status     string `json:"status"`
	Version    int    `json:"version"`
}

// ReadingListModel handles the database interactions for reading lists.
type ReadingListModel struct {
	DB *sql.DB
}

// Insert a new reading list
func (m *ReadingListModel) CreateReadingList(list *ReadingList) error {
	query := `
		INSERT INTO lists_names (name, description, created_by)
		VALUES ($1, $2, $3)
		RETURNING id, version
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []interface{}{list.Name, list.Description, list.CreatedBy}

	return m.DB.QueryRowContext(ctx, query, args...).Scan(
		&list.ID,
		&list.Version,
	)

}

// Get a single reading list by ID
func (m *ReadingListModel) Get(id int) (*ReadingList, error) {

	query := `
		SELECT id, name, description, created_by, created_at, version
		FROM lists_names
		WHERE id = $1`

	var list ReadingList

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&list.ID, &list.Name, &list.Description, &list.CreatedBy, &list.CreatedAt, &list.Version,
	)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}

	return &list, nil
}

// Update an existing reading list
func (m *ReadingListModel) Update(list *ReadingList) error {
	query := `
		UPDATE lists_names
		SET name = $1, description = $2, version = version+1
		WHERE id = $3 AND version = $4
		RETURNING version`
	args := []interface{}{list.Name, list.Description, list.ID, list.Version}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(
		&list.Version,
	)

}

// Delete a reading list by ID
func (m *ReadingListModel) Delete(id int) error {
	query := `DELETE FROM lists_names WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (m *ReadingListModel) AddBook(bookForList *BookINlist) error {
	query := `
	    INSERT INTO book_lists (list_name, book_id, status)
	    VALUES ($1, $2, $3)
		RETURNING version
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{bookForList.ListNameID, bookForList.BookID, bookForList.Status}

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&bookForList.Version,
	)
	return err
}

func (m *ReadingListModel) RemoveBook(readingListID int, bookID int) error {
	query := `
        DELETE FROM book_lists
        WHERE list_name = $1 AND book_id = $2
    `
	result, err := m.DB.Exec(query, readingListID, bookID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// GetAll retrieves all reading lists based on the filters.
func (m *ReadingListModel) GetAll(filters Filters) ([]*ReadingList, Metadata, error) {
	query := fmt.Sprintf(`
        SELECT COUNT(*) OVER(), id, name, description, created_by, created_at, version
        FROM lists_names
        ORDER BY %s %s
        LIMIT $1 OFFSET $2`, filters.SortColumn(), filters.SortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, filters.Limit(), filters.Offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	var readingLists []*ReadingList

	for rows.Next() {
		var readingList ReadingList
		err := rows.Scan(
			&totalRecords,
			&readingList.ID,
			&readingList.Name,
			&readingList.Description,
			&readingList.CreatedBy,
			&readingList.CreatedAt,
			&readingList.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		readingLists = append(readingLists, &readingList)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return readingLists, metadata, nil
}

func (m *ReadingListModel) GetAllByUser(userID int64) ([]*ReadingList, error) {
	query := `
        SELECT id, name, description, created_at, created_by, version
        FROM lists_names
        WHERE created_by = $1
	`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	readingLists := []*ReadingList{}

	for rows.Next() {
		var list ReadingList
		err := rows.Scan(
			&list.ID,
			&list.Name,
			&list.Description,
			&list.CreatedAt,
			&list.CreatedBy,
			&list.Version,
		)
		if err != nil {
			return nil, err
		}
		readingLists = append(readingLists, &list)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return readingLists, nil
}

// ValidateReadingList function to validate ReadingList fields
func ValidateReadingList(v *validator.Validator, readingList *ReadingList) {
	v.Check(readingList.Name != "", "name", "must be provided")
	v.Check(len(readingList.Name) <= 255, "name", "must not be more than 255 characters long")
	v.Check(readingList.Description != "", "description", "must be provided")
	v.Check(len(readingList.Description) <= 1000, "description", "must not be more than 1000 characters long")

}

func ValidateBookInList(v *validator.Validator, status string) {
	v.Check(status != "", "status", "must be provided")
	v.Check(status == "currently Reading" || status == "completed", "status", "status values must either be 'currently reading' or 'completed'")
}

func (m *ReadingListModel) ReadingListExist(id int) error {
	query := `
	SELECT id
	FROM lists_names
	WHERE id = $1`

	var list ReadingList

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&list.ID,
	)
	if err != nil {
		return err
	}
	return nil
}
