package routes

import (
	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

func GetUnitControllers(r *gin.Engine) {
	group := r.Group("/units")

	// Public
	//group.GET("/:id", controllers.GetUnitByID)

	// Require Auth
	priv := group.Group("")
	priv.Use(middleware.RequireAuth())
	//priv.GET("", controllers.GetAllUnits)
	priv.POST("/create", controllers.CreateWarband)
	//priv.PATCH("/:id", controllers.UpdateUnit)
	//priv.DELETE("/:id", controllers.DeleteUnit)
}
