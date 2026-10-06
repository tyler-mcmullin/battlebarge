package models

import (
	"time"

	"github.com/google/uuid"
)

// Database Structs
type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Username  string    `json:"username" db:"username"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Warband struct {
	ID                uuid.UUID `json:"id"`
	UserID            string    `json:"user_id"`
	Name              string    `json:"name"`
	Faction           string    `json:"faction"`
	Description       string    `json:"description"`
	RequisitionPoints int       `json:"requisition_points"`
	SupplyLimit       int       `json:"supply_limit"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	// Computed fields populated at fetch time
	// from units table via GetWarbandTotals/GetUnitsByWarbandID.
	Units           []Unit `json:"units"`
	NumUnits        int    `json:"num_units"`
	TotalPointsCost int    `json:"total_points_cost"`
	CrusadePoints   int    `json:"crusade_points"`
}

type Unit struct {
	ID            uuid.UUID `json:"id" db:"id"`
	WarbandID     uuid.UUID `json:"warband_id" db:"warband_id"`
	UnitName      string    `json:"unit_name" db:"unit_name"`
	NarrativeName string    `json:"narrative_name" db:"narrative_name"`
	Bio           string    `json:"bio" db:"bio"`
	Points        int       `json:"points" db:"points"`
	Kills         int       `json:"kills" db:"kills"`
	Experience    int       `json:"experience" db:"experience"`
	Perks         []Perk    `json:"perks" db:"perks"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Perk struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsScar      bool      `json:"is_scar"`
}

type CampaignSettings struct {
	PointsPerWin        int `json:"points_per_win"`
	PointsPerLoss       int `json:"points_per_loss"`
	StartingRequisition int `json:"starting_requisition"`
}

type Campaign struct {
	ID          uuid.UUID        `json:"id"`
	OwnerID     string           `json:"owner_id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Settings    CampaignSettings `json:"settings"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`

	// JoinCode is only ever filled in for the campaign's owner. It is never
	// loaded by the repository queries, so public responses cannot include it
	// by accident; controllers set it explicitly for the owner.
	JoinCode string `json:"join_code,omitempty"`

	// Populated at fetch time from their own tables.
	Chapters []CampaignChapter `json:"chapters"`
	Teams    []CampaignTeam    `json:"teams"`
	Warbands []CampaignWarband `json:"warbands"`
}

type CampaignChapter struct {
	ID          uuid.UUID `json:"id"`
	CampaignID  uuid.UUID `json:"campaign_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CampaignTeam is a named team within one campaign. A campaign can have any
// number of teams, and teams can be renamed.
type CampaignTeam struct {
	ID         uuid.UUID `json:"id"`
	CampaignID uuid.UUID `json:"campaign_id"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CampaignWarband is a warband's membership in a campaign: which team it is on.
type CampaignWarband struct {
	CampaignID  uuid.UUID `json:"campaign_id"`
	WarbandID   uuid.UUID `json:"warband_id"`
	WarbandName string    `json:"warband_name"`
	TeamID      uuid.UUID `json:"team_id"`
	JoinedAt    time.Time `json:"joined_at"`
}

// Request Structs
//
// Validation limits (enforced by the binding tags below, and mirrored in
// docs/openapi.yaml): names and titles 1-100 characters and not blank;
// descriptions up to 2000 characters (unit bio 5000, perk description 1000);
// point-style numbers 0-1000000; increments -1000000 to 1000000 (0 is allowed).
// "safetext" rejects NUL and control characters.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email,max=255"`
	Username string `json:"username" binding:"required,notblank,safetext,max=50"`
	Password string `json:"password" binding:"required,max=128"`
}

type CreateWarbandRequest struct {
	Name              string  `json:"name" binding:"required,notblank,safetext,max=100"`
	Faction           *string `json:"faction" binding:"omitempty,safetext,max=100"`
	Description       *string `json:"description" binding:"omitempty,safetext,max=2000"`
	RequisitionPoints *int    `json:"requisition_points" binding:"omitempty,min=0,max=1000000"`
	SupplyLimit       *int    `json:"supply_limit" binding:"omitempty,min=0,max=1000000"`
}

