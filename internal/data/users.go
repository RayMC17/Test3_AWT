package data

import (
	//"context"
	//"errors"
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/RayMC17/bookclub-api/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

var ErrUserNotFound = errors.New("user not found")

// User represents a user in the book club system.
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Activated bool      `json:"activated"`
	Version   int       `json:"-"`
}

type password struct {
	plaintext *string
	hash      []byte
}

// UserModel handles the database interactions for users.
type UserModel struct {
	DB *sql.DB
}

var AnonymouseUser = &User{}

// validation for the email address
func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

// check for password to be valid
func ValidatePassword(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 7, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "mustnot be more than 72 bytes long")
}

// validate username
func ValidateUser(v *validator.Validator, user *User) {
	v.Check(user.Username != "", "username", "must be provided")
	v.Check(len(user.Username) <= 200, "username", "must not be more than 200 bytes long")

	//validate user for email
	ValidateEmail(v, user.Email)
	//validate the plain text email
	if user.Password.plaintext != nil {
		ValidatePassword(v, *user.Password.plaintext)
	}

	//check if we messed up in our codebase
	if user.Password.hash == nil {
		panic("missing password hash for user")
	}
}

// check if current user is anonymous
func (u *User) IsAnonymous() bool {
	return u == AnonymouseUser
}

func (p *password) Set(plainTextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainTextPassword), 12)
	if err != nil {
		return err
	}

	p.plaintext = &plainTextPassword
	p.hash = hash

	return nil
}

// Insert adds a new user to the database.
func (m *UserModel) Insert(user *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, activated)
		VALUES ($1, $2, $3, $4)
		RETURNING id, createdat, version`
	args := []any{user.Username, user.Email, user.Password.hash, user.Activated}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.Version)

	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return ErrDuplicateEmail
		default:
			return err
		}
	}
	return nil
}

// Get retrieves a user by their ID.
func (m *UserModel) Get(id int) (*User, error) {
	query := `
		SELECT id, username, email, createdat
		FROM users
		WHERE id = $1`

	var user User
	err := m.DB.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrRecordNotFound
	}
	return &user, err
}

// Update modifies an existing user in the database.
func (m *UserModel) Update(user *User) error {
	query := `
    UPDATE users
    SET username = $1, email = $2, password_hash = $3, activated = $4, version = version + 1
    WHERE id = $5 AND version = $6
    RETURNING version
`
	fmt.Printf("Executing update with user ID: %d and version: %d\n", user.ID, user.Version)

	args := []interface{}{user.Username, user.Email, user.Password.hash, user.Activated, user.ID, user.Version}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var newVersion int
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&newVersion)
	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique key constraints "users_email_key"`:
			return ErrDuplicateEmail
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConfilct
		default:
			return err
		}
	}

	user.Version = newVersion
	return nil
}

// Delete removes a user by their ID.
func (m *UserModel) Delete(id int) error {
	query := `DELETE FROM users WHERE id = $1`
	_, err := m.DB.Exec(query, id)
	return err
}

func (u *UserModel) GetForToken(tokenScope, tokenPlainText string) (*User, error) {
	tokenHash := sha256.Sum256([]byte(tokenPlainText))

	query := `
	SELECT users.id, users.createdat, users.username, users.email, users.password_hash , users.activated, users.version
	FROM users
	INNER JOIN tokens
	ON users.id = tokens.user_id
	WHERE tokens.hash = $1
	AND tokens.scope = $2
	AND tokens.expiry > $3
	`

	args := []any{tokenHash[:], tokenScope, time.Now()}

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := u.DB.QueryRowContext(ctx, query, args...).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)

	if err != nil {
		println(err.Error())
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	//return the correct user
	return &user, nil
}

func (u UserModel) GetByEmail(email string) (*User, error) {
	query := `
	SELECT id, createdat, username, email, password_hash, activated, version
	FROM users
	WHERE email = $1
   `
	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := u.DB.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Email,
		&user.Password.hash,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (u *UserModel) GetByID(id int64) (*User, error) {
	query := `
	SELECT id, createdat, username, email, activated, version
	FROM users
	WHERE id = $1
   `
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user User
	err := u.DB.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.Username,
		&user.Email,
		&user.Activated,
		&user.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &user, nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil

		default:
			return false, nil
		}
	}
	return true, nil
}

func (m *UserModel) UserExists(userID int) error {
	query := `
	SELECT id
	FROM users
	WHERE id = $1`

	var user User

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, userID).Scan(
		&user.ID,
	)
	if err != nil {
		return err
	}
	return nil
}
