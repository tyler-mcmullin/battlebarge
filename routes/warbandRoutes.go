package routes

import (
	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

func GetWarbandControllers(r *gin.Engine) {
	group := r.Group("/warbands")
	group.Use(middleware.RequireAuth())
	group.POST("/create", controllers.CreateWarband)
}
