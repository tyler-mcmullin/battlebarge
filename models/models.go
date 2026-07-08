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
	ID                uuid.UUID `json:"id" db:"id"`
	UserID            string    `json:"user_id" db:"user_id"`
	Name              string    `json:"name" db:"name"`
	Faction           string    `json:"faction" db:"faction"`
	Description       string    `json:"description" db:"description"`
	Units             []Unit    `json:"units" db:"units"`
	NumUnits          int       `json:"num_units" db:"num_units"`
	TotalPointsCost   int       `json:"total_points_cost" db:"total_points_cost"`
	CrusadePoints     int       `json:"crusade_points" db:"crusade_points"`
	RequisitionPoints int       `json:"requisition_points" db:"requisition_points"`
	SupplyLimit       int       `json:"supply_limit" db:"supply_limit"`
	SupplyCost        int       `json:"supply_cost" db:"supply_cost"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type Unit struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UnitName      string    `json:"unit_name" db:"unit_name"`
	NarrativeName string    `json:"narrative_name" db:"narrative_name"`
	Bio           string    `json:"bio" db:"bio"`
	Points        int       `json:"points" db:"points"`
	XP            int       `json:"xp" db:"xp"`
	Kills         int       `json:"kills" db:"kills"`
	Experience    int       `json:"experience" db:"experience"`
	Perks         []Perk    `json:"perks" db:"perks"`
}

type Perk struct {
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	IsScar      bool   `json:"is_scar" db:"is_scar"`
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
	UnitName      string  `json:"unit_name" binding:"required"`
	NarrativeName *string `json:"narrative_name"`
	Bio           *string `json:"bio"`
	Points        *int    `json:"points"`
}
