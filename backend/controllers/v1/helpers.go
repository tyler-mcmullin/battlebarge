package v1

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"battlebarge/middleware"
)

// validUUID reports whether id is a well-formed UUID, so malformed IDs can be rejected
// before they reach Postgres (where the uuid cast would fail with a 500)
//
// Arguments: id (string) - a path or body parameter that should be a UUID
//
// Returns: bool - true if id parses as a UUID
func validUUID(id string) bool {
	_, err := uuid.Parse(id)
	return err == nil
}

// serverError logs err and responds 500 without exposing database or Firebase error text to the client
//
// Arguments: c (gin context); err (error) - the underlying failure
//
// Returns: None (responds 500 with a generic message)
func serverError(c *gin.Context, err error) {
	log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}

// pgErrCode extracts the SQLSTATE so controllers can map constraint violations to 4xx responses
//
// Arguments: err (error) - an error returned from a repository call
//
// Returns: string - the Postgres SQLSTATE code (e.g. "23505" for a unique violation), or "" if err is not a Postgres error
func pgErrCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// requireUID reads the uid set by the auth middleware, responding 401 if it is missing
//
// Arguments: c (gin context)
//
// Returns: string - the authenticated user's ID; bool - false if there is none (a 401 has already been written)
func requireUID(c *gin.Context) (string, bool) {
	uid := c.GetString(middleware.ContextUIDKey)
	if uid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context"})
		return "", false
	}
	return uid, true
}
