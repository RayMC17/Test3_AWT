package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"time"

	"github.com/RayMC17/bookclub-api/internal/validator"
	//"golang.org/x/crypto/bcrypt"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// purpose of the token
const ScopeActivation = "Activation"
const ScopeAuthentication = "Authentication"
const ScopePasswordReset = "password-reset"

// token definition
type Token struct {
	PlainText string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int       `json:"-"`
	Expiry    time.Time `json:"expiry"`
	Scope     string    `json:"-"`
}

// database access
type TokenModel struct {
	DB *sql.DB
}

// generate a token for the user
func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: int(userID),
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	//generating the acctual token. creating a byte slice and filling it with random values (rand.read)
	randoBytes := make([]byte, 16)
	_, err := rand.Read(randoBytes)
	if err != nil {
		return nil, err
	}

	//encoding the random bytes useing base-32
	token.PlainText = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randoBytes)

	//hash the encoding
	hash := sha256.Sum256([]byte(token.PlainText))
	token.Hash = hash[:] //array to slice conversion

	return token, nil
}

// validate the token client sent to us to be 26 bytes long
func ValidatetokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

// create and return new token, uses insert as a helper method
func (t *TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = t.Insert(token)
	return token, err
}

func (t *TokenModel) Insert(token *Token) error {
	query := `
	INSERT INTO tokens (hash, user_id, expiry, scope)
	VALUES ($1, $2, $3, $4)
	`

	args := []any{token.Hash, token.UserID, token.Expiry, token.Scope}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, args...)
	return err
}

// delete token based of the type and the user
func (t *TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
	DELETE FROM tokens
	WHERE scope = $1 AND user_id = $2
	`

	args := []any{scope, userID}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := t.DB.ExecContext(ctx, query, args...)
	return err
}

// Validate checks if a token is valid and returns the associated user ID if it is.
func (m TokenModel) Validate(plainText string, scope string) (int64, error) {
	var token Token

	query := `
		SELECT hash, user_id, expiry
		FROM tokens
		WHERE hash = $1
		AND scope = $2
		AND expiry > NOW()
	`

	// Hash the plaintext token
	hashedToken := sha256.Sum256([]byte(plainText))

	// Fetch token from the database
	err := m.DB.QueryRow(query, hashedToken[:], scope).Scan(
		&token.Hash,
		&token.UserID,
		&token.Expiry,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidToken
		}
		return 0, err
	}

	return int64(token.UserID), nil
}
