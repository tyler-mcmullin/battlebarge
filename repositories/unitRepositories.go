package repositories

// unitRepositories
// Handles database interactions used by unitControllers

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"battlebarge/db"
	"battlebarge/models"
)

// unitColumns is the column list shared by every query that returns a unit
// row. Perks live in the unit_perks table and are loaded separately.
const unitColumns = `id, warband_id, unit_name, narrative_name, bio,
		points, kills, experience, created_at, updated_at`

// Arguments: row (pgx.Row) - a row selecting unitColumns
//
// Returns: models.Unit - the scanned unit with an empty (non-nil) perks slice; error - on scan failure
//
// Scans a unit row without loading its perks
func scanUnit(row pgx.Row) (models.Unit, error) {
	var u models.Unit
	err := row.Scan(
		&u.ID, &u.WarbandID, &u.UnitName, &u.NarrativeName, &u.Bio,
		&u.Points, &u.Kills, &u.Experience, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return models.Unit{}, err
	}
	u.Perks = []models.Perk{}
	return u, nil
}

// Arguments: unitID (string) - unit ID
//
// Returns: []models.Perk - the unit's perks, oldest first (empty if none); error - on query or scan failure
//
// Fetches all perks belonging to a unit
func GetPerksByUnitID(unitID string) ([]models.Perk, error) {
	query := `
		SELECT id, name, description, is_scar
		FROM unit_perks
		WHERE unit_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := db.PGClient.Query(context.Background(), query, unitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	perks := []models.Perk{}
	for rows.Next() {
		var p models.Perk
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.IsScar); err != nil {
			return nil, err
		}
		perks = append(perks, p)
	}

	return perks, rows.Err()
}

// Arguments: u (models.Unit) - unit whose perks should be loaded
//
// Returns: models.Unit - the unit with its perks populated; error - on query failure
//
// Loads a unit's perks from unit_perks and attaches them to the unit
func withPerks(u models.Unit) (models.Unit, error) {
	perks, err := GetPerksByUnitID(u.ID.String())
	if err != nil {
		return models.Unit{}, err
	}
	u.Perks = perks
	return u, nil
}

// Arguments: unit (models.Unit) - unit record to insert, including any initial perks
//
// Returns: error - non-nil if the unit or a perk insert fails (nothing is saved in that case)
//
// Inserts a new unit row and its perks in a single transaction
func CreateUnit(unit models.Unit) error {
	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO units (
			id, warband_id, unit_name, narrative_name, bio,
			points, kills, experience, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = tx.Exec(ctx, query,
		unit.ID, unit.WarbandID, unit.UnitName, unit.NarrativeName, unit.Bio,
		unit.Points, unit.Kills, unit.Experience, unit.CreatedAt, unit.UpdatedAt,
	)
	if err != nil {
		return err
	}

	for _, p := range unit.Perks {
		_, err = tx.Exec(ctx, `
			INSERT INTO unit_perks (id, unit_id, name, description, is_scar)
			VALUES ($1, $2, $3, $4, $5)
		`, p.ID, unit.ID, p.Name, p.Description, p.IsScar)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Arguments: id (string) - unit ID
//
// Returns: models.Unit - the matching unit with its perks; error - pgx.ErrNoRows if not found
//
// Fetches a single unit by ID
func GetUnitByID(id string) (models.Unit, error) {
	query := `SELECT ` + unitColumns + ` FROM units WHERE id = $1`

	u, err := scanUnit(db.PGClient.QueryRow(context.Background(), query, id))
	if err != nil {
		return models.Unit{}, err
	}

	return withPerks(u)
}

// Arguments: warbandID (string) - warband ID
//
// Returns: []models.Unit - the warband's units with their perks, oldest first; error - on query or scan failure
//
// Fetches all units belonging to a warband, loading every unit's perks in one extra query
func GetUnitsByWarbandID(warbandID string) ([]models.Unit, error) {
	query := `
		SELECT ` + unitColumns + `
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
	index := map[uuid.UUID]int{}

	for rows.Next() {
		u, err := scanUnit(rows)
		if err != nil {
			return nil, err
		}
		index[u.ID] = len(units)
		units = append(units, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	if len(units) == 0 {
		return units, nil
	}

	perkQuery := `
		SELECT p.unit_id, p.id, p.name, p.description, p.is_scar
		FROM unit_perks p
		JOIN units u ON u.id = p.unit_id
		WHERE u.warband_id = $1
		ORDER BY p.created_at ASC, p.id ASC
	`

	perkRows, err := db.PGClient.Query(context.Background(), perkQuery, warbandID)
	if err != nil {
		return nil, err
	}
	defer perkRows.Close()

	for perkRows.Next() {
		var unitID uuid.UUID
		var p models.Perk
		if err := perkRows.Scan(&unitID, &p.ID, &p.Name, &p.Description, &p.IsScar); err != nil {
			return nil, err
		}
		if i, ok := index[unitID]; ok {
			units[i].Perks = append(units[i].Perks, p)
		}
	}
	if err := perkRows.Err(); err != nil {
		return nil, err
	}

	return units, nil
}

// Arguments: id (string) - unit ID; req (models.UpdateUnitRequest) - fields to change, nil fields are left as-is
//
// Returns: models.Unit - the updated unit with its perks; error - pgx.ErrNoRows if not found
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
		RETURNING ` + unitColumns

	u, err := scanUnit(db.PGClient.QueryRow(context.Background(), query,
		req.UnitName, req.NarrativeName, req.Bio, req.Points, id,
	))
	if err != nil {
		return models.Unit{}, err
	}

	return withPerks(u)
}

// Arguments: id (string) - unit ID; amount (int) - value to add, may be negative
//
// Returns: models.Unit - the updated unit with its perks; error - pgx.ErrNoRows if not found
//
// Adds to a unit's kill count, clamping the result at 0
func IncrementUnitKills(id string, amount int) (models.Unit, error) {
	query := `
		UPDATE units
		SET kills = GREATEST(kills + $1, 0), updated_at = now()
		WHERE id = $2
		RETURNING ` + unitColumns

	u, err := scanUnit(db.PGClient.QueryRow(context.Background(), query, amount, id))
	if err != nil {
		return models.Unit{}, err
	}

	return withPerks(u)
}

// Arguments: id (string) - unit ID; amount (int) - value to add, may be negative
//
// Returns: models.Unit - the updated unit with its perks; error - pgx.ErrNoRows if not found
//
// Adds to a unit's experience, clamping the result at 0
func IncrementUnitXP(id string, amount int) (models.Unit, error) {
	query := `
		UPDATE units
		SET experience = GREATEST(experience + $1, 0), updated_at = now()
		WHERE id = $2
		RETURNING ` + unitColumns

	u, err := scanUnit(db.PGClient.QueryRow(context.Background(), query, amount, id))
	if err != nil {
		return models.Unit{}, err
	}

	return withPerks(u)
}

// Arguments: id (string) - unit ID; req (models.AddPerkRequest) - perk to add
//
// Returns: models.Unit - the updated unit with its perks; error - pgx.ErrNoRows if the unit is not found, or another error on failure
//
// Inserts a perk (or scar) into unit_perks and bumps the unit's updated_at, atomically
func AddUnitPerk(id string, req models.AddPerkRequest) (models.Unit, error) {
	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return models.Unit{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE units SET updated_at = now() WHERE id = $1`, id)
	if err != nil {
		return models.Unit{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Unit{}, pgx.ErrNoRows
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO unit_perks (id, unit_id, name, description, is_scar)
		VALUES ($1, $2, $3, $4, $5)
	`, req.ID, id, req.Name, description, req.IsScar)
	if err != nil {
		return models.Unit{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Unit{}, err
	}

	return GetUnitByID(id)
}

// Arguments: unitID (string) - unit ID; perkID (string) - ID of the perk to remove
//
// Returns: models.Unit - the updated unit with its perks; error - pgx.ErrNoRows if the unit or perk is not found
//
// Deletes a perk from unit_perks and bumps the unit's updated_at, atomically
func DeleteUnitPerk(unitID string, perkID string) (models.Unit, error) {
	perkUUID, err := uuid.Parse(perkID)
	if err != nil {
		return models.Unit{}, pgx.ErrNoRows
	}

	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return models.Unit{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM unit_perks WHERE id = $1 AND unit_id = $2`, perkUUID, unitID)
	if err != nil {
		return models.Unit{}, err
	}
	if tag.RowsAffected() == 0 {
		return models.Unit{}, pgx.ErrNoRows
	}

	if _, err := tx.Exec(ctx, `UPDATE units SET updated_at = now() WHERE id = $1`, unitID); err != nil {
		return models.Unit{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Unit{}, err
	}

	return GetUnitByID(unitID)
}

// Arguments: id (string) - unit ID
//
// Returns: error - pgx.ErrNoRows if the unit does not exist, or another error on failure
//
// Deletes a unit by ID; its perks are removed by the unit_perks foreign key's ON DELETE CASCADE
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
