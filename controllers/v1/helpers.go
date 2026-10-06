package v1

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Arguments: id (string) - a path or body parameter that should be a UUID
//
// Returns: bool - true if id parses as a UUID
//
// Reports whether id is a well-formed UUID, so malformed IDs can be rejected
// before they reach Postgres (where the uuid cast would fail with a 500)
func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// Arguments: c (gin context); err (error) - the underlying failure
//
// Returns: None (responds 500 with a generic message)
//
// Logs err and responds 500 without exposing database or Firebase error text to the client
func serverError(c *gin.Context, err error) {
	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
