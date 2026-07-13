package routes

import (
	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
)

// Arguments: gin router
// Returns: None
// Gets auth controllers
func GetAuthControllers(r *gin.Engine) {
	group := r.Group("/auth")
	group.POST("/register", controllers.RegisterUser)

}
