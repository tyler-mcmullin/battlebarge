package v1_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
	"battlebarge/models"
)

func TestGetCurrentUser(t *testing.T) {
	r := gin.New()
	r.GET("/me", func(c *gin.Context) {
		c.Set(middleware.ContextUserKey, models.User{ID: uuid.New(), FirebaseId: "firebase-secret", Username: "alice"})
	}, controllers.GetCurrentUser)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me", nil))
	expect(t, w, http.StatusOK)
	if got := w.Body.String(); !contains(got, `"username":"alice"`) {
		t.Errorf("body = %s, want the user's username", got)
	}
	if contains(w.Body.String(), "firebase") {
		t.Errorf("body = %s, must not expose the Firebase UID", w.Body.String())
	}
}

func TestGetCurrentUser_NoUserInContext(t *testing.T) {
	r := gin.New()
	r.GET("/me", controllers.GetCurrentUser)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/me", nil))
	expect(t, w, http.StatusInternalServerError)
}
