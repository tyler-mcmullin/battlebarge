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
// Gets campaign controllers
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

	priv.POST("/:id/warbands", controllers.JoinCampaign)
	priv.PATCH("/:id/warbands/:warbandId", controllers.ChangeWarbandTeam)
	priv.DELETE("/:id/warbands/:warbandId", controllers.LeaveCampaign)
}
