package v1

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"battlebarge/middleware"
	"battlebarge/models"
	"battlebarge/repositories"
)

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
		Units:             []models.Unit{},
		NumUnits:          0,
		TotalPointsCost:   0,
		CrusadePoints:     0,
		RequisitionPoints: 0,
		SupplyLimit:       0,
		SupplyCost:        0,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := repositories.CreateWarband(warband); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, warband)
}
