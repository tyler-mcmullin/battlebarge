package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"battlebarge/middleware"
)

func GetCurrentUser(c *gin.Context) {
	user, exists := c.Get(middleware.ContextUserKey)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found in context"})
		return
	}
	c.JSON(http.StatusOK, user)
}
