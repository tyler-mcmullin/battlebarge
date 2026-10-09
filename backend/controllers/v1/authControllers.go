package v1

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"

	"battlebarge/db"
	"battlebarge/middleware"
	"battlebarge/models"
	"battlebarge/repositories"
)

// userInsertConflict maps a unique violation on the users table to a 409 message, using the
// constraint's name so an email conflict is not reported as a username one
//
// Arguments: err (error) - the error from inserting a user row
//
// Returns: string - the message to send the client; bool - true if err is a unique-constraint conflict on the username (any capitalization) or email
func userInsertConflict(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return "", false
	}

	switch pgErr.ConstraintName {
	case "users_username_key", "users_username_lower_key":
		return "username already taken", true
	case "users_email_key":
		return "email already exists", true
	}
	return "", false
}

// RegisterUser handles POST /auth/register. Creates a Firebase user and a matching Postgres user, rolling back the Firebase user if the database insert fails. Usernames are unique ignoring case. The new user must verify their email before the API will accept their token
//
// Arguments: gin context
//
// Returns: None (responds 201 with the new user ID and email_verification_required; 400 on bad input, including a username over 50 or email over 255 characters; 409 if email or username is taken)
func RegisterUser(c *gin.Context) {
	var req models.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	params := (&auth.UserToCreate{}).
		Email(req.Email).
		Password(req.Password)

	firebaseUser, err := db.AuthClient.CreateUser(c.Request.Context(), params)
	if err != nil {
		if strings.Contains(err.Error(), "email already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": "email already exists"})
			return
		}
		serverError(c, err)
		return
	}

	user := models.User{
		ID:        firebaseUser.UID,
		Email:     req.Email,
		Username:  req.Username,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := repositories.CreateUser(user); err != nil {
		// undo the Firebase account; if that fails, say so in the log, since
		// the orphaned account would need cleaning up by hand
		if delErr := db.AuthClient.DeleteUser(c.Request.Context(), firebaseUser.UID); delErr != nil {
			log.Printf("register: could not roll back Firebase user %s: %v", firebaseUser.UID, delErr)
		}

		if msg, ok := userInsertConflict(err); ok {
			c.JSON(http.StatusConflict, gin.H{"error": msg})
			return
		}
		serverError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":                     "user created",
		"user_id":                     firebaseUser.UID,
		"email_verification_required": middleware.EmailVerificationRequired(),
	})
}
