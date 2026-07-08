package routes

import (
	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

func GetWarbandControllers(r *gin.Engine) {
	group := r.Group("/warbands")

	// Public
	group.GET("/:id", controllers.GetWarbandByID)

	// Require Auth
	priv := group.Group("")
	priv.Use(middleware.RequireAuth())
	priv.GET("", controllers.GetAllWarbands)
	priv.POST("/create", controllers.CreateWarband)
	priv.PATCH("/:id", controllers.UpdateWarband)
	priv.DELETE("/:id", controllers.DeleteWarband)
}
