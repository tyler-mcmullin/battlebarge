package v1

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"battlebarge/models"
	"battlebarge/repositories"
)

// Arguments: c (gin context); campaignID (string) - the :id path parameter; uid (string) - authenticated user ID
//
// Returns: bool - true if campaignID is a valid UUID and uid owns that campaign; otherwise false after responding 404 (or 500 on a database error)
//
// Guards campaign-owner-only routes. Non-owners get 404 so a campaign's existence is not revealed
func requireCampaignOwner(c *gin.Context, campaignID string, uid string) bool {
	if !validUUID(campaignID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return false
	}

	owns, err := repositories.IsCampaignOwner(campaignID, uid)
	if err != nil {
		serverError(c, err)
		return false
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return false
	}

	return true
}

// Arguments: c (gin context)
//
// Returns: None (responds 201 with the campaign; 400 on bad input; 401 if unauthenticated; 409 if the user already owns the maximum number of campaigns)
//
// POST /campaigns/create. Creates a campaign owned by the authenticated user, applying defaults for omitted settings
func CreateCampaign(c *gin.Context) {
	var req models.CreateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	uid, ok := requireUID(c)
	if !ok {
		return
	}

	now := time.Now()
	campaign := models.Campaign{
		ID:        uuid.New(),
		OwnerID:   uid,
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
		Chapters:  []models.CampaignChapter{},
		Teams:     []models.CampaignTeam{},
		Warbands:  []models.CampaignWarband{},
	}
	if req.Description != nil {
		campaign.Description = *req.Description
	}
	if req.PointsPerWin != nil {
		campaign.Settings.PointsPerWin = *req.PointsPerWin
	}
	if req.PointsPerLoss != nil {
		campaign.Settings.PointsPerLoss = *req.PointsPerLoss
	}
	if req.StartingRequisition != nil {
		campaign.Settings.StartingRequisition = *req.StartingRequisition
	}

	if err := repositories.CreateCampaign(campaign); err != nil {
		if errors.Is(err, repositories.ErrLimitReached) {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("limit reached: at most %d campaigns per user", repositories.MaxCampaignsPerUser)})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusCreated, campaign)
}

// Arguments: c (gin context)
//
// Returns: None (responds 200 with the campaigns the user owns or has a warband in; 401 if unauthenticated)
//
// GET /campaigns. Lists the authenticated user's campaigns
func GetMyCampaigns(c *gin.Context) {
	uid, ok := requireUID(c)
	if !ok {
		return
	}

	campaigns, err := repositories.GetCampaignsForUser(uid)
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, campaigns)
}

// Arguments: c (gin context)
//
// Returns: None (responds 200 with the campaign; 404 if not found or the id is malformed)
//
// GET /campaigns/:id. Public endpoint returning a campaign with its chapters, teams, and member warbands
func GetCampaign(c *gin.Context) {
	id := c.Param("id")
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}

	campaign, err := repositories.GetCampaignByID(id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// Arguments: c (gin context)
//
// Returns: None (responds 200 with the updated campaign; 400 on bad input; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// PATCH /campaigns/:id. Partially updates a campaign owned by the authenticated user
func UpdateCampaign(c *gin.Context) {
	id := c.Param("id")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}

	var req models.UpdateCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	campaign, err := repositories.UpdateCampaign(id, uid, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// Arguments: c (gin context)
//
// Returns: None (responds 204; 401 if unauthenticated; 404 if not found, not owned, or the id is malformed)
//
// DELETE /campaigns/:id. Deletes a campaign owned by the authenticated user along with its chapters, teams, and memberships
func DeleteCampaign(c *gin.Context) {
	id := c.Param("id")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}

	if err := repositories.DeleteCampaign(id, uid); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
			return
		}
		serverError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Arguments: c (gin context)
//
// Returns: None (responds 201 with the chapter; 400 on bad input; 401 if unauthenticated; 404 if the campaign is not found, not owned, or the id is malformed; 409 if the campaign already has the maximum number of chapters)
//
// POST /campaigns/:id/chapters. Adds a chapter to a campaign owned by the authenticated user, appended after the last chapter unless sort_order is given
func CreateChapter(c *gin.Context) {
	id := c.Param("id")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}

	var req models.CreateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	now := time.Now()
	chapter := models.CampaignChapter{
		ID:         uuid.New(),
		CampaignID: uuid.MustParse(id),
		Title:      req.Title,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if req.Description != nil {
		chapter.Description = *req.Description
	}
	if req.SortOrder != nil {
		chapter.SortOrder = *req.SortOrder
	}

	created, err := repositories.AddChapter(chapter, req.SortOrder == nil)
	if err != nil {
		if errors.Is(err, repositories.ErrLimitReached) {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("limit reached: at most %d chapters per campaign", repositories.MaxChaptersPerCampaign)})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// Arguments: c (gin context)
//
// Returns: None (responds 200 with the updated chapter; 400 on bad input; 401 if unauthenticated; 404 if the campaign or chapter is not found, not owned, or an id is malformed)
//
// PATCH /campaigns/:id/chapters/:chapterId. Partially updates a chapter of a campaign owned by the authenticated user
func UpdateChapter(c *gin.Context) {
	id := c.Param("id")
	chapterID := c.Param("chapterId")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}
	if !validUUID(chapterID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
		return
	}

	var req models.UpdateChapterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	chapter, err := repositories.UpdateChapter(id, chapterID, req)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, chapter)
}

