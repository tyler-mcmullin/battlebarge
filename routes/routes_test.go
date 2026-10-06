package routes_test

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"battlebarge/middleware"
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
		"GET /campaigns/:id/join-code",
		"POST /campaigns/:id/join-code/rotate",
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
		{"GET", "/campaigns/abc/join-code"},
		{"POST", "/campaigns/abc/join-code/rotate"},
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

// Registration creates Firebase accounts, so it has its own tight per-IP limit.
// Invalid bodies are used so no Firebase call is made: they are answered 400
// until the limit is hit, then 429.
func TestRegisterIsRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	routes.GetAuthControllers(r)

	const burst = 5
	for i := 1; i <= burst; i++ {
		if w := serve(r, "POST", "/auth/register", `{}`); w.Code != 400 {
			t.Fatalf("request %d: status = %d, want 400", i, w.Code)
		}
	}
	w := serve(r, "POST", "/auth/register", `{}`)
	if w.Code != 429 {
		t.Errorf("request %d: status = %d, want 429", burst+1, w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("429 should include Retry-After")
	}
}

// Joining is the one route where a stranger can guess a secret, so it has its
// own tight limit. A fake verifier lets requests through auth, and invalid
// bodies are answered 400 before any database work.
func TestJoinIsRateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)
	defer middleware.SetTokenVerifier(func(ctx context.Context, idToken string) (*auth.Token, error) {
		return &auth.Token{UID: "u1", Claims: map[string]any{"email_verified": true}}, nil
	})()

	r := gin.New()
	routes.GetCampaignControllers(r)
	path := "/campaigns/" + uuid.NewString() + "/warbands"

	post := func() int {
		req := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
		req.Header.Set("Authorization", "Bearer t")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	const burst = 10
	for i := 1; i <= burst; i++ {
		if code := post(); code != 400 {
			t.Fatalf("attempt %d: status = %d, want 400", i, code)
		}
	}
	if code := post(); code != 429 {
		t.Errorf("attempt %d: status = %d, want 429", burst+1, code)
	}
}
