package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

// Arguments: gin router
//
// Returns: None
//
// Gets auth controllers
func GetAuthControllers(r *gin.Engine) {
	group := r.Group("/auth")

	// Registration creates Firebase accounts, so it gets a much tighter limit
	// than the rest of the API: 5 at once, then one per minute per client IP.
	group.POST("/register", middleware.RateLimit(rate.Every(time.Minute), 5), controllers.RegisterUser)

}
