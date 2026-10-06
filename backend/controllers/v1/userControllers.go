package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"battlebarge/middleware"
)

// Arguments: gin context
//
// Returns: None (responds 200 with the user; 500 if no user is in context)
//
// GET /users/me. Returns the user loaded into context by the LoadUser middleware
func GetCurrentUser(c *gin.Context) {
	user, exists := c.Get(middleware.ContextUserKey)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found in context"})
		return
	}
	c.JSON(http.StatusOK, user)
}
