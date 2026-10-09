package routes

import (
	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

// GetUserControllers gets user controllers
//
// Arguments: gin router
//
// Returns: None
func GetUserControllers(r *gin.Engine) {
	group := r.Group("/users")
	group.Use(middleware.RequireAuth(), middleware.LoadUser())
	group.GET("/me", controllers.GetCurrentUser)

}