type UpdateWarbandRequest struct {
	Name              *string `json:"name" binding:"omitempty,notblank,safetext,max=100"`
	Faction           *string `json:"faction" binding:"omitempty,safetext,max=100"`
	Description       *string `json:"description" binding:"omitempty,safetext,max=2000"`
	RequisitionPoints *int    `json:"requisition_points" binding:"omitempty,min=0,max=1000000"`
	SupplyLimit       *int    `json:"supply_limit" binding:"omitempty,min=0,max=1000000"`
}

type CreateUnitRequest struct {
	WarbandID     string  `json:"warband_id" binding:"required,max=36"`
	UnitName      string  `json:"unit_name" binding:"required,notblank,safetext,max=100"`
	NarrativeName *string `json:"narrative_name" binding:"omitempty,safetext,max=100"`
	Bio           *string `json:"bio" binding:"omitempty,safetext,max=5000"`
	Points        *int    `json:"points" binding:"omitempty,min=0,max=1000000"`
}

type UpdateUnitRequest struct {
	UnitName      *string `json:"unit_name" binding:"omitempty,notblank,safetext,max=100"`
	NarrativeName *string `json:"narrative_name" binding:"omitempty,safetext,max=100"`
	Bio           *string `json:"bio" binding:"omitempty,safetext,max=5000"`
	Points        *int    `json:"points" binding:"omitempty,min=0,max=1000000"`
}

// Amount is a pointer so a missing field (400) is distinguishable from an
// explicit 0, which is allowed and leaves the value unchanged.
type IncrementRequest struct {
	Amount *int `json:"amount" binding:"required,min=-1000000,max=1000000"`
}

type AddPerkRequest struct {
	ID          uuid.UUID `json:"perk_id"`
	Name        string    `json:"name" binding:"required,notblank,safetext,max=100"`
	Description *string   `json:"description" binding:"omitempty,safetext,max=1000"`
	IsScar      bool      `json:"is_scar"`
}

type CreateCampaignRequest struct {
	Name                string  `json:"name" binding:"required,notblank,safetext,max=100"`
	Description         *string `json:"description" binding:"omitempty,safetext,max=2000"`
	PointsPerWin        *int    `json:"points_per_win" binding:"omitempty,min=0,max=1000000"`
	PointsPerLoss       *int    `json:"points_per_loss" binding:"omitempty,min=0,max=1000000"`
	StartingRequisition *int    `json:"starting_requisition" binding:"omitempty,min=0,max=1000000"`
}

type UpdateCampaignRequest struct {
	Name                *string `json:"name" binding:"omitempty,notblank,safetext,max=100"`
	Description         *string `json:"description" binding:"omitempty,safetext,max=2000"`
	PointsPerWin        *int    `json:"points_per_win" binding:"omitempty,min=0,max=1000000"`
	PointsPerLoss       *int    `json:"points_per_loss" binding:"omitempty,min=0,max=1000000"`
	StartingRequisition *int    `json:"starting_requisition" binding:"omitempty,min=0,max=1000000"`
}

type CreateChapterRequest struct {
	Title       string  `json:"title" binding:"required,notblank,safetext,max=100"`
	Description *string `json:"description" binding:"omitempty,safetext,max=2000"`
	SortOrder   *int    `json:"sort_order" binding:"omitempty,min=0,max=1000000"`
}

type UpdateChapterRequest struct {
	Title       *string `json:"title" binding:"omitempty,notblank,safetext,max=100"`
	Description *string `json:"description" binding:"omitempty,safetext,max=2000"`
	SortOrder   *int    `json:"sort_order" binding:"omitempty,min=0,max=1000000"`
}

type TeamRequest struct {
	Name string `json:"name" binding:"required,notblank,safetext,max=50"`
}

type JoinCampaignRequest struct {
	WarbandID string `json:"warband_id" binding:"required,max=36"`
	TeamID    string `json:"team_id" binding:"required,max=36"`
	// JoinCode is required unless the caller owns the campaign.
	JoinCode string `json:"join_code" binding:"omitempty,max=32"`
}

type JoinCodeResponse struct {
	JoinCode string `json:"join_code"`
}

type ChangeTeamRequest struct {
	TeamID string `json:"team_id" binding:"required,max=36"`
}
