package routes

import (
	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

func GetUnitControllers(r *gin.Engine) {
	group := r.Group("/units")

	// Public
	group.GET("/:id", controllers.GetUnit)

	// Require Auth
	priv := group.Group("")
	priv.Use(middleware.RequireAuth())
	priv.POST("/create", controllers.CreateUnit)
}
