package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"battlebarge/models"
	"battlebarge/testutil"
)

func decodeChapter(t *testing.T, w *httptest.ResponseRecorder) models.CampaignChapter {
	t.Helper()
	var out models.CampaignChapter
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode chapter: %v; body: %s", err, w.Body.String())
	}
	return out
}

func decodeTeam(t *testing.T, w *httptest.ResponseRecorder) models.CampaignTeam {
	t.Helper()
	var out models.CampaignTeam
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode team: %v; body: %s", err, w.Body.String())
	}
	return out
}

func TestCampaignEndpoints_Unauthenticated(t *testing.T) {
	r := newRouter()
	id := uuid.NewString()

	// These reject before touching the database, so no DB is needed.
	tests := []struct{ method, path, body string }{
		{"GET", "/campaigns", ``},
		{"POST", "/campaigns/create", `{"name":"x"}`},
		{"PATCH", "/campaigns/" + id, `{"name":"x"}`},
		{"DELETE", "/campaigns/" + id, ``},
		{"POST", "/campaigns/" + id + "/chapters", `{"title":"x"}`},
		{"PATCH", "/campaigns/" + id + "/chapters/" + id, `{"title":"x"}`},
		{"DELETE", "/campaigns/" + id + "/chapters/" + id, ``},
		{"POST", "/campaigns/" + id + "/teams", `{"name":"x"}`},
		{"PATCH", "/campaigns/" + id + "/teams/" + id, `{"name":"x"}`},
		{"DELETE", "/campaigns/" + id + "/teams/" + id, ``},
		{"POST", "/campaigns/" + id + "/warbands", `{"warband_id":"` + id + `","team_id":"` + id + `"}`},
		{"PATCH", "/campaigns/" + id + "/warbands/" + id, `{"team_id":"` + id + `"}`},
		{"DELETE", "/campaigns/" + id + "/warbands/" + id, ``},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			expect(t, call(r, "", tt.method, tt.path, tt.body), http.StatusUnauthorized)
		})
	}
}

