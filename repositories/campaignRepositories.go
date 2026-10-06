package repositories

// campaignRepositories
// Handles database interactions used by campaignControllers

import (
	"context"

	"github.com/jackc/pgx/v5"

	"battlebarge/db"
	"battlebarge/models"
)

const campaignColumns = `id, owner_id, name, description,
		points_per_win, points_per_loss, starting_requisition,
		created_at, updated_at`

// Arguments: row (pgx.Row) - a row selecting campaignColumns
//
// Returns: models.Campaign - the scanned campaign with empty (non-nil) chapters, teams and warbands; error - on scan failure
//
// Scans a campaign row without loading its chapters, teams, or warbands
func scanCampaign(row pgx.Row) (models.Campaign, error) {
	var c models.Campaign
	err := row.Scan(
		&c.ID, &c.OwnerID, &c.Name, &c.Description,
		&c.Settings.PointsPerWin, &c.Settings.PointsPerLoss, &c.Settings.StartingRequisition,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return models.Campaign{}, err
	}
	c.Chapters = []models.CampaignChapter{}
	c.Teams = []models.CampaignTeam{}
	c.Warbands = []models.CampaignWarband{}
	return c, nil
}

// Arguments: c (models.Campaign) - campaign whose chapters, teams, and warbands should be loaded
//
// Returns: models.Campaign - the campaign with its chapters, teams, and warbands populated; error - on query failure
//
// Loads a campaign's child records from their tables and attaches them
func withCampaignDetails(c models.Campaign) (models.Campaign, error) {
	id := c.ID.String()

	chapters, err := GetChaptersByCampaignID(id)
	if err != nil {
		return models.Campaign{}, err
	}
	teams, err := GetTeamsByCampaignID(id)
	if err != nil {
		return models.Campaign{}, err
	}
	warbands, err := GetCampaignWarbands(id)
	if err != nil {
		return models.Campaign{}, err
	}

	c.Chapters = chapters
	c.Teams = teams
	c.Warbands = warbands
	return c, nil
}

