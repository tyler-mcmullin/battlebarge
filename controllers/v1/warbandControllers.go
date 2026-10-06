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
// Returns: None (responds 201 with the warband; 400 on bad input; 401 if unauthenticated)
//
// POST /warbands/create. Creates a warband owned by the authenticated user, applying defaults for omitted optional fields
func CreateWarband(c *gin.Context) {
	var req models.CreateWarbandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}

	now := time.Now()

	warband := models.Warband{
		ID:                uuid.New(),
		UserID:            uid,
		Name:              req.Name,
		Faction:           "",
		Description:       "",
		CrusadePoints:     0,
		RequisitionPoints: 0,
		SupplyLimit:       0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if req.Faction != nil {
		warband.Faction = *req.Faction
	}
	if req.Description != nil {
		warband.Description = *req.Description
	}
	if req.RequisitionPoints != nil {
		warband.RequisitionPoints = *req.RequisitionPoints
	}
	if req.SupplyLimit != nil {
		warband.SupplyLimit = *req.SupplyLimit
	}

	if err := repositories.CreateWarband(warband); err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusCreated, warband)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the user's warbands)
//
// GET /warbands. Lists all warbands owned by the authenticated user
func GetAllWarbands(c *gin.Context) {
	uid := c.GetString(middleware.ContextUIDKey)

	warbands, err := repositories.GetAllWarbands(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch warbands"})
		return
	}

	c.JSON(http.StatusOK, warbands)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the warband; 404 if not found or the id is malformed)
//
// GET /warbands/:id. Public endpoint returning a single warband with its units
func GetWarbandByID(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
		return
	}

	warband, err := repositories.GetWarbandByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch warband"})
		return
	}

	c.JSON(http.StatusOK, warband)
}

// Arguments: gin context
//
// Returns: None (responds 200 with the updated warband; 400 on bad input; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// PATCH /warbands/:id. Partially updates a warband owned by the authenticated user
func UpdateWarband(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
		return
	}

	var req models.UpdateWarbandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	warband, err := repositories.UpdateWarband(id, uid, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update warband"})
		return
	}

	c.JSON(http.StatusOK, warband)
}

// Arguments: gin context
//
// Returns: None (responds 204; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// DELETE /warbands/:id. Deletes a warband owned by the authenticated user
func DeleteWarband(c *gin.Context) {
	id := c.Param("id")
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
		return
	}

	err := repositories.DeleteWarband(id, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete warband"})
		return
	}

	c.Status(http.StatusNoContent)
}