func TestCampaignEndpoints_InvalidBody(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	base := "/campaigns/" + c.ID.String()
	id := uuid.NewString()
	r := newRouter()

	tests := []struct{ name, method, path, body string }{
		{"create malformed", "POST", "/campaigns/create", `{oops`},
		{"create missing name", "POST", "/campaigns/create", `{"description":"x"}`},
		{"update malformed", "PATCH", base, `{oops`},
		{"chapter missing title", "POST", base + "/chapters", `{}`},
		{"chapter update malformed", "PATCH", base + "/chapters/" + id, `{oops`},
		{"team missing name", "POST", base + "/teams", `{}`},
		{"team name too long", "POST", base + "/teams", `{"name":"` + strings.Repeat("x", 51) + `"}`},
		{"team rename missing name", "PATCH", base + "/teams/" + id, `{}`},
		{"join missing fields", "POST", base + "/warbands", `{}`},
		{"join malformed warband_id", "POST", base + "/warbands", `{"warband_id":"nope","team_id":"` + id + `"}`},
		{"join malformed team_id", "POST", base + "/warbands", `{"warband_id":"` + id + `","team_id":"nope"}`},
		{"move missing team", "PATCH", base + "/warbands/" + id, `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(r, "owner", tt.method, tt.path, tt.body), http.StatusBadRequest)
		})
	}
}

func TestCampaignLifecycle(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "other")
	r := newRouter()

	// create with only the required field
	w := call(r, "owner", http.MethodPost, "/campaigns/create", `{"name":"Crusade"}`)
	expect(t, w, http.StatusCreated)
	c := decodeCampaign(t, w)
	if c.Name != "Crusade" || c.OwnerID != "owner" || c.Settings != (models.CampaignSettings{}) {
		t.Errorf("unexpected campaign: %+v", c)
	}
	path := "/campaigns/" + c.ID.String()

	// create with settings
	w = call(r, "owner", http.MethodPost, "/campaigns/create",
		`{"name":"Second","description":"d","points_per_win":3,"points_per_loss":1,"starting_requisition":5}`)
	expect(t, w, http.StatusCreated)
	if got := decodeCampaign(t, w); got.Settings.PointsPerWin != 3 || got.Settings.PointsPerLoss != 1 || got.Settings.StartingRequisition != 5 || got.Description != "d" {
		t.Errorf("settings not applied: %+v", got)
	}

	// get is public and includes empty child collections
	w = call(r, "", http.MethodGet, path, ``)
	expect(t, w, http.StatusOK)
	got := decodeCampaign(t, w)
	if got.ID != c.ID || got.Chapters == nil || got.Teams == nil || got.Warbands == nil {
		t.Errorf("fetched campaign wrong: %+v", got)
	}

	// list shows only the caller's campaigns
	call(r, "other", http.MethodPost, "/campaigns/create", `{"name":"Not Mine"}`)
	w = call(r, "owner", http.MethodGet, "/campaigns", ``)
	expect(t, w, http.StatusOK)
	if n := countJSONArray(t, w.Body.Bytes()); n != 2 {
		t.Errorf("owner sees %d campaigns, want 2", n)
	}

	// update
	w = call(r, "owner", http.MethodPatch, path, `{"name":"Renamed","points_per_win":9}`)
	expect(t, w, http.StatusOK)
	if got := decodeCampaign(t, w); got.Name != "Renamed" || got.Settings.PointsPerWin != 9 {
		t.Errorf("update wrong: %+v", got)
	}

	// non-owners get 404 on update and delete
	expect(t, call(r, "other", http.MethodPatch, path, `{"name":"x"}`), http.StatusNotFound)
	expect(t, call(r, "other", http.MethodDelete, path, ``), http.StatusNotFound)

	// delete
	expect(t, call(r, "owner", http.MethodDelete, path, ``), http.StatusNoContent)
	expect(t, call(r, "", http.MethodGet, path, ``), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodDelete, path, ``), http.StatusNotFound)
}

func TestChapterEndpoints(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "intruder")
	c := testutil.InsertCampaign(t, "owner", "C")
	base := "/campaigns/" + c.ID.String() + "/chapters"
	r := newRouter()

	// appended in order when sort_order is omitted
	w := call(r, "owner", http.MethodPost, base, `{"title":"Prologue","description":"intro"}`)
	expect(t, w, http.StatusCreated)
	first := decodeChapter(t, w)
	second := decodeChapter(t, call(r, "owner", http.MethodPost, base, `{"title":"Act I"}`))
	if first.Title != "Prologue" || first.Description != "intro" || first.SortOrder != 1 || second.SortOrder != 2 {
		t.Errorf("unexpected chapters: %+v, %+v", first, second)
	}

	// explicit sort_order, including 0
	w = call(r, "owner", http.MethodPost, base, `{"title":"Zero","sort_order":0}`)
	expect(t, w, http.StatusCreated)
	if got := decodeChapter(t, w); got.SortOrder != 0 {
		t.Errorf("sort_order = %d, want explicit 0", got.SortOrder)
	}

	// reorder and edit
	w = call(r, "owner", http.MethodPatch, base+"/"+first.ID.String(), `{"title":"Opening","sort_order":5}`)
	expect(t, w, http.StatusOK)
	if got := decodeChapter(t, w); got.Title != "Opening" || got.SortOrder != 5 || got.Description != "intro" {
		t.Errorf("update wrong: %+v", got)
	}

	// visible on the campaign, ordered by sort_order
	w = call(r, "", http.MethodGet, "/campaigns/"+c.ID.String(), ``)
	expect(t, w, http.StatusOK)
	chapters := decodeCampaign(t, w).Chapters
	if len(chapters) != 3 || chapters[0].Title != "Zero" || chapters[2].Title != "Opening" {
		t.Errorf("chapters on campaign = %+v", chapters)
	}

	// non-owner can't touch them
	expect(t, call(r, "intruder", http.MethodPost, base, `{"title":"x"}`), http.StatusNotFound)
	expect(t, call(r, "intruder", http.MethodPatch, base+"/"+first.ID.String(), `{"title":"x"}`), http.StatusNotFound)
	expect(t, call(r, "intruder", http.MethodDelete, base+"/"+first.ID.String(), ``), http.StatusNotFound)

	// unknown / malformed chapter ids
	expect(t, call(r, "owner", http.MethodPatch, base+"/"+uuid.NewString(), `{"title":"x"}`), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodPatch, base+"/nope", `{"title":"x"}`), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodDelete, base+"/nope", ``), http.StatusNotFound)

	// delete
	expect(t, call(r, "owner", http.MethodDelete, base+"/"+first.ID.String(), ``), http.StatusNoContent)
	expect(t, call(r, "owner", http.MethodDelete, base+"/"+first.ID.String(), ``), http.StatusNotFound)
}

func TestTeamEndpoints(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "intruder")
	testutil.InsertUser(t, "player")
	c := testutil.InsertCampaign(t, "owner", "C")
	base := "/campaigns/" + c.ID.String() + "/teams"
	r := newRouter()

	// any number of teams
	var teams []models.CampaignTeam
	for _, n := range []string{"Red", "Blue", "Green", "Yellow", "Purple"} {
		w := call(r, "owner", http.MethodPost, base, `{"name":"`+n+`"}`)
		expect(t, w, http.StatusCreated)
		teams = append(teams, decodeTeam(t, w))
	}
	if teams[0].Name != "Red" || teams[0].CampaignID != c.ID {
		t.Errorf("unexpected team: %+v", teams[0])
	}

	// duplicate name conflicts
	expect(t, call(r, "owner", http.MethodPost, base, `{"name":"Red"}`), http.StatusConflict)

	// rename
	w := call(r, "owner", http.MethodPatch, base+"/"+teams[0].ID.String(), `{"name":"Crimson"}`)
	expect(t, w, http.StatusOK)
	if got := decodeTeam(t, w); got.Name != "Crimson" || got.ID != teams[0].ID {
		t.Errorf("rename wrong: %+v", got)
	}
	expect(t, call(r, "owner", http.MethodPatch, base+"/"+teams[1].ID.String(), `{"name":"Crimson"}`), http.StatusConflict)

	// all teams show on the campaign
	w = call(r, "", http.MethodGet, "/campaigns/"+c.ID.String(), ``)
	if n := len(decodeCampaign(t, w).Teams); n != 5 {
		t.Errorf("campaign has %d teams, want 5", n)
	}

	// non-owner can't manage teams, not even a participant
	expect(t, call(r, "intruder", http.MethodPost, base, `{"name":"Sneaky"}`), http.StatusNotFound)
	expect(t, call(r, "player", http.MethodPatch, base+"/"+teams[0].ID.String(), `{"name":"x"}`), http.StatusNotFound)
	expect(t, call(r, "intruder", http.MethodDelete, base+"/"+teams[0].ID.String(), ``), http.StatusNotFound)

	// unknown / malformed team ids
	expect(t, call(r, "owner", http.MethodPatch, base+"/"+uuid.NewString(), `{"name":"x"}`), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodPatch, base+"/nope", `{"name":"x"}`), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodDelete, base+"/nope", ``), http.StatusNotFound)

	// a team with a warband on it can't be deleted
	wb := testutil.InsertWarband(t, "player", "W")
	expect(t, call(r, "player", http.MethodPost, "/campaigns/"+c.ID.String()+"/warbands",
		`{"warband_id":"`+wb.ID.String()+`","team_id":"`+teams[2].ID.String()+`","join_code":"TESTCODE01"}`), http.StatusCreated)
	expect(t, call(r, "owner", http.MethodDelete, base+"/"+teams[2].ID.String(), ``), http.StatusConflict)

	// empty teams delete fine
	expect(t, call(r, "owner", http.MethodDelete, base+"/"+teams[3].ID.String(), ``), http.StatusNoContent)
	expect(t, call(r, "owner", http.MethodDelete, base+"/"+teams[3].ID.String(), ``), http.StatusNotFound)
}

func TestWarbandsJoinTeams(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	testutil.InsertUser(t, "other")
	c := testutil.InsertCampaign(t, "owner", "C")
	red := testutil.InsertTeam(t, c.ID, "Red")
	blue := testutil.InsertTeam(t, c.ID, "Blue")
	elsewhere := testutil.InsertCampaign(t, "owner", "Elsewhere")
	foreign := testutil.InsertTeam(t, elsewhere.ID, "Foreign")
	wb := testutil.InsertWarband(t, "player", "Da Boyz")
	theirs := testutil.InsertWarband(t, "other", "Theirs")
	base := "/campaigns/" + c.ID.String() + "/warbands"
	r := newRouter()

	join := func(uid string, warband, team uuid.UUID) *httptest.ResponseRecorder {
		return call(r, uid, http.MethodPost, base, `{"warband_id":"`+warband.String()+`","team_id":"`+team.String()+`","join_code":"TESTCODE01"}`)
	}

	// can't join with someone else's warband (404 hides it), or an unknown one
	expect(t, join("player", theirs.ID, red.ID), http.StatusNotFound)
	expect(t, join("player", uuid.New(), red.ID), http.StatusNotFound)
	// can't join a team from another campaign, or an unknown one
	expect(t, join("player", wb.ID, foreign.ID), http.StatusNotFound)
	expect(t, join("player", wb.ID, uuid.New()), http.StatusNotFound)
	// unknown campaign
	expect(t, call(r, "player", http.MethodPost, "/campaigns/"+uuid.NewString()+"/warbands",
		`{"warband_id":"`+wb.ID.String()+`","team_id":"`+red.ID.String()+`","join_code":"TESTCODE01"}`), http.StatusNotFound)

	// the warband's owner joins, no campaign ownership needed
	w := join("player", wb.ID, red.ID)
	expect(t, w, http.StatusCreated)
	members := decodeCampaign(t, w).Warbands
	if len(members) != 1 || members[0].WarbandID != wb.ID || members[0].TeamID != red.ID || members[0].WarbandName != "Da Boyz" {
		t.Errorf("unexpected members: %+v", members)
	}

	// only once per campaign
	expect(t, join("player", wb.ID, blue.ID), http.StatusConflict)

	// the same warband can be in another campaign
	expect(t, call(r, "player", http.MethodPost, "/campaigns/"+elsewhere.ID.String()+"/warbands",
		`{"warband_id":"`+wb.ID.String()+`","team_id":"`+foreign.ID.String()+`","join_code":"TESTCODE01"}`), http.StatusCreated)

	// moving teams: the warband owner and the campaign owner can; others can't
	move := func(uid string, team uuid.UUID) *httptest.ResponseRecorder {
		return call(r, uid, http.MethodPatch, base+"/"+wb.ID.String(), `{"team_id":"`+team.String()+`"}`)
	}
	w = move("player", blue.ID)
	expect(t, w, http.StatusOK)
	if got := decodeCampaign(t, w).Warbands; len(got) != 1 || got[0].TeamID != blue.ID {
		t.Errorf("warband not moved: %+v", got)
	}
	expect(t, move("owner", red.ID), http.StatusOK)
	expect(t, move("other", blue.ID), http.StatusNotFound)
	expect(t, move("player", foreign.ID), http.StatusNotFound)                                                                           // team from another campaign
	expect(t, call(r, "owner", http.MethodPatch, base+"/"+theirs.ID.String(), `{"team_id":"`+red.ID.String()+`"}`), http.StatusNotFound) // not a member
	expect(t, call(r, "owner", http.MethodPatch, base+"/nope", `{"team_id":"`+red.ID.String()+`"}`), http.StatusNotFound)

	// leaving: strangers can't remove it; the warband owner and campaign owner can
	expect(t, call(r, "other", http.MethodDelete, base+"/"+wb.ID.String(), ``), http.StatusNotFound)
	expect(t, call(r, "player", http.MethodDelete, base+"/"+wb.ID.String(), ``), http.StatusNoContent)
	expect(t, call(r, "player", http.MethodDelete, base+"/"+wb.ID.String(), ``), http.StatusNotFound)

	expect(t, join("player", wb.ID, red.ID), http.StatusCreated)
	expect(t, call(r, "owner", http.MethodDelete, base+"/"+wb.ID.String(), ``), http.StatusNoContent)
	expect(t, call(r, "owner", http.MethodDelete, base+"/nope", ``), http.StatusNotFound)

	// participants see the campaign in their list
	expect(t, join("player", wb.ID, red.ID), http.StatusCreated)
	w = call(r, "player", http.MethodGet, "/campaigns", ``)
	expect(t, w, http.StatusOK)
	if n := countJSONArray(t, w.Body.Bytes()); n != 2 {
		t.Errorf("player sees %d campaigns, want 2 (both joined)", n)
	}
}

func TestCampaignMalformedIDs(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	r := newRouter()

	for _, tt := range []struct{ method, path, body string }{
		{"GET", "/campaigns/nope", ``},
		{"PATCH", "/campaigns/nope", `{"name":"x"}`},
		{"DELETE", "/campaigns/nope", ``},
		{"POST", "/campaigns/nope/chapters", `{"title":"x"}`},
		{"POST", "/campaigns/nope/teams", `{"name":"x"}`},
		{"POST", "/campaigns/nope/warbands", `{"warband_id":"` + uuid.NewString() + `","team_id":"` + uuid.NewString() + `"}`},
		{"PATCH", "/campaigns/nope/warbands/" + uuid.NewString(), `{"team_id":"` + uuid.NewString() + `"}`},
		{"DELETE", "/campaigns/nope/warbands/" + uuid.NewString(), ``},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := call(r, "owner", tt.method, tt.path, tt.body)
			expect(t, w, http.StatusNotFound)
			if strings.Contains(w.Body.String(), "SQLSTATE") {
				t.Errorf("response leaks database error: %s", w.Body.String())
			}
		})
	}
}