// Arguments: campaign (models.Campaign) - campaign record to insert
//
// Returns: error - ErrLimitReached if the owner already has MaxCampaignsPerUser campaigns, or another error if the insert fails
//
// Inserts a new campaign row
func CreateCampaign(campaign models.Campaign) error {
	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = lockAndCheckLimit(ctx, tx, "users", campaign.OwnerID,
		`SELECT count(*) FROM campaigns WHERE owner_id = $1`, MaxCampaignsPerUser)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO campaigns (
			id, owner_id, name, description,
			points_per_win, points_per_loss, starting_requisition,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = tx.Exec(ctx, query,
		campaign.ID, campaign.OwnerID, campaign.Name, campaign.Description,
		campaign.Settings.PointsPerWin, campaign.Settings.PointsPerLoss, campaign.Settings.StartingRequisition,
		campaign.CreatedAt, campaign.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Arguments: id (string) - campaign ID
//
// Returns: models.Campaign - the campaign with its chapters, teams, and warbands; error - pgx.ErrNoRows if not found
//
// Fetches a single campaign by ID, regardless of owner
func GetCampaignByID(id string) (models.Campaign, error) {
	query := `SELECT ` + campaignColumns + ` FROM campaigns WHERE id = $1`

	c, err := scanCampaign(db.PGClient.QueryRow(context.Background(), query, id))
	if err != nil {
		return models.Campaign{}, err
	}

	return withCampaignDetails(c)
}

// Arguments: userID (string) - user ID
//
// Returns: []models.Campaign - campaigns the user owns or has a warband in, newest first, each with details; error - on query or scan failure
//
// Fetches every campaign a user owns or participates in through one of their warbands
func GetCampaignsForUser(userID string) ([]models.Campaign, error) {
	query := `
		SELECT ` + campaignColumns + `
		FROM campaigns c
		WHERE c.owner_id = $1
		   OR EXISTS (
			SELECT 1
			FROM campaign_warbands cw
			JOIN warbands w ON w.id = cw.warband_id
			WHERE cw.campaign_id = c.id AND w.user_id = $1
		   )
		ORDER BY c.created_at DESC
	`

	rows, err := db.PGClient.Query(context.Background(), query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	campaigns := []models.Campaign{}
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

	for i := range campaigns {
		campaigns[i], err = withCampaignDetails(campaigns[i])
		if err != nil {
			return nil, err
		}
	}

	return campaigns, nil
}

// Arguments: id (string) - campaign ID; ownerID (string) - ID of the owning user; req (models.UpdateCampaignRequest) - fields to change, nil fields are left as-is
//
// Returns: models.Campaign - the updated campaign with details; error - pgx.ErrNoRows if not found or not owned by ownerID
//
// Partially updates a campaign's name, description, and settings
func UpdateCampaign(id string, ownerID string, req models.UpdateCampaignRequest) (models.Campaign, error) {
	query := `
		UPDATE campaigns
		SET name = COALESCE($1, name),
		    description = COALESCE($2, description),
		    points_per_win = COALESCE($3, points_per_win),
		    points_per_loss = COALESCE($4, points_per_loss),
		    starting_requisition = COALESCE($5, starting_requisition),
		    updated_at = now()
		WHERE id = $6 AND owner_id = $7
		RETURNING ` + campaignColumns

	c, err := scanCampaign(db.PGClient.QueryRow(context.Background(), query,
		req.Name, req.Description, req.PointsPerWin, req.PointsPerLoss, req.StartingRequisition,
		id, ownerID,
	))
	if err != nil {
		return models.Campaign{}, err
	}

	return withCampaignDetails(c)
}

// Arguments: id (string) - campaign ID; ownerID (string) - ID of the owning user
//
// Returns: error - pgx.ErrNoRows if no campaign matched the ID and owner, or another error on failure
//
// Deletes a campaign owned by the user; its chapters, teams, and memberships are removed by foreign key cascades
func DeleteCampaign(id string, ownerID string) error {
	query := `DELETE FROM campaigns WHERE id = $1 AND owner_id = $2`

	tag, err := db.PGClient.Exec(context.Background(), query, id, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Arguments: campaignID (string) - campaign ID; userID (string) - user ID to check
//
// Returns: bool - true if the user owns the campaign; error - on query failure
//
// Checks whether a user owns a campaign
func IsCampaignOwner(campaignID string, userID string) (bool, error) {
	var exists bool

	query := `
		SELECT EXISTS (
			SELECT 1 FROM campaigns WHERE id = $1 AND owner_id = $2
		)
	`

	err := db.PGClient.QueryRow(context.Background(), query, campaignID, userID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}

// Arguments: campaignID (string) - campaign ID
//
// Returns: []models.CampaignChapter - the campaign's chapters ordered by sort_order (empty if none); error - on query or scan failure
//
// Fetches all chapters of a campaign
func GetChaptersByCampaignID(campaignID string) ([]models.CampaignChapter, error) {
	query := `
		SELECT id, campaign_id, title, description, sort_order, created_at, updated_at
		FROM campaign_chapters
		WHERE campaign_id = $1
		ORDER BY sort_order ASC, created_at ASC
	`

	rows, err := db.PGClient.Query(context.Background(), query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chapters := []models.CampaignChapter{}
	for rows.Next() {
		var ch models.CampaignChapter
		if err := rows.Scan(&ch.ID, &ch.CampaignID, &ch.Title, &ch.Description, &ch.SortOrder, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, err
		}
		chapters = append(chapters, ch)
	}

	return chapters, rows.Err()
}

// Arguments: chapter (models.CampaignChapter) - chapter to insert; autoOrder (bool) - when true, ignore chapter.SortOrder and append after the last chapter
//
// Returns: models.CampaignChapter - the inserted chapter with its final sort_order; error - pgx.ErrNoRows if the campaign does not exist, ErrLimitReached if it already has MaxChaptersPerCampaign chapters, or another error on failure
//
// Inserts a chapter into a campaign
func AddChapter(chapter models.CampaignChapter, autoOrder bool) (models.CampaignChapter, error) {
	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return models.CampaignChapter{}, err
	}
	defer tx.Rollback(ctx)

	err = lockAndCheckLimit(ctx, tx, "campaigns", chapter.CampaignID,
		`SELECT count(*) FROM campaign_chapters WHERE campaign_id = $1`, MaxChaptersPerCampaign)
	if err != nil {
		return models.CampaignChapter{}, err
	}

	query := `
		INSERT INTO campaign_chapters (id, campaign_id, title, description, sort_order, created_at, updated_at)
		VALUES (
			$1, $2, $3, $4,
			CASE WHEN $5 THEN
				(SELECT COALESCE(MAX(sort_order), 0) + 1 FROM campaign_chapters WHERE campaign_id = $2)
			ELSE $6 END,
			$7, $8
		)
		RETURNING id, campaign_id, title, description, sort_order, created_at, updated_at
	`

	var ch models.CampaignChapter
	err = tx.QueryRow(ctx, query,
		chapter.ID, chapter.CampaignID, chapter.Title, chapter.Description,
		autoOrder, chapter.SortOrder,
		chapter.CreatedAt, chapter.UpdatedAt,
	).Scan(&ch.ID, &ch.CampaignID, &ch.Title, &ch.Description, &ch.SortOrder, &ch.CreatedAt, &ch.UpdatedAt)
	if err != nil {
		return models.CampaignChapter{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.CampaignChapter{}, err
	}

	return ch, nil
}

// Arguments: campaignID (string) - campaign ID; chapterID (string) - chapter ID; req (models.UpdateChapterRequest) - fields to change, nil fields are left as-is
//
// Returns: models.CampaignChapter - the updated chapter; error - pgx.ErrNoRows if the chapter is not in that campaign
//
// Partially updates a chapter's title, description, and sort order
func UpdateChapter(campaignID string, chapterID string, req models.UpdateChapterRequest) (models.CampaignChapter, error) {
	query := `
		UPDATE campaign_chapters
		SET title = COALESCE($1, title),
		    description = COALESCE($2, description),
		    sort_order = COALESCE($3, sort_order),
		    updated_at = now()
		WHERE id = $4 AND campaign_id = $5
		RETURNING id, campaign_id, title, description, sort_order, created_at, updated_at
	`

	var ch models.CampaignChapter
	err := db.PGClient.QueryRow(context.Background(), query,
		req.Title, req.Description, req.SortOrder, chapterID, campaignID,
	).Scan(&ch.ID, &ch.CampaignID, &ch.Title, &ch.Description, &ch.SortOrder, &ch.CreatedAt, &ch.UpdatedAt)

	return ch, err
}

// Arguments: campaignID (string) - campaign ID; chapterID (string) - chapter ID
//
// Returns: error - pgx.ErrNoRows if the chapter is not in that campaign, or another error on failure
//
// Deletes a chapter from a campaign
func DeleteChapter(campaignID string, chapterID string) error {
	query := `DELETE FROM campaign_chapters WHERE id = $1 AND campaign_id = $2`

	tag, err := db.PGClient.Exec(context.Background(), query, chapterID, campaignID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Arguments: campaignID (string) - campaign ID
//
// Returns: []models.CampaignTeam - the campaign's teams, oldest first (empty if none); error - on query or scan failure
//
// Fetches all teams of a campaign
func GetTeamsByCampaignID(campaignID string) ([]models.CampaignTeam, error) {
	query := `
		SELECT id, campaign_id, name, created_at, updated_at
		FROM campaign_teams
		WHERE campaign_id = $1
		ORDER BY created_at ASC, id ASC
	`

	rows, err := db.PGClient.Query(context.Background(), query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := []models.CampaignTeam{}
	for rows.Next() {
		var t models.CampaignTeam
		if err := rows.Scan(&t.ID, &t.CampaignID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}

	return teams, rows.Err()
}

// Arguments: team (models.CampaignTeam) - team to insert
//
// Returns: error - SQLSTATE 23505 if the campaign already has a team with that name, ErrLimitReached if it already has MaxTeamsPerCampaign teams, or another error on failure
//
// Inserts a team into a campaign
func CreateTeam(team models.CampaignTeam) error {
	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = lockAndCheckLimit(ctx, tx, "campaigns", team.CampaignID,
		`SELECT count(*) FROM campaign_teams WHERE campaign_id = $1`, MaxTeamsPerCampaign)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO campaign_teams (id, campaign_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = tx.Exec(ctx, query,
		team.ID, team.CampaignID, team.Name, team.CreatedAt, team.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// Arguments: campaignID (string) - campaign ID; teamID (string) - team ID; name (string) - new team name
//
// Returns: models.CampaignTeam - the renamed team; error - pgx.ErrNoRows if the team is not in that campaign, SQLSTATE 23505 if the name is taken
//
// Renames a team
func RenameTeam(campaignID string, teamID string, name string) (models.CampaignTeam, error) {
	query := `
		UPDATE campaign_teams
		SET name = $1, updated_at = now()
		WHERE id = $2 AND campaign_id = $3
		RETURNING id, campaign_id, name, created_at, updated_at
	`

	var t models.CampaignTeam
	err := db.PGClient.QueryRow(context.Background(), query, name, teamID, campaignID).
		Scan(&t.ID, &t.CampaignID, &t.Name, &t.CreatedAt, &t.UpdatedAt)

	return t, err
}

// Arguments: campaignID (string) - campaign ID; teamID (string) - team ID
//
// Returns: error - pgx.ErrNoRows if the team is not in that campaign, SQLSTATE 23503 if warbands are still on the team, or another error on failure
//
// Deletes a team that has no warbands on it
func DeleteTeam(campaignID string, teamID string) error {
	query := `DELETE FROM campaign_teams WHERE id = $1 AND campaign_id = $2`

	tag, err := db.PGClient.Exec(context.Background(), query, teamID, campaignID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Arguments: campaignID (string) - campaign ID
//
// Returns: []models.CampaignWarband - the campaign's member warbands with their teams, in join order (empty if none); error - on query or scan failure
//
// Fetches all warbands that have joined a campaign
func GetCampaignWarbands(campaignID string) ([]models.CampaignWarband, error) {
	query := `
		SELECT cw.campaign_id, cw.warband_id, w.name, cw.team_id, cw.joined_at
		FROM campaign_warbands cw
		JOIN warbands w ON w.id = cw.warband_id
		WHERE cw.campaign_id = $1
		ORDER BY cw.joined_at ASC, cw.warband_id ASC
	`

	rows, err := db.PGClient.Query(context.Background(), query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []models.CampaignWarband{}
	for rows.Next() {
		var m models.CampaignWarband
		if err := rows.Scan(&m.CampaignID, &m.WarbandID, &m.WarbandName, &m.TeamID, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, rows.Err()
}

// Arguments: campaignID (string) - campaign ID; warbandID (string) - warband ID; teamID (string) - team to join
//
// Returns: error - pgx.ErrNoRows if the campaign or team does not exist, ErrLimitReached if the campaign already has MaxWarbandsPerCampaign warbands, SQLSTATE 23505 if the warband is already in the campaign, SQLSTATE 23503 if the warband does not exist
//
// Adds a warband to a campaign on the given team
func JoinCampaign(campaignID string, warbandID string, teamID string) error {
	ctx := context.Background()

	tx, err := db.PGClient.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = lockAndCheckLimit(ctx, tx, "campaigns", campaignID,
		`SELECT count(*) FROM campaign_warbands WHERE campaign_id = $1`, MaxWarbandsPerCampaign)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO campaign_warbands (campaign_id, warband_id, team_id)
		SELECT $1, $2, t.id
		FROM campaign_teams t
		WHERE t.id = $3 AND t.campaign_id = $1
	`

	tag, err := tx.Exec(ctx, query, campaignID, warbandID, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return tx.Commit(ctx)
}

// Arguments: campaignID (string) - campaign ID; warbandID (string) - warband ID; teamID (string) - team to move to
//
// Returns: error - pgx.ErrNoRows if the warband is not in the campaign or the team is not in that campaign, or another error on failure
//
// Moves a member warband to another team in the same campaign
func SetWarbandTeam(campaignID string, warbandID string, teamID string) error {
	query := `
		UPDATE campaign_warbands cw
		SET team_id = t.id
		FROM campaign_teams t
		WHERE cw.campaign_id = $1 AND cw.warband_id = $2
		  AND t.id = $3 AND t.campaign_id = $1
	`

	tag, err := db.PGClient.Exec(context.Background(), query, campaignID, warbandID, teamID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

// Arguments: campaignID (string) - campaign ID; warbandID (string) - warband ID
//
// Returns: error - pgx.ErrNoRows if the warband is not in the campaign, or another error on failure
//
// Removes a warband from a campaign
func LeaveCampaign(campaignID string, warbandID string) error {
	query := `DELETE FROM campaign_warbands WHERE campaign_id = $1 AND warband_id = $2`

	tag, err := db.PGClient.Exec(context.Background(), query, campaignID, warbandID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
