package repositories

// Repositories package handles database insertions and queries
// Functions called by controllers as needed

import (
	"context"
	"encoding/json"

	"battlebarge/db"
	"battlebarge/models"
)

// USER FUNCTIONS

// DB Insertions
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

// DB Queries
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

// DATA FUNCTIONS

// Data Insertions
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

// Data Queries

func GetAllWarbands(id string) ([]models.Warband, error) {
	query := `
		SELECT id, user_id, name, faction, description, units,
		       num_units, total_points_cost, crusade_points,
		       requisition_points, supply_limit, supply_cost,
		       created_at, updated_at
		FROM warbands
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := db.PGClient.Query(context.Background(), query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	warbands := []models.Warband{}

	for rows.Next() {
		var w models.Warband
		var unitsJSON []byte

		err := rows.Scan(
			&w.ID, &w.UserID, &w.Name, &w.Faction, &w.Description, &unitsJSON,
			&w.NumUnits, &w.TotalPointsCost, &w.CrusadePoints,
			&w.RequisitionPoints, &w.SupplyLimit, &w.SupplyCost,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(unitsJSON, &w.Units); err != nil {
			return nil, err
		}

		warbands = append(warbands, w)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return warbands, nil
}

func GetWarbandByID(id string) (models.Warband, error) {
	var w models.Warband
	var unitsJSON []byte

	query := `
		SELECT id, user_id, name, faction, description, units,
		       num_units, total_points_cost, crusade_points,
		       requisition_points, supply_limit, supply_cost,
		       created_at, updated_at
		FROM warbands
		WHERE id = $1
	`

	err := db.PGClient.QueryRow(context.Background(), query, id).Scan(
		&w.ID, &w.UserID, &w.Name, &w.Faction, &w.Description, &unitsJSON,
		&w.NumUnits, &w.TotalPointsCost, &w.CrusadePoints,
		&w.RequisitionPoints, &w.SupplyLimit, &w.SupplyCost,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return models.Warband{}, err
	}

	if err := json.Unmarshal(unitsJSON, &w.Units); err != nil {
		return models.Warband{}, err
	}

	return w, nil
}