// Arguments: c (gin context)
//
// Returns: None (responds 204; 401 if unauthenticated; 404 if the campaign or chapter is not found, not owned, or an id is malformed)
//
// DELETE /campaigns/:id/chapters/:chapterId. Deletes a chapter of a campaign owned by the authenticated user
func DeleteChapter(c *gin.Context) {
	id := c.Param("id")
	chapterID := c.Param("chapterId")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}
	if !validUUID(chapterID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
		return
	}

	if err := repositories.DeleteChapter(id, chapterID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "chapter not found"})
			return
		}
		serverError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Arguments: c (gin context)
//
// Returns: None (responds 201 with the team; 400 on bad input; 401 if unauthenticated; 404 if the campaign is not found, not owned, or the id is malformed; 409 if the campaign already has a team with that name or the maximum number of teams)
//
// POST /campaigns/:id/teams. Adds a named team to a campaign owned by the authenticated user. A campaign can have any number of teams
func CreateTeam(c *gin.Context) {
	id := c.Param("id")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}

	var req models.TeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	now := time.Now()
	team := models.CampaignTeam{
		ID:         uuid.New(),
		CampaignID: uuid.MustParse(id),
		Name:       req.Name,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := repositories.CreateTeam(team); err != nil {
		if pgErrCode(err) == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "team name already used in this campaign"})
			return
		}
		if errors.Is(err, repositories.ErrLimitReached) {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("limit reached: at most %d teams per campaign", repositories.MaxTeamsPerCampaign)})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusCreated, team)
}

// Arguments: c (gin context)
//
// Returns: None (responds 200 with the renamed team; 400 on bad input; 401 if unauthenticated; 404 if the campaign or team is not found, not owned, or an id is malformed; 409 if the name is already used in the campaign)
//
// PATCH /campaigns/:id/teams/:teamId. Renames a team of a campaign owned by the authenticated user
func RenameTeam(c *gin.Context) {
	id := c.Param("id")
	teamID := c.Param("teamId")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}
	if !validUUID(teamID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	var req models.TeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	team, err := repositories.RenameTeam(id, teamID, req.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
			return
		}
		if pgErrCode(err) == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "team name already used in this campaign"})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, team)
}

// Arguments: c (gin context)
//
// Returns: None (responds 204; 401 if unauthenticated; 404 if the campaign or team is not found, not owned, or an id is malformed; 409 if warbands are still on the team)
//
// DELETE /campaigns/:id/teams/:teamId. Deletes an empty team of a campaign owned by the authenticated user
func DeleteTeam(c *gin.Context) {
	id := c.Param("id")
	teamID := c.Param("teamId")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireCampaignOwner(c, id, uid) {
		return
	}
	if !validUUID(teamID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
		return
	}

	if err := repositories.DeleteTeam(id, teamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "team not found"})
			return
		}
		if pgErrCode(err) == "23503" {
			c.JSON(http.StatusConflict, gin.H{"error": "team still has warbands; move them first"})
			return
		}
		serverError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// Arguments: c (gin context)
