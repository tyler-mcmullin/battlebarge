package v1

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"battlebarge/middleware"
	"battlebarge/models"
	"battlebarge/repositories"
)

func AddUnit(c *gin.Context) {
	warbandID := c.Param("id")

	var req models.CreateUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}

	warband, err := repositories.GetWarbandByIDForOwner(warbandID, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	unit := models.Unit{
		ID:            uuid.New(),
		UnitName:      req.UnitName,
		NarrativeName: "",
		Bio:           "",
		Points:        0,
		Kills:         0,
		Experience:    0,
		Perks:         []models.Perk{},
	}

	if req.NarrativeName != nil {
		unit.NarrativeName = *req.NarrativeName
	}
	if req.Bio != nil {
		unit.Bio = *req.Bio
	}
	if req.Points != nil {
		unit.Points = *req.Points
	}

	warband.Units = append(warband.Units, unit)
	warband.NumUnits = len(warband.Units)

	total := 0
	for _, u := range warband.Units {
		total += u.Points
	}
	warband.TotalPointsCost = total

	warband.UpdatedAt = time.Now()

	if err := repositories.SaveWarband(warband); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, warband)
}
