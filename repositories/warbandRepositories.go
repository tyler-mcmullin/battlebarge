package repositories

// warbandRepositories
// Handles database interactions used by warbadControllers

import (
	"context"

	"github.com/jackc/pgx/v5"

	"battlebarge/db"
	"battlebarge/models"
)

// Arguments: warband (models.Warband) - warband record to insert
//
// Returns: error - ErrLimitReached if the user already has MaxWarbandsPerUser warbands, or another error if the insert fails
//
// Inserts a new warband row into the warbands table
func CreateWarband(warband models.Warband) error {
	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = lockAndCheckLimit(ctx, tx, "users", warband.UserID,
		`SELECT count(*) FROM warbands WHERE user_id = $1`, MaxWarbandsPerUser)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO warbands (
			id, user_id, name, faction, description,
			requisition_points, supply_limit,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = tx.Exec(
		ctx,
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
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Arguments: id (string) - warband ID; userID (string) - ID of the owning user; req (models.UpdateWarbandRequest) - fields to change, nil fields are left as-is
//
// Returns: models.Warband - the updated warband with its units and computed totals; error - pgx.ErrNoRows if not found or not owned by userID
//
// Partially updates a warband owned by the user and returns it with units, point totals, and crusade points populated
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

// Arguments: id (string) - ID of the owning user
//
// Returns: []models.Warband - the user's warbands, newest first, each with units and computed totals; error - on query or scan failure
//
// Fetches every warband belonging to a user
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

// Arguments: id (string) - warband ID
//
// Returns: models.Warband - the warband with units and computed totals; error - pgx.ErrNoRows if not found
//
// Fetches a single warband by ID, regardless of owner
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

// Arguments: id (string) - warband ID; userID (string) - ID of the owning user
//
// Returns: error - pgx.ErrNoRows if no warband matched the ID and owner, or another error on failure
//
// Deletes a warband owned by the user
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

// Arguments: warbandID (string) - warband ID; userID (string) - user ID to check
//
// Returns: bool - true if the user owns the warband; error - on query failure
//
// Checks whether a user owns a warband
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

// Arguments: warband (models.Warband) - warband with updated fields
//
// Returns: error - pgx.ErrNoRows if no warband matched the ID and owner, or another error on failure
//
// Writes all editable fields of a warband back to the database, overwriting existing values
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

// Arguments: units ([]models.Unit) - units to total
//
// Returns: int - crusade points
//
// Totals crusade points across units: +1 for each perk and -1 for each scar
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
