package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"battlebarge/middleware"
	"battlebarge/models"
	"battlebarge/testutil"
)

func init() { gin.SetMode(gin.TestMode) }

// do runs one request through a router with the given middleware and a
// handler that records whether it was reached.
func do(t *testing.T, header string, mw ...gin.HandlerFunc) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	reached := false
	r := gin.New()
	r.Use(mw...)
	r.GET("/", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, reached
}

// Only the rejection paths are covered: accepting a token needs a Firebase
// client (db.AuthClient), which tests don't create.
func TestRequireAuth_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"no scheme", "sometoken"},
		{"wrong scheme", "Basic abc123"},
		{"bearer without token", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, reached := do(t, tt.header, middleware.RequireAuth())
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", w.Code)
			}
			if reached {
				t.Error("handler should not run when auth fails")
			}
		})
	}
}

func TestLoadUser_MissingUID(t *testing.T) {
	w, reached := do(t, "", middleware.LoadUser())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if reached {
		t.Error("handler should not run without a uid")
	}
}

func setUID(uid string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Set(middleware.ContextUIDKey, uid) }
}

func TestLoadUser_UnknownUser(t *testing.T) {
	testutil.SetupDB(t)

	w, reached := do(t, "", setUID("ghost"), middleware.LoadUser())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if reached {
		t.Error("handler should not run for an unknown user")
	}
}

func TestLoadUser_AttachesUser(t *testing.T) {
	testutil.SetupDB(t)
	want := testutil.InsertUser(t, "real")

	var got models.User
	r := gin.New()
	r.Use(setUID("real"), middleware.LoadUser())
	r.GET("/", func(c *gin.Context) {
		got = c.MustGet(middleware.ContextUserKey).(models.User)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got.ID != want.ID || got.Username != want.Username {
		t.Errorf("context user = %+v, want %+v", got, want)
	}
}
