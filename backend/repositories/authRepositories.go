package repositories

// authRepositories
// Handles database interactions used by authControllers

import (
	"context"

	"battlebarge/db"
	"battlebarge/models"
)

// CreateUser inserts a new user row into the users table
//
// Arguments: user (models.User) - user record to insert
//
// Returns: error - non-nil if the insert fails (e.g. duplicate username)
func CreateUser(user models.User) error {
	query := `
		INSERT INTO users (id, email, username, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := db.PGClient.Exec(
		context.Background(),
		query,
		user.ID,
		user.Email,
		user.Username,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

// GetUserByID fetches a single user from the users table by ID
//
// Arguments: id (string) - Firebase UID of the user
//
// Returns: models.User - the matching user; error - pgx.ErrNoRows if not found or on query failure
func GetUserByID(id string) (models.User, error) {
	query := `
		SELECT id, email, username, created_at, updated_at 
		FROM users 
		WHERE id = $1
	`

	var user models.User
	err := db.PGClient.QueryRow(context.Background(), query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return user, err
}
