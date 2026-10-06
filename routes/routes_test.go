package routes_test

import (
	"testing"

	"github.com/gin-gonic/gin"

	"battlebarge/routes"
)

func TestRoutesRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.GetAuthControllers(r)
	routes.GetUserControllers(r)
	routes.GetWarbandControllers(r)
	routes.GetUnitControllers(r)
	routes.GetCampaignControllers(r)

	want := []string{
		"POST /auth/register",
		"GET /users/me",
		"GET /warbands",
		"GET /warbands/:id",
		"POST /warbands/create",
		"PATCH /warbands/:id",
		"DELETE /warbands/:id",
		"GET /units/:id",
		"POST /units/create",
		"PATCH /units/:id",
		"DELETE /units/:id",
		"PATCH /units/:id/kills",
		"PATCH /units/:id/xp",
		"PATCH /units/:id/perk",
		"DELETE /units/:id/perk/:perkId",
		"GET /campaigns",
		"GET /campaigns/:id",
		"POST /campaigns/create",
		"PATCH /campaigns/:id",
		"DELETE /campaigns/:id",
		"POST /campaigns/:id/chapters",
		"PATCH /campaigns/:id/chapters/:chapterId",
		"DELETE /campaigns/:id/chapters/:chapterId",
		"POST /campaigns/:id/teams",
		"PATCH /campaigns/:id/teams/:teamId",
		"DELETE /campaigns/:id/teams/:teamId",
		"POST /campaigns/:id/warbands",
		"PATCH /campaigns/:id/warbands/:warbandId",
		"DELETE /campaigns/:id/warbands/:warbandId",
	}

	got := map[string]bool{}
	for _, ri := range r.Routes() {
		got[ri.Method+" "+ri.Path] = true
	}

	for _, w := range want {
		if !got[w] {
			t.Errorf("route %q not registered", w)
		}
	}
	if len(got) != len(want) {
		t.Errorf("registered %d routes, tests list %d; update the list if routes changed", len(got), len(want))
	}
}

// Routes that change data must sit behind auth. Hitting them without an
// Authorization header must be rejected before reaching any handler or database.
func TestProtectedRoutesRequireAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.GetUserControllers(r)
	routes.GetWarbandControllers(r)
	routes.GetUnitControllers(r)
	routes.GetCampaignControllers(r)

	protected := []struct{ method, path string }{
		{"GET", "/users/me"},
		{"GET", "/warbands"},
		{"POST", "/warbands/create"},
		{"PATCH", "/warbands/abc"},
		{"DELETE", "/warbands/abc"},
		{"POST", "/units/create"},
		{"PATCH", "/units/abc"},
		{"DELETE", "/units/abc"},
		{"PATCH", "/units/abc/kills"},
		{"PATCH", "/units/abc/xp"},
		{"PATCH", "/units/abc/perk"},
		{"DELETE", "/units/abc/perk/def"},
		{"GET", "/campaigns"},
		{"POST", "/campaigns/create"},
		{"PATCH", "/campaigns/abc"},
		{"DELETE", "/campaigns/abc"},
		{"POST", "/campaigns/abc/chapters"},
		{"PATCH", "/campaigns/abc/chapters/def"},
		{"DELETE", "/campaigns/abc/chapters/def"},
		{"POST", "/campaigns/abc/teams"},
		{"PATCH", "/campaigns/abc/teams/def"},
		{"DELETE", "/campaigns/abc/teams/def"},
		{"POST", "/campaigns/abc/warbands"},
		{"PATCH", "/campaigns/abc/warbands/def"},
		{"DELETE", "/campaigns/abc/warbands/def"},
	}

	for _, p := range protected {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			w := serve(r, p.method, p.path, "")
			if w.Code != 401 {
				t.Errorf("status = %d, want 401", w.Code)
			}
		})
	}
}
