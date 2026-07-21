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

// Request Structs
type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateWarbandRequest struct {
	Name              string  `json:"name" binding:"required"`
	Faction           *string `json:"faction"`
	Description       *string `json:"description"`
	RequisitionPoints *int    `json:"requisition_points"`
	SupplyLimit       *int    `json:"supply_limit"`
}

type UpdateWarbandRequest struct {
	Name              *string `json:"name"`
	Faction           *string `json:"faction"`
	Description       *string `json:"description"`
	RequisitionPoints *int    `json:"requisition_points"`
	SupplyLimit       *int    `json:"supply_limit"`
}

type CreateUnitRequest struct {
	WarbandID     string  `json:"warband_id" binding:"required"`
	UnitName      string  `json:"unit_name" binding:"required"`
	NarrativeName *string `json:"narrative_name"`
	Bio           *string `json:"bio"`
	Points        *int    `json:"points"`
}

type UpdateUnitRequest struct {
	UnitName      *string `json:"unit_name"`
	NarrativeName *string `json:"narrative_name"`
	Bio           *string `json:"bio"`
	Points        *int    `json:"points"`
}

type IncrementRequest struct {
	Amount int `json:"amount" binding:"required"`
}

type AddPerkRequest struct {
	ID          uuid.UUID `json:"perk_id"`
	Name        string    `json:"name" binding:"required"`
	Description *string   `json:"description"`
	IsScar      bool      `json:"is_scar"`
}
