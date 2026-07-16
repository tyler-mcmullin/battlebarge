package routes

import (
	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

// Arguments: gin router
//
// Returns: None
//
// Gets unit controllers
func GetUnitControllers(r *gin.Engine) {
	group := r.Group("/units")

	// Public
	group.GET("/:id", controllers.GetUnit)

	// Require Auth
	priv := group.Group("")
	priv.Use(middleware.RequireAuth())
	priv.POST("/create", controllers.CreateUnit)
	priv.DELETE("/:id", controllers.DeleteUnit)
	priv.PATCH("/:id", controllers.UpdateUnit)
	priv.PATCH("/:id/kills", controllers.AddUnitKills)
	priv.PATCH("/:id/xp", controllers.AddUnitXP)
}
