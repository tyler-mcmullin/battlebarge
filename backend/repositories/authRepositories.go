package repositories

// authRepositories
// Handles database interactions used by authControllers

import (
	"context"

	"github.com/google/uuid"

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
		INSERT INTO users (id, firebase_id, email, username, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := db.PGClient.Exec(
		context.Background(),
		query,
		user.ID,
		user.FirebaseId,
		user.Email,
		user.Username,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}

// GetUserByID fetches a single user from the users table by ID
//
// Arguments: id (string) - the user's own UUID (not the Firebase UID)
//
// Returns: models.User - the matching user; error - pgx.ErrNoRows if not found or on query failure
func GetUserByID(id string) (models.User, error) {
	query := `
		SELECT id, firebase_id, email, username, created_at, updated_at 
		FROM users 
		WHERE id = $1
	`

	var user models.User
	err := db.PGClient.QueryRow(context.Background(), query, id).Scan(
		&user.ID,
		&user.FirebaseId,
		&user.Email,
		&user.Username,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	return user, err
}

// GetUserIDByFirebaseID looks up our own user ID for a verified Firebase UID
//
// Arguments: firebaseID (string) - the UID from a verified Firebase token
//
// Returns: string - the user's UUID; error - pgx.ErrNoRows if no user has that Firebase UID
func GetUserIDByFirebaseID(firebaseID string) (string, error) {
	var id uuid.UUID
	err := db.PGClient.QueryRow(context.Background(),
		`SELECT id FROM users WHERE firebase_id = $1`, firebaseID).Scan(&id)
	if err != nil {
		return "", err
	}
	return id.String(), nil
}
