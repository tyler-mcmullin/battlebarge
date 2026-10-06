package repositories

// limits
// Caps how many of each child record a parent can have, so one account cannot
// fill the database. Enforced inside the insert's transaction while holding a
// lock on the parent row, so concurrent requests cannot slip past the cap.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const (
	MaxWarbandsPerUser     = 50
	MaxCampaignsPerUser    = 20
	MaxUnitsPerWarband     = 100
	MaxPerksPerUnit        = 50
	MaxChaptersPerCampaign = 100
	MaxTeamsPerCampaign    = 20
	MaxWarbandsPerCampaign = 100
)

// ErrLimitReached is returned when creating a record would exceed its cap.
var ErrLimitReached = errors.New("limit reached")

// Arguments: ctx (context.Context); tx (pgx.Tx) - the open transaction; parentTable (string) - table of the parent row, a trusted constant; parentID (any) - the parent's ID; countQuery (string) - a trusted constant query counting existing children for $1; limit (int) - the maximum allowed
//
// Returns: error - pgx.ErrNoRows if the parent does not exist, ErrLimitReached if the parent is already at the limit, or another error on failure
//
// Locks the parent row until the transaction ends, then checks the child count against the limit
func lockAndCheckLimit(ctx context.Context, tx pgx.Tx, parentTable string, parentID any, countQuery string, limit int) error {
	var one int
	lock := `SELECT 1 FROM ` + parentTable + ` WHERE id = $1 FOR NO KEY UPDATE`
	if err := tx.QueryRow(ctx, lock, parentID).Scan(&one); err != nil {
		return err
	}

	var n int
	if err := tx.QueryRow(ctx, countQuery, parentID).Scan(&n); err != nil {
		return err
	}
	if n >= limit {
		return ErrLimitReached
	}

	return nil
}
