package repositories

import (
	"context"

	"battlebarge/db"
	"battlebarge/models"
)

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
