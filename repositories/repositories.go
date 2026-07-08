package repositories

// Repositories package handles database insertions and queries
// Functions called by controllers as needed

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

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
	query := `
		INSERT INTO warbands (
			id, user_id, name, faction, description,
			crusade_points, requisition_points,
			supply_limit, supply_cost, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := db.PGClient.Exec(
		context.Background(),
		query,
		warband.ID,
		warband.UserID,
		warband.Name,
		warband.Faction,
		warband.Description,
		warband.CrusadePoints,
		warband.RequisitionPoints,
		warband.SupplyLimit,
		warband.SupplyCost,
		warband.CreatedAt,
		warband.UpdatedAt,
	)

	return err
}

func UpdateWarband(id string, userID string, req models.UpdateWarbandRequest) (models.Warband, error) {
	query := `
		UPDATE warbands
		SET name = COALESCE($1, name),
		    faction = COALESCE($2, faction),
		    description = COALESCE($3, description),
		    requisition_points = COALESCE($4, requisition_points),
		    supply_limit = COALESCE($5, supply_limit),
		    updated_at = now()
		WHERE id = $6 AND user_id = $7
		RETURNING id, user_id, name, faction, description,
		          crusade_points, requisition_points, supply_limit, supply_cost,
		          created_at, updated_at
	`

	var w models.Warband

	err := db.PGClient.QueryRow(context.Background(), query,
		req.Name, req.Faction, req.Description,
		req.RequisitionPoints, req.SupplyLimit,
		id, userID,
	).Scan(
		&w.ID, &w.UserID, &w.Name, &w.Faction, &w.Description,
		&w.CrusadePoints, &w.RequisitionPoints, &w.SupplyLimit, &w.SupplyCost,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return models.Warband{}, err
	}

	numUnits, totalPoints, err := GetWarbandTotals(id)
	if err != nil {
		return models.Warband{}, err
	}
	w.NumUnits = numUnits
	w.TotalPointsCost = totalPoints

	units, err := GetUnitsByWarbandID(id)
	if err != nil {
		return models.Warband{}, err
	}
	w.Units = units

	return w, nil
}

// Data Deletions

func DeleteWarband(id string, userID string) error {
	query := `
		DELETE FROM warbands
		WHERE id = $1 AND user_id = $2
	`

	tag, err := db.PGClient.Exec(context.Background(), query, id, userID)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Data Queries

// Warband Utilities
func GetAllWarbands(id string) ([]models.Warband, error) {
	query := `
		SELECT id, user_id, name, faction, description,
		       crusade_points, requisition_points, supply_limit, supply_cost,
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

		err := rows.Scan(
			&w.ID, &w.UserID, &w.Name, &w.Faction, &w.Description,
			&w.CrusadePoints, &w.RequisitionPoints, &w.SupplyLimit, &w.SupplyCost,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		numUnits, totalPoints, err := GetWarbandTotals(w.ID.String())
		if err != nil {
			return nil, err
		}
		w.NumUnits = numUnits
		w.TotalPointsCost = totalPoints

		units, err := GetUnitsByWarbandID(w.ID.String())
		if err != nil {
			return nil, err
		}
		w.Units = units

		warbands = append(warbands, w)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return warbands, nil
}

func GetWarbandByID(id string) (models.Warband, error) {
	var w models.Warband

	query := `
		SELECT id, user_id, name, faction, description,
		       crusade_points, requisition_points, supply_limit, supply_cost,
		       created_at, updated_at
		FROM warbands
		WHERE id = $1
	`

	err := db.PGClient.QueryRow(context.Background(), query, id).Scan(
		&w.ID, &w.UserID, &w.Name, &w.Faction, &w.Description,
		&w.CrusadePoints, &w.RequisitionPoints, &w.SupplyLimit, &w.SupplyCost,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return models.Warband{}, err
	}

	numUnits, totalPoints, err := GetWarbandTotals(id)
	if err != nil {
		return models.Warband{}, err
	}
	w.NumUnits = numUnits
	w.TotalPointsCost = totalPoints

	units, err := GetUnitsByWarbandID(id)
	if err != nil {
		return models.Warband{}, err
	}
	w.Units = units

	return w, nil
}

func GetUnitsByWarbandID(warbandID string) ([]models.Unit, error) {
	query := `
		SELECT id, warband_id, unit_name, narrative_name, bio,
		       points, kills, experience, perks, created_at, updated_at
		FROM units
		WHERE warband_id = $1
		ORDER BY created_at ASC
	`

	rows, err := db.PGClient.Query(context.Background(), query, warbandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	units := []models.Unit{}

	for rows.Next() {
		var u models.Unit
		var perksJSON []byte

		if err := rows.Scan(
			&u.ID, &u.WarbandID, &u.UnitName, &u.NarrativeName, &u.Bio,
			&u.Points, &u.Kills, &u.Experience, &perksJSON, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}

		if err := json.Unmarshal(perksJSON, &u.Perks); err != nil {
			return nil, err
		}

		units = append(units, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return units, nil
}

func GetWarbandTotals(warbandID string) (int, int, error) {
	var numUnits, totalPoints int

	query := `
		SELECT COUNT(*), COALESCE(SUM(points), 0)
		FROM units
		WHERE warband_id = $1
	`

	err := db.PGClient.QueryRow(context.Background(), query, warbandID).Scan(&numUnits, &totalPoints)
	if err != nil {
		return 0, 0, err
	}

	return numUnits, totalPoints, nil
}

func IsWarbandOwner(warbandID string, userID string) (bool, error) {
	var exists bool

	query := `
		SELECT EXISTS (
			SELECT 1 FROM warbands WHERE id = $1 AND user_id = $2
		)
	`

	err := db.PGClient.QueryRow(context.Background(), query, warbandID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func SaveWarband(warband models.Warband) error {
	query := `
		UPDATE warbands
		SET name = $1, faction = $2, description = $3,
		    crusade_points = $4, requisition_points = $5,
		    supply_limit = $6, supply_cost = $7, updated_at = $8
		WHERE id = $9 AND user_id = $10
	`

	tag, err := db.PGClient.Exec(context.Background(), query,
		warband.Name, warband.Faction, warband.Description,
		warband.CrusadePoints, warband.RequisitionPoints,
		warband.SupplyLimit, warband.SupplyCost, warband.UpdatedAt,
		warband.ID, warband.UserID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Unit Utilities
func CreateUnit(unit models.Unit) error {
	perksJSON, err := json.Marshal(unit.Perks)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO units (
			id, warband_id, unit_name, narrative_name, bio,
			points, kills, experience, perks, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = db.PGClient.Exec(context.Background(), query,
		unit.ID, unit.WarbandID, unit.UnitName, unit.NarrativeName, unit.Bio,
		unit.Points, unit.Kills, unit.Experience, perksJSON, unit.CreatedAt, unit.UpdatedAt,
	)

	return err
}

func GetUnitByID(id string) (models.Unit, error) {
	var u models.Unit
	var perksJSON []byte

	query := `
		SELECT id, warband_id, unit_name, narrative_name, bio,
		       points, kills, experience, perks, created_at, updated_at
		FROM units
		WHERE id = $1
	`

	err := db.PGClient.QueryRow(context.Background(), query, id).Scan(
		&u.ID, &u.WarbandID, &u.UnitName, &u.NarrativeName, &u.Bio,
		&u.Points, &u.Kills, &u.Experience, &perksJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return models.Unit{}, err
	}

	if err := json.Unmarshal(perksJSON, &u.Perks); err != nil {
		return models.Unit{}, err
	}

	return u, nil
}
