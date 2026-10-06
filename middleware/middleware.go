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
// context. Routes that only need to know who the caller is should use this
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
// data (username, email, etc.) otherwise use RequireAuth() alone for lightweight auth.
func LoadUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := c.GetString(ContextUIDKey)
		if uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing uid in context my require RequireAuth chained before LoadUser"})
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

// Arguments: raw (string) - comma-separated origins, e.g. the CORS_ALLOWED_ORIGINS env var
//
// Returns: []string - the origins with whitespace, empty entries, and trailing slashes removed
//
// Parses a comma-separated origin list for the CORS middleware
func ParseOrigins(raw string) []string {
	origins := []string{}
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// Arguments: allowedOrigins ([]string) - exact origins (scheme://host[:port]) that browsers may call the API from
//
// Returns: gin.HandlerFunc - the CORS middleware
//
// Lets browser frontends on the allowed origins call the API. Requests from an
// allowed origin get Access-Control-Allow-Origin; preflight (OPTIONS) requests
// are answered with 204 and the allowed methods and headers, or 403 if the
// origin is not allowed. Requests with no Origin header (curl, same-origin,
// server-to-server) are untouched. With an empty list no origin is allowed.
// Origins are matched exactly; "*" is deliberately not supported.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			c.Next()
			return
		}

		// The response depends on the Origin header, so caches must key on it.
		c.Writer.Header().Add("Vary", "Origin")

		isPreflight := c.Request.Method == http.MethodOptions &&
			c.GetHeader("Access-Control-Request-Method") != ""

		if !allowed[origin] {
			if isPreflight {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", origin)

		if isPreflight {
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
