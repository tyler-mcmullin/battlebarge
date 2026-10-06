package v1_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"battlebarge/repositories"
	"battlebarge/testutil"
)

func repeat(s string, n int) string { return strings.Repeat(s, n) }

// Bodies that must be rejected with 400 before any database work. The routes
// here validate the body before touching the database, so no DB is needed.
func TestValidation_Rejected(t *testing.T) {
	r := newRouter()
	id := uuid.NewString()
	w101 := repeat("n", 101)

	tests := []struct{ name, method, path, body string }{
		// register
		{"register: username blank", "POST", "/auth/register", `{"email":"a@example.com","username":"   ","password":"secret123"}`},
		{"register: username with control char", "POST", "/auth/register", `{"email":"a@example.com","username":"a\u0001b","password":"secret123"}`},
		{"register: password over 128", "POST", "/auth/register", `{"email":"a@example.com","username":"alice","password":"` + repeat("p", 129) + `"}`},

		// warbands
		{"warband: blank name", "POST", "/warbands/create", `{"name":"   "}`},
		{"warband: name over 100", "POST", "/warbands/create", `{"name":"` + w101 + `"}`},
		{"warband: NUL in name", "POST", "/warbands/create", `{"name":"a\u0000b"}`},
		{"warband: control char in name", "POST", "/warbands/create", `{"name":"a\u0007b"}`},
		{"warband: description over 2000", "POST", "/warbands/create", `{"name":"x","description":"` + repeat("d", 2001) + `"}`},
		{"warband: faction over 100", "POST", "/warbands/create", `{"name":"x","faction":"` + w101 + `"}`},
		{"warband: negative requisition", "POST", "/warbands/create", `{"name":"x","requisition_points":-1}`},
		{"warband: requisition over max", "POST", "/warbands/create", `{"name":"x","requisition_points":1000001}`},
		{"warband: negative supply limit", "POST", "/warbands/create", `{"name":"x","supply_limit":-1}`},
		{"warband update: empty name", "PATCH", "/warbands/" + id, `{"name":""}`},
		{"warband update: blank name", "PATCH", "/warbands/" + id, `{"name":"  "}`},
		{"warband update: negative requisition", "PATCH", "/warbands/" + id, `{"requisition_points":-9}`},

		// units
		{"unit: blank name", "POST", "/units/create", `{"warband_id":"` + id + `","unit_name":" "}`},
		{"unit: name over 100", "POST", "/units/create", `{"warband_id":"` + id + `","unit_name":"` + w101 + `"}`},
		{"unit: bio over 5000", "POST", "/units/create", `{"warband_id":"` + id + `","unit_name":"x","bio":"` + repeat("b", 5001) + `"}`},
		{"unit: negative points", "POST", "/units/create", `{"warband_id":"` + id + `","unit_name":"x","points":-5}`},
		{"unit: points over max", "POST", "/units/create", `{"warband_id":"` + id + `","unit_name":"x","points":1000001}`},
		{"unit: warband_id over 36 chars", "POST", "/units/create", `{"warband_id":"` + repeat("a", 37) + `","unit_name":"x"}`},
		{"unit update: negative points", "PATCH", "/units/" + id, `{"points":-500}`},
		{"unit update: empty name", "PATCH", "/units/" + id, `{"unit_name":""}`},
		{"kills: amount missing", "PATCH", "/units/" + id + "/kills", `{}`},
		{"kills: amount null", "PATCH", "/units/" + id + "/kills", `{"amount":null}`},
		{"kills: above int32", "PATCH", "/units/" + id + "/kills", `{"amount":3000000000}`},
		{"kills: above max", "PATCH", "/units/" + id + "/kills", `{"amount":1000001}`},
		{"xp: below min", "PATCH", "/units/" + id + "/xp", `{"amount":-1000001}`},

		// perks
		{"perk: blank name", "PATCH", "/units/" + id + "/perk", `{"name":"  "}`},
		{"perk: name over 100", "PATCH", "/units/" + id + "/perk", `{"name":"` + w101 + `"}`},
		{"perk: description over 1000", "PATCH", "/units/" + id + "/perk", `{"name":"x","description":"` + repeat("d", 1001) + `"}`},

		// campaigns
		{"campaign: blank name", "POST", "/campaigns/create", `{"name":"   "}`},
		{"campaign: name over 100", "POST", "/campaigns/create", `{"name":"` + w101 + `"}`},
		{"campaign: negative points_per_win", "POST", "/campaigns/create", `{"name":"x","points_per_win":-1}`},
		{"campaign: starting_requisition over max", "POST", "/campaigns/create", `{"name":"x","starting_requisition":1000001}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(r, "owner", tt.method, tt.path, tt.body), http.StatusBadRequest)
		})
	}
}

// Routes that check campaign ownership before reading the body need a database.
func TestValidation_RejectedCampaignBodies(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	base := "/campaigns/" + c.ID.String()
	id := uuid.NewString()
	r := newRouter()

	tests := []struct{ name, method, path, body string }{
		{"update: blank name", "PATCH", base, `{"name":" "}`},
		{"update: negative points_per_loss", "PATCH", base, `{"points_per_loss":-1}`},
		{"chapter: blank title", "POST", base + "/chapters", `{"title":"  "}`},
		{"chapter: title over 100", "POST", base + "/chapters", `{"title":"` + repeat("t", 101) + `"}`},
		{"chapter: negative sort_order", "POST", base + "/chapters", `{"title":"x","sort_order":-1}`},
		{"chapter: description over 2000", "POST", base + "/chapters", `{"title":"x","description":"` + repeat("d", 2001) + `"}`},
		{"chapter update: empty title", "PATCH", base + "/chapters/" + id, `{"title":""}`},
		{"team: blank name", "POST", base + "/teams", `{"name":"  "}`},
		{"team: NUL in name", "POST", base + "/teams", `{"name":"a\u0000b"}`},
		{"team rename: blank name", "PATCH", base + "/teams/" + id, `{"name":" "}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(r, "owner", tt.method, tt.path, tt.body), http.StatusBadRequest)
		})
	}
}

