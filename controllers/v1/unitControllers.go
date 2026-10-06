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

// Arguments: gin context
//
// Returns: None (responds 201 with the unit; 400 on bad input or a malformed warband_id; 401 if unauthenticated; 404 if the warband is not found or not owned)
//
// POST /units/create. Creates a unit in a warband owned by the authenticated user
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

	warbandUUID, err := uuid.Parse(req.WarbandID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid warband_id"})
		return
	}

	owns, err := repositories.IsWarbandOwner(req.WarbandID, uid)
	if err != nil {
		serverError(c, err)
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
		return
	}

	now := time.Now()

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
		serverError(c, err)
		return
	}

	c.JSON(http.StatusCreated, unit)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the unit; 404 if not found or the id is malformed)
//
// GET /units/:id. Public endpoint returning a single unit
func GetUnit(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, unit)
}

// Arguments: gin context
//
// Returns: None (responds 204; 401 if unauthenticated; 404 if not found, its warband is not owned, or the id is malformed)
//
// DELETE /units/:id. Deletes a unit in a warband owned by the authenticated user
func DeleteUnit(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	existing, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		serverError(c, err)
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		serverError(c, err)
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

// Arguments: gin context
//
// Returns: None (responds 200 with the updated unit; 400 on bad input; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// PATCH /units/:id. Partially updates a unit in a warband owned by the authenticated user
func UpdateUnit(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
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
		serverError(c, err)
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		serverError(c, err)
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.UpdateUnit(id, req)
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, unit)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the updated unit; 400 on bad input; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// PATCH /units/:id/kills. Adds the requested amount to a unit's kills
func AddUnitKills(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
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
		serverError(c, err)
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		serverError(c, err)
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.IncrementUnitKills(id, req.Amount)
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, unit)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the updated unit; 400 on bad input; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// PATCH /units/:id/xp. Adds the requested amount to a unit's experience
func AddUnitXP(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
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
		serverError(c, err)
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		serverError(c, err)
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.IncrementUnitXP(id, req.Amount)
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, unit)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the updated unit; 400 on bad input; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// PATCH /units/:id/perk. Adds a perk or scar with a newly generated ID to a unit
func AddUnitPerk(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	var req models.AddPerkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	req.ID = uuid.New()

	existing, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		serverError(c, err)
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		serverError(c, err)
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.AddUnitPerk(id, req)
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, unit)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the updated unit; 401 if unauthenticated; 404 if the unit or perk is not found, not owned, or the id is malformed)
//
// DELETE /units/:id/perk/:perkId. Removes a perk from a unit
func DeleteUnitPerk(c *gin.Context) {
	id := c.Param("id")
	perkID := c.Param("perkId")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	existing, err := repositories.GetUnitByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
			return
		}
		serverError(c, err)
		return
	}

	owns, err := repositories.IsWarbandOwner(existing.WarbandID.String(), uid)
	if err != nil {
		serverError(c, err)
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "unit not found"})
		return
	}

	unit, err := repositories.DeleteUnitPerk(id, perkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "perk not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete perk"})
		return
	}

	c.JSON(http.StatusOK, unit)
}
