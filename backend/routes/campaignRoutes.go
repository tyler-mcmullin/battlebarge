package routes

import (
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
)

// GetCampaignControllers gets campaign controllers
//
// Arguments: gin router
//
// Returns: None
func GetCampaignControllers(r *gin.Engine) {
	group := r.Group("/campaigns")

	// Public
	group.GET("/:id", controllers.GetCampaign)

	// Require Auth
	priv := group.Group("")
	priv.Use(middleware.RequireAuth())
	priv.GET("", controllers.GetMyCampaigns)
	priv.POST("/create", controllers.CreateCampaign)
	priv.PATCH("/:id", controllers.UpdateCampaign)
	priv.DELETE("/:id", controllers.DeleteCampaign)

	priv.POST("/:id/chapters", controllers.CreateChapter)
	priv.PATCH("/:id/chapters/:chapterId", controllers.UpdateChapter)
	priv.DELETE("/:id/chapters/:chapterId", controllers.DeleteChapter)

	priv.POST("/:id/teams", controllers.CreateTeam)
	priv.PATCH("/:id/teams/:teamId", controllers.RenameTeam)
	priv.DELETE("/:id/teams/:teamId", controllers.DeleteTeam)

	priv.GET("/:id/join-code", controllers.GetCampaignJoinCode)
	priv.POST("/:id/join-code/rotate", controllers.RotateCampaignJoinCode)

	// Joining is the one route where a stranger can guess a secret (the join
	// code), so it gets its own tight per-IP limit: 10 attempts, then 1 per 6s.
	priv.POST("/:id/warbands", middleware.RateLimit(rate.Every(6*time.Second), 10), controllers.JoinCampaign)
	priv.PATCH("/:id/warbands/:warbandId", controllers.ChangeWarbandTeam)
	priv.DELETE("/:id/warbands/:warbandId", controllers.LeaveCampaign)
}
