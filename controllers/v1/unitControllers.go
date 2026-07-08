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

func CreateUnit(c *gin.Context) {
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

	owns, err := repositories.IsWarbandOwner(req.WarbandID, uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
		return
	}

	now := time.Now()
	warbandUUID, err := uuid.Parse(req.WarbandID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid warband_id"})
		return
	}

	unit := models.Unit{
		ID:            uuid.New(),
		WarbandID:     warbandUUID,
		UnitName:      req.UnitName,
		NarrativeName: "",
		Bio:           "",
		Points:        0,
		Kills:         0,
		Experience:    0,
		Perks:         []models.Perk{},
		CreatedAt:     now,
		UpdatedAt:     now,
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

	if err := repositories.CreateUnit(unit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, unit)
}

func GetUnit(c *gin.Context) {
	id := c.Param("id")

	unit, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, unit)
}

func DeleteUnit(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}

	existing, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	if err := repositories.DeleteUnit(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete unit"})
		return
	}

	c.Status(http.StatusNoContent)
}

func UpdateUnit(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}

	var req models.UpdateUnitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	existing, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.UpdateUnit(id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, unit)
}

func AddUnitKills(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}

	var req models.IncrementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	existing, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.IncrementUnitKills(id, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, unit)
}

func AddUnitXP(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}

	var req models.IncrementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	existing, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.IncrementUnitXP(id, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, unit)
}
