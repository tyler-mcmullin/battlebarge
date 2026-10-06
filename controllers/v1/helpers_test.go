package v1_test

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	controllers "battlebarge/controllers/v1"
	"battlebarge/middleware"
	"battlebarge/models"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeAuth stands in for middleware.RequireAuth: it trusts the X-Test-UID
// header instead of verifying a Firebase token. With no header the uid is left
// unset, which is what the handlers' "missing uid" guard checks for.
func fakeAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if uid := c.GetHeader("X-Test-UID"); uid != "" {
			c.Set(middleware.ContextUIDKey, uid)
		}
	}
}

// newRouter wires the real controllers with the same paths as the routes
// package, but with fakeAuth in place of Firebase verification.
func newRouter() *gin.Engine {
	r := gin.New()
	r.Use(fakeAuth())

	r.POST("/auth/register", controllers.RegisterUser)

	r.GET("/warbands", controllers.GetAllWarbands)
	r.GET("/warbands/:id", controllers.GetWarbandByID)
	r.POST("/warbands/create", controllers.CreateWarband)
	r.PATCH("/warbands/:id", controllers.UpdateWarband)
	r.DELETE("/warbands/:id", controllers.DeleteWarband)

	r.GET("/units/:id", controllers.GetUnit)
	r.POST("/units/create", controllers.CreateUnit)
	r.PATCH("/units/:id", controllers.UpdateUnit)
	r.DELETE("/units/:id", controllers.DeleteUnit)
	r.PATCH("/units/:id/kills", controllers.AddUnitKills)
	r.PATCH("/units/:id/xp", controllers.AddUnitXP)
	r.PATCH("/units/:id/perk", controllers.AddUnitPerk)
	r.DELETE("/units/:id/perk/:perkId", controllers.DeleteUnitPerk)

	return r
}

// call sends a request as the given user ("" for unauthenticated) and returns the recorder.
func call(r *gin.Engine, uid, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if uid != "" {
		req.Header.Set("X-Test-UID", uid)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// expect fails the test unless the response has the wanted status code.
func expect(t *testing.T, w *httptest.ResponseRecorder, status int) {
	t.Helper()
	if w.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", w.Code, status, w.Body.String())
	}
}

func decodeWarband(t *testing.T, w *httptest.ResponseRecorder) models.Warband {
	t.Helper()
	var out models.Warband
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode warband: %v; body: %s", err, w.Body.String())
	}
	return out
}

func decodeUnit(t *testing.T, w *httptest.ResponseRecorder) models.Unit {
	t.Helper()
	var out models.Unit
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode unit: %v; body: %s", err, w.Body.String())
	}
	return out
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func countJSONArray(t *testing.T, body []byte) int {
	t.Helper()
	var arr []json.RawMessage
	if err := json.Unmarshal(body, &arr); err != nil {
		t.Fatalf("decode array: %v; body: %s", err, body)
	}
	return len(arr)
}
