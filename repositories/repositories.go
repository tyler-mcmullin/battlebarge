package repositories

import (
	"context"
	"encoding/json"

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

func CreateWarband(warband models.Warband) error {
	unitsJSON, err := json.Marshal(warband.Units)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO warbands (
			id, user_id, name, faction, description, units, num_units,
			total_points_cost, crusade_points, requisition_points,
			supply_limit, supply_cost, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`

	_, err = db.PGClient.Exec(
		context.Background(),
		query,
		warband.ID,
		warband.UserID,
		warband.Name,
		warband.Faction,
		warband.Description,
		unitsJSON,
		warband.NumUnits,
		warband.TotalPointsCost,
		warband.CrusadePoints,
		warband.RequisitionPoints,
		warband.SupplyLimit,
		warband.SupplyCost,
		warband.CreatedAt,
		warband.UpdatedAt,
	)

	return err
}
