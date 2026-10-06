package repositories

// unitRepositories
// Handles database interactions used by unitControllers

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"

	"battlebarge/db"
	"battlebarge/models"
)

// Arguments: unit (models.Unit) - unit record to insert
//
// Returns: error - non-nil if marshalling perks or the insert fails
//
// Inserts a new unit row, storing its perks as JSON
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

// Arguments: id (string) - unit ID
//
// Returns: models.Unit - the matching unit; error - pgx.ErrNoRows if not found
//
// Fetches a single unit by ID
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

// Arguments: warbandID (string) - warband ID
//
// Returns: []models.Unit - the warband's units, oldest first; error - on query, scan, or perk decoding failure
//
// Fetches all units belonging to a warband
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

// Arguments: id (string) - unit ID; req (models.UpdateUnitRequest) - fields to change, nil fields are left as-is
//
// Returns: models.Unit - the updated unit; error - pgx.ErrNoRows if not found
//
// Partially updates a unit's name, narrative name, bio, and points
func UpdateUnit(id string, req models.UpdateUnitRequest) (models.Unit, error) {
	query := `
		UPDATE units
		SET unit_name = COALESCE($1, unit_name),
		    narrative_name = COALESCE($2, narrative_name),
		    bio = COALESCE($3, bio),
		    points = COALESCE($4, points),
		    updated_at = now()
		WHERE id = $5
		RETURNING id, warband_id, unit_name, narrative_name, bio,
		          points, kills, experience, perks, created_at, updated_at
	`

	var u models.Unit
	var perksJSON []byte

	err := db.PGClient.QueryRow(context.Background(), query,
		req.UnitName, req.NarrativeName, req.Bio, req.Points, id,
	).Scan(
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

// Arguments: id (string) - unit ID; amount (int) - value to add, may be negative
//
// Returns: models.Unit - the updated unit; error - pgx.ErrNoRows if not found
//
// Adds to a unit's kill count, clamping the result at 0
func IncrementUnitKills(id string, amount int) (models.Unit, error) {
	query := `
		UPDATE units
		SET kills = GREATEST(kills + $1, 0), updated_at = now()
		WHERE id = $2
		RETURNING id, warband_id, unit_name, narrative_name, bio,
		          points, kills, experience, perks, created_at, updated_at
	`

	var u models.Unit
	var perksJSON []byte

	err := db.PGClient.QueryRow(context.Background(), query, amount, id).Scan(
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

// Arguments: id (string) - unit ID; amount (int) - value to add, may be negative
//
// Returns: models.Unit - the updated unit; error - pgx.ErrNoRows if not found
//
// Adds to a unit's experience, clamping the result at 0
func IncrementUnitXP(id string, amount int) (models.Unit, error) {
	query := `
		UPDATE units
		SET experience = GREATEST(experience + $1, 0), updated_at = now()
		WHERE id = $2
		RETURNING id, warband_id, unit_name, narrative_name, bio,
		          points, kills, experience, perks, created_at, updated_at
	`

	var u models.Unit
	var perksJSON []byte

	err := db.PGClient.QueryRow(context.Background(), query, amount, id).Scan(
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

// Arguments: id (string) - unit ID; req (models.AddPerkRequest) - perk to add
//
// Returns: models.Unit - the updated unit; error - pgx.ErrNoRows if the unit is not found, or another error on failure
//
// Appends a perk (or scar) to a unit's perks
func AddUnitPerk(id string, req models.AddPerkRequest) (models.Unit, error) {
	unit, err := GetUnitByID(id)
	if err != nil {
		return models.Unit{}, err
	}

	newPerk := models.Perk{
		ID:          req.ID,
		Name:        req.Name,
		Description: "",
		IsScar:      req.IsScar,
	}
	if req.Description != nil {
		newPerk.Description = *req.Description
	}

	unit.Perks = append(unit.Perks, newPerk)

	perksJSON, err := json.Marshal(unit.Perks)
	if err != nil {
		return models.Unit{}, err
	}

	query := `
		UPDATE units
		SET perks = $1, updated_at = now()
		WHERE id = $2
		RETURNING id, warband_id, unit_name, narrative_name, bio,
		          points, kills, experience, perks, created_at, updated_at
	`

	var u models.Unit
	var updatedPerksJSON []byte

	err = db.PGClient.QueryRow(context.Background(), query, perksJSON, id).Scan(
		&u.ID, &u.WarbandID, &u.UnitName, &u.NarrativeName, &u.Bio,
		&u.Points, &u.Kills, &u.Experience, &updatedPerksJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return models.Unit{}, err
	}

	if err := json.Unmarshal(updatedPerksJSON, &u.Perks); err != nil {
		return models.Unit{}, err
	}

	return u, nil
}

// Arguments: unitID (string) - unit ID; perkID (string) - ID of the perk to remove
//
// Returns: models.Unit - the updated unit; error - pgx.ErrNoRows if the unit or perk is not found
//
// Removes a perk from a unit's perks
func DeleteUnitPerk(unitID string, perkID string) (models.Unit, error) {
	unit, err := GetUnitByID(unitID)
	if err != nil {
		return models.Unit{}, err
	}

	updatedPerks := []models.Perk{}
	found := false
	for _, p := range unit.Perks {
		if p.ID.String() == perkID {
			found = true
			continue
		}
		updatedPerks = append(updatedPerks, p)
	}

	if !found {
		return models.Unit{}, pgx.ErrNoRows
	}

	perksJSON, err := json.Marshal(updatedPerks)
	if err != nil {
		return models.Unit{}, err
	}

	query := `
		UPDATE units
		SET perks = $1, updated_at = now()
		WHERE id = $2
		RETURNING id, warband_id, unit_name, narrative_name, bio,
		          points, kills, experience, perks, created_at, updated_at
	`

	var u models.Unit
	var updatedPerksJSON []byte

	err = db.PGClient.QueryRow(context.Background(), query, perksJSON, unitID).Scan(
		&u.ID, &u.WarbandID, &u.UnitName, &u.NarrativeName, &u.Bio,
		&u.Points, &u.Kills, &u.Experience, &updatedPerksJSON, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return models.Unit{}, err
	}

	if err := json.Unmarshal(updatedPerksJSON, &u.Perks); err != nil {
		return models.Unit{}, err
	}

	return u, nil
}

// Arguments: id (string) - unit ID
//
// Returns: error - pgx.ErrNoRows if the unit does not exist, or another error on failure
//
// Deletes a unit by ID
func DeleteUnit(id string) error {
	query := `DELETE FROM units WHERE id = $1`

	tag, err := db.PGClient.Exec(context.Background(), query, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