//
// Returns: None (responds 201 with the updated campaign; 400 on bad input or malformed warband_id/team_id; 401 if unauthenticated; 404 if the campaign id is malformed, the warband is not owned by the user, or the team is not in the campaign; 409 if the warband is already in the campaign or the campaign has the maximum number of warbands)
//
// POST /campaigns/:id/warbands. Joins a warband owned by the authenticated user to a campaign on one of its teams. Anyone who knows the campaign ID can join
func JoinCampaign(c *gin.Context) {
	id := c.Param("id")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !validUUID(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return
	}

	var req models.JoinCampaignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	if !validUUID(req.WarbandID) || !validUUID(req.TeamID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid warband_id or team_id"})
		return
	}

	owns, err := repositories.IsWarbandOwner(req.WarbandID, uid)
	if err != nil {
		serverError(c, err)
		return
	}
	if !owns {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found"})
		return
	}

	if err := repositories.JoinCampaign(id, req.WarbandID, req.TeamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "team not found in this campaign"})
			return
		}
		if pgErrCode(err) == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "warband already in this campaign"})
			return
		}
		if errors.Is(err, repositories.ErrLimitReached) {
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("limit reached: at most %d warbands per campaign", repositories.MaxWarbandsPerCampaign)})
			return
		}
		serverError(c, err)
		return
	}

	campaign, err := repositories.GetCampaignByID(id)
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusCreated, campaign)
}

// Arguments: c (gin context); campaignID (string) - campaign ID; warbandID (string) - warband ID from the path; uid (string) - authenticated user ID
//
// Returns: bool - true if the ids are valid and uid owns the campaign or the warband; otherwise false after responding 404 (or 500 on a database error)
//
// Guards membership changes: the campaign owner or the warband's owner may move or remove a member
func requireMembershipAccess(c *gin.Context, campaignID string, warbandID string, uid string) bool {
	if !validUUID(campaignID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "campaign not found"})
		return false
	}
	if !validUUID(warbandID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found in this campaign"})
		return false
	}

	ownsCampaign, err := repositories.IsCampaignOwner(campaignID, uid)
	if err != nil {
		serverError(c, err)
		return false
	}
	if ownsCampaign {
		return true
	}

	ownsWarband, err := repositories.IsWarbandOwner(warbandID, uid)
	if err != nil {
		serverError(c, err)
		return false
	}
	if !ownsWarband {
		c.JSON(http.StatusNotFound, gin.H{"error": "warband not found in this campaign"})
		return false
	}

	return true
}

// Arguments: c (gin context)
//
// Returns: None (responds 200 with the updated campaign; 400 on bad input; 401 if unauthenticated; 404 if the warband or team is not in the campaign, the user owns neither the campaign nor the warband, or an id is malformed)
//
// PATCH /campaigns/:id/warbands/:warbandId. Moves a warband to another team; allowed for the campaign owner or the warband's owner
func ChangeWarbandTeam(c *gin.Context) {
	id := c.Param("id")
	warbandID := c.Param("warbandId")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireMembershipAccess(c, id, warbandID, uid) {
		return
	}

	var req models.ChangeTeamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if !validUUID(req.TeamID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid team_id"})
		return
	}

	if err := repositories.SetWarbandTeam(id, warbandID, req.TeamID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warband or team not found in this campaign"})
			return
		}
		serverError(c, err)
		return
	}

	campaign, err := repositories.GetCampaignByID(id)
	if err != nil {
		serverError(c, err)
		return
	}

	c.JSON(http.StatusOK, campaign)
}

// Arguments: c (gin context)
//
// Returns: None (responds 204; 401 if unauthenticated; 404 if the warband is not in the campaign, the user owns neither the campaign nor the warband, or an id is malformed)
//
// DELETE /campaigns/:id/warbands/:warbandId. Removes a warband from a campaign; allowed for the campaign owner or the warband's owner
func LeaveCampaign(c *gin.Context) {
	id := c.Param("id")
	warbandID := c.Param("warbandId")
	uid, ok := requireUID(c)
	if !ok {
		return
	}
	if !requireMembershipAccess(c, id, warbandID, uid) {
		return
	}

	if err := repositories.LeaveCampaign(id, warbandID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "warband not found in this campaign"})
			return
		}
		serverError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}
