package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"battlebarge/db"
	"battlebarge/repositories"
)

const (
	ContextUIDKey  = "uid"
	ContextUserKey = "user"
)

// RequireAuth verifies the Firebase ID token sent in the Authorization header
// (format: "Bearer <token>") and attaches the verified UID to the request
// context. It does NOT touch Postgres - routes that only need to know who
// the caller is (e.g. to stamp ownership on a new record) should use this
// alone. Routes that need full user data should additionally chain LoadUser().
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}
		idToken := parts[1]

		token, err := db.AuthClient.VerifyIDToken(c.Request.Context(), idToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		c.Set(ContextUIDKey, token.UID)

		c.Next()
	}
}

// LoadUser fetches the full User record from Postgres using the UID attached
// to context by RequireAuth, and attaches it to context. Must be chained
// after RequireAuth(). Use this only on routes that actually need profile
// data (username, email, etc.) - otherwise prefer RequireAuth() alone.
func LoadUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString(ContextUIDKey)
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context - is RequireAuth chained before LoadUser?"})
			return
		}

		user, err := repositories.GetUserByID(uid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		c.Set(ContextUserKey, user)

		c.Next()
	}
}
