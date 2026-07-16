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
// Gets user controllers
func GetUserControllers(r *gin.Engine) {
	group := r.Group("/users")
	group.Use(middleware.RequireAuth(), middleware.LoadUser())
	group.GET("/me", controllers.GetCurrentUser)

}
