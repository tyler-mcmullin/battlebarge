package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"battlebarge/db"
	"battlebarge/repositories"
)

const (
	ContextUIDKey  = "uid"
	ContextUserKey = "user"
)

// TokenVerifier verifies a Firebase ID token and returns its claims.
type TokenVerifier func(ctx context.Context, idToken string) (*auth.Token, error)

// verifyToken checks the token's signature and expiry and also that it has
// not been revoked and its user is not disabled. The revocation check costs
// one extra Firebase call per request, but means a revoked or disabled
// account loses access immediately instead of when the token expires.
var verifyToken TokenVerifier = func(ctx context.Context, idToken string) (*auth.Token, error) {
	return db.AuthClient.VerifyIDTokenAndCheckRevoked(ctx, idToken)
}

// requireVerifiedEmail makes RequireAuth reject accounts whose email address
// has not been verified. It is on by default; main.go can turn it off for
// local development with REQUIRE_EMAIL_VERIFICATION=false.
var requireVerifiedEmail = true

// SetRequireVerifiedEmail turns the verified-email requirement on or off
//
// Arguments: required (bool) - whether RequireAuth should reject unverified emails
//
// Returns: func() - restores the previous setting
func SetRequireVerifiedEmail(required bool) func() {
	prev := requireVerifiedEmail
	requireVerifiedEmail = required
	return func() { requireVerifiedEmail = prev }
}

// EmailVerificationRequired reports the verified-email requirement, so the registration response can tell clients what to expect
//
// Arguments: None
//
// Returns: bool - true if RequireAuth currently rejects unverified emails
func EmailVerificationRequired() bool {
	return requireVerifiedEmail
}

// SetTokenVerifier replaces the token verifier so tests can run RequireAuth without Firebase
//
// Arguments: v (TokenVerifier) - the verifier RequireAuth should use
//
// Returns: func() - restores the previous verifier
func SetTokenVerifier(v TokenVerifier) func() {
	prev := verifyToken
	verifyToken = v
	return func() { verifyToken = prev }
}

// RequireAuth verifies the Firebase ID token sent in the Authorization header
// (format: "Bearer <token>"), requires the account's email to be verified
// (unless turned off), and attaches the verified UID to the request context. Routes that only need to know who the caller is should use this
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

		token, err := verifyToken(c.Request.Context(), idToken)
		if err != nil {
			// the client only gets a generic message; the real reason goes to the log
			log.Printf("auth: token rejected for %s %s: %v", c.Request.Method, c.Request.URL.Path, err)
			msg := "invalid or expired token"
			switch {
			case auth.IsUserDisabled(err):
				msg = "account disabled"
			case auth.IsIDTokenRevoked(err):
				msg = "token revoked"
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": msg})
			return
		}

		// A signed-in user whose email is not verified is authenticated but not
		// allowed yet: 403 (not 401) so clients know to prompt for verification
		// instead of asking them to sign in again. The claim is refreshed when
		// the client fetches a new ID token after verifying.
		if requireVerifiedEmail {
			if verified, _ := token.Claims["email_verified"].(bool); !verified {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "email not verified"})
				return
			}
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

// ParseOrigins parses a comma-separated origin list for the CORS middleware
//
// Arguments: raw (string) - comma-separated origins, e.g. the CORS_ALLOWED_ORIGINS env var
//
// Returns: []string - the origins with whitespace, empty entries, and trailing slashes removed
func ParseOrigins(raw string) []string {
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimRight(strings.TrimSpace(o), "/")
		if o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}

// CORS lets browser frontends on the allowed origins call the API. Requests from an
// allowed origin get Access-Control-Allow-Origin; preflight (OPTIONS) requests
// are answered with 204 and the allowed methods and headers, or 403 if the
// origin is not allowed. Requests with no Origin header (curl, same-origin,
// server-to-server) are untouched. With an empty list no origin is allowed.
// Origins are matched exactly; "*" is deliberately not supported.
//
// Arguments: allowedOrigins ([]string) - exact origins (scheme://host[:port]) that browsers may call the API from
//
// Returns: gin.HandlerFunc - the CORS middleware
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

// MaxBodySize rejects requests whose declared Content-Length is over maxBytes with 413,
// and caps the bytes read from bodies with no declared length (chunked), so a
// client cannot make the server buffer an arbitrarily large body
//
// Arguments: maxBytes (int64) - largest request body to accept
//
// Returns: gin.HandlerFunc - the body size limit middleware
func MaxBodySize(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large"})
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimit limits each client IP with a token bucket and responds 429 with a
// Retry-After header when the bucket is empty. State is in memory, so it is
// per server instance. The client IP comes from c.ClientIP(), so configure
// trusted proxies correctly or all clients behind a proxy share one bucket.
//
// Arguments: limit (rate.Limit) - sustained requests per second per client IP; burst (int) - how many requests a client may make at once
//
// Returns: gin.HandlerFunc - the rate limiting middleware
func RateLimit(limit rate.Limit, burst int) gin.HandlerFunc {
	var mu sync.Mutex
	visitors := map[string]*visitor{}
	lastSweep := time.Now()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		mu.Lock()
		// drop clients not seen for 10 minutes, at most once a minute
		if now.Sub(lastSweep) > time.Minute {
			for k, v := range visitors {
				if now.Sub(v.lastSeen) > 10*time.Minute {
					delete(visitors, k)
				}
			}
			lastSweep = now
		}
		v, ok := visitors[ip]
		if !ok {
			v = &visitor{limiter: rate.NewLimiter(limit, burst)}
			visitors[ip] = v
		}
		v.lastSeen = now
		res := v.limiter.Reserve()
		delay := res.Delay()
		if delay > 0 {
			res.Cancel()
		}
		mu.Unlock()

		if delay > 0 {
			retry := int(delay.Seconds()) + 1
			c.Header("Retry-After", strconv.Itoa(retry))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests"})
			return
		}

		c.Next()
	}
}