// Values at the limits are accepted, and rejected bodies don't change anything.
func TestValidation_Accepted(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	r := newRouter()

	w := call(r, "owner", http.MethodPost, "/warbands/create",
		`{"name":"`+repeat("n", 100)+`","description":"`+repeat("d", 2000)+`","requisition_points":1000000,"supply_limit":0}`)
	expect(t, w, http.StatusCreated)
	wb := decodeWarband(t, w)

	// tabs, newlines and non-ASCII text are fine
	expect(t, call(r, "owner", http.MethodPatch, "/warbands/"+wb.ID.String(), `{"description":"line one\n\tline two — héros 戦"}`), http.StatusOK)
	// a description can be cleared with an empty string
	w = call(r, "owner", http.MethodPatch, "/warbands/"+wb.ID.String(), `{"description":""}`)
	expect(t, w, http.StatusOK)
	if got := decodeWarband(t, w); got.Description != "" {
		t.Errorf("description = %q, want cleared", got.Description)
	}

	u := decodeUnit(t, call(r, "owner", http.MethodPost, "/units/create",
		`{"warband_id":"`+wb.ID.String()+`","unit_name":"x","points":1000000,"bio":"`+repeat("b", 5000)+`"}`))
	for _, amount := range []string{"1000000", "-1000000"} {
		expect(t, call(r, "owner", http.MethodPatch, "/units/"+u.ID.String()+"/kills", `{"amount":`+amount+`}`), http.StatusOK)
	}
}

// 0 is a valid amount for kills and XP: it succeeds and changes nothing.
func TestIncrement_ZeroIsAllowed(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	wb := testutil.InsertWarband(t, "owner", "W")
	unit := testutil.InsertUnit(t, wb.ID, "Boss", 10)
	r := newRouter()
	path := "/units/" + unit.ID.String()

	// a brand new unit starts at 0 kills and 0 XP, and adding 0 keeps it there
	for _, suffix := range []string{"/kills", "/xp"} {
		w := call(r, "owner", http.MethodPatch, path+suffix, `{"amount":0}`)
		expect(t, w, http.StatusOK)
		if got := decodeUnit(t, w); got.Kills != 0 || got.Experience != 0 {
			t.Errorf("%s with 0: kills = %d, xp = %d; want both 0", suffix, got.Kills, got.Experience)
		}
	}

	// after real gains, adding 0 leaves them as they were
	expect(t, call(r, "owner", http.MethodPatch, path+"/kills", `{"amount":4}`), http.StatusOK)
	expect(t, call(r, "owner", http.MethodPatch, path+"/xp", `{"amount":7}`), http.StatusOK)
	for _, suffix := range []string{"/kills", "/xp"} {
		w := call(r, "owner", http.MethodPatch, path+suffix, `{"amount":0}`)
		expect(t, w, http.StatusOK)
		if got := decodeUnit(t, w); got.Kills != 4 || got.Experience != 7 {
			t.Errorf("%s with 0: kills = %d, xp = %d; want 4 and 7", suffix, got.Kills, got.Experience)
		}
	}
}

// A user can only have a fixed number of each kind of record; going over is a 409.
func TestQuotas_Return409(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	r := newRouter()

	// warbands per user
	var wb uuid.UUID
	for i := 0; i < repositories.MaxWarbandsPerUser; i++ {
		wb = testutil.InsertWarband(t, "owner", "w").ID
	}
	w := call(r, "owner", http.MethodPost, "/warbands/create", `{"name":"one too many"}`)
	expect(t, w, http.StatusConflict)
	if !strings.Contains(w.Body.String(), "limit reached") {
		t.Errorf("body = %s, want a limit message", w.Body.String())
	}

	// units per warband
	for i := 0; i < repositories.MaxUnitsPerWarband; i++ {
		testutil.InsertUnit(t, wb, "u", 1)
	}
	expect(t, call(r, "owner", http.MethodPost, "/units/create", `{"warband_id":"`+wb.String()+`","unit_name":"too many"}`), http.StatusConflict)

	// perks per unit
	unit := testutil.InsertUnit(t, uuid.MustParse(testutil.InsertWarband(t, "owner", "p").ID.String()), "perky", 1)
	for i := 0; i < repositories.MaxPerksPerUnit; i++ {
		expect(t, call(r, "owner", http.MethodPatch, "/units/"+unit.ID.String()+"/perk", `{"name":"p"}`), http.StatusOK)
	}
	expect(t, call(r, "owner", http.MethodPatch, "/units/"+unit.ID.String()+"/perk", `{"name":"p"}`), http.StatusConflict)

	// teams per campaign
	c := testutil.InsertCampaign(t, "owner", "C")
	for i := 0; i < repositories.MaxTeamsPerCampaign; i++ {
		expect(t, call(r, "owner", http.MethodPost, "/campaigns/"+c.ID.String()+"/teams", `{"name":"`+uuid.NewString()[:8]+`"}`), http.StatusCreated)
	}
	expect(t, call(r, "owner", http.MethodPost, "/campaigns/"+c.ID.String()+"/teams", `{"name":"one more"}`), http.StatusConflict)

	// campaigns per user
	for i := 1; i < repositories.MaxCampaignsPerUser; i++ { // one already exists
		testutil.InsertCampaign(t, "owner", "c")
	}
	expect(t, call(r, "owner", http.MethodPost, "/campaigns/create", `{"name":"over"}`), http.StatusConflict)
}
