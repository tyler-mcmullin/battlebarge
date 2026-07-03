package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	Username  string    `json:"username" db:"username"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Warband struct {
	ID                uuid.UUID `json:"id"`
	UserID            string    `json:"user_id"`
	Name              string    `json:"name"`
	Faction           string    `json:"faction"`
	Description       string    `json:"description"`
	Units             []Unit    `json:"units"`
	NumUnits          int       `json:"num_units"`
	TotalPointsCost   int       `json:"total_points_cost"`
	CrusadePoints     int       `json:"crusade_points"`
	RequisitionPoints int       `json:"requisition_points"`
	SupplyLimit       int       `json:"supply_limit"`
	SupplyCost        int       `json:"supply_cost"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type Unit struct {
	ID            uuid.UUID `json:"id"`
	UnitName      string    `json:"unit_name"`
	NarrativeName string    `json:"narrative_name"`
	Points        int       `json:"points"`
	Experience    int       `json:"experience"`
	Perks         []Perk    `json:"perks"`
}

type Perk struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateWarbandRequest struct {
	Name string `json:"name" binding:"required"`
}
