package repositories

// Repositories package handles database insertions and queries
// Functions called by controllers as needed

import (
	"context"

	"github.com/jackc/pgx/v5"

	"battlebarge/db"
	"battlebarge/models"
)

func CreateWarband(warband models.Warband) error {
	query := `
		INSERT INTO warbands (
			id, user_id, name, faction, description,
			requisition_points, supply_limit,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := db.PGClient.Exec(
		context.Background(),
		query,
		warband.ID,
		warband.UserID,
		warband.Name,
		warband.Faction,
		warband.Description,
		warband.RequisitionPoints,
		warband.SupplyLimit,
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
		          requisition_points, supply_limit,
		          created_at, updated_at
	`

	var w models.Warband

	err := db.PGClient.QueryRow(context.Background(), query,
		req.Name, req.Faction, req.Description,
		req.RequisitionPoints, req.SupplyLimit,
		id, userID,
	).Scan(
		&w.ID, &w.UserID, &w.Name, &w.Faction, &w.Description,
		&w.RequisitionPoints, &w.SupplyLimit,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return models.Warband{}, err
	}

	units, err := GetUnitsByWarbandID(id)
	if err != nil {
		return models.Warband{}, err
	}
	w.Units = units
	w.NumUnits = len(units)

	total := 0
	for _, u := range units {
		total += u.Points
	}
	w.TotalPointsCost = total
	w.CrusadePoints = CalculateCrusadePoints(units)

	return w, nil
}

func GetAllWarbands(id string) ([]models.Warband, error) {
	query := `
		SELECT id, user_id, name, faction, description,
		       requisition_points, supply_limit,
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
			&w.RequisitionPoints, &w.SupplyLimit,
			&w.CreatedAt, &w.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		units, err := GetUnitsByWarbandID(w.ID.String())
		if err != nil {
			return nil, err
		}
		w.Units = units
		w.NumUnits = len(units)

		total := 0
		for _, u := range units {
			total += u.Points
		}
		w.TotalPointsCost = total
		w.CrusadePoints = CalculateCrusadePoints(units)

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
		       requisition_points, supply_limit,
		       created_at, updated_at
		FROM warbands
		WHERE id = $1
	`

	err := db.PGClient.QueryRow(context.Background(), query, id).Scan(
		&w.ID, &w.UserID, &w.Name, &w.Faction, &w.Description,
		&w.RequisitionPoints, &w.SupplyLimit,
		&w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return models.Warband{}, err
	}

	units, err := GetUnitsByWarbandID(id)
	if err != nil {
		return models.Warband{}, err
	}
	w.Units = units
	w.NumUnits = len(units)

	total := 0
	for _, u := range units {
		total += u.Points
	}
	w.TotalPointsCost = total

	w.CrusadePoints = CalculateCrusadePoints(units)

	return w, nil
}

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

// Helpers

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
		    requisition_points = $4,
		    supply_limit = $5, updated_at = $6
		WHERE id = $7 AND user_id = $8
	`

	tag, err := db.PGClient.Exec(context.Background(), query,
		warband.Name, warband.Faction, warband.Description,
		warband.RequisitionPoints,
		warband.SupplyLimit, warband.UpdatedAt,
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

func CalculateCrusadePoints(units []models.Unit) int {
	points := 0
	for _, u := range units {
		for _, p := range u.Perks {
			if p.IsScar {
				points--
			} else {
				points++
			}
		}
	}
	return points
}
