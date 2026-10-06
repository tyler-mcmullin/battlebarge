package v1_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	"battlebarge/testutil"
)

func TestUnitEndpoints_Unauthenticated(t *testing.T) {
	r := newRouter()
	id := uuid.NewString()

	// These reject before touching the database, so no DB is needed.
	tests := []struct{ method, path, body string }{
		{"POST", "/units/create", `{"warband_id":"` + id + `","unit_name":"x"}`},
		{"PATCH", "/units/" + id, `{"unit_name":"x"}`},
		{"DELETE", "/units/" + id, ``},
		{"PATCH", "/units/" + id + "/kills", `{"amount":1}`},
		{"PATCH", "/units/" + id + "/xp", `{"amount":1}`},
		{"PATCH", "/units/" + id + "/perk", `{"name":"x"}`},
		{"DELETE", "/units/" + id + "/perk/" + id, ``},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			expect(t, call(r, "", tt.method, tt.path, tt.body), http.StatusUnauthorized)
		})
	}
}

func TestUnitEndpoints_InvalidBody(t *testing.T) {
	r := newRouter()
	id := uuid.NewString()

	tests := []struct{ name, method, path, body string }{
		{"create malformed", "POST", "/units/create", `{oops`},
		{"create missing unit_name", "POST", "/units/create", `{"warband_id":"` + id + `"}`},
		{"create missing warband_id", "POST", "/units/create", `{"unit_name":"x"}`},
		{"update malformed", "PATCH", "/units/" + id, `{oops`},
		{"kills missing amount", "PATCH", "/units/" + id + "/kills", `{}`},
		{"xp missing amount", "PATCH", "/units/" + id + "/xp", `{}`},
		{"perk missing name", "PATCH", "/units/" + id + "/perk", `{"is_scar":true}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(r, "owner", tt.method, tt.path, tt.body), http.StatusBadRequest)
		})
	}
}

func TestUnitLifecycle(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "other")
	wb := testutil.InsertWarband(t, "owner", "W")
	r := newRouter()

	// create with only required fields
	w := call(r, "owner", http.MethodPost, "/units/create", `{"warband_id":"`+wb.ID.String()+`","unit_name":"Boss"}`)
	expect(t, w, http.StatusCreated)
	unit := decodeUnit(t, w)
	if unit.UnitName != "Boss" || unit.WarbandID != wb.ID || unit.Points != 0 || unit.Kills != 0 || len(unit.Perks) != 0 {
		t.Errorf("unexpected created unit: %+v", unit)
	}
	path := "/units/" + unit.ID.String()

	// get is public
	w = call(r, "", http.MethodGet, path, ``)
	expect(t, w, http.StatusOK)
	if got := decodeUnit(t, w); got.ID != unit.ID {
		t.Errorf("fetched wrong unit: %+v", got)
	}

	// update
	w = call(r, "owner", http.MethodPatch, path, `{"unit_name":"Warboss","points":150}`)
	expect(t, w, http.StatusOK)
	if got := decodeUnit(t, w); got.UnitName != "Warboss" || got.Points != 150 {
		t.Errorf("update wrong: %+v", got)
	}

	// kills and xp, including the clamp at zero
	w = call(r, "owner", http.MethodPatch, path+"/kills", `{"amount":4}`)
	expect(t, w, http.StatusOK)
	if got := decodeUnit(t, w); got.Kills != 4 {
		t.Errorf("kills = %d, want 4", got.Kills)
	}
	w = call(r, "owner", http.MethodPatch, path+"/kills", `{"amount":-10}`)
	expect(t, w, http.StatusOK)
	if got := decodeUnit(t, w); got.Kills != 0 {
		t.Errorf("kills = %d, want 0 after clamp", got.Kills)
	}
	w = call(r, "owner", http.MethodPatch, path+"/xp", `{"amount":6}`)
	expect(t, w, http.StatusOK)
	if got := decodeUnit(t, w); got.Experience != 6 {
		t.Errorf("xp = %d, want 6", got.Experience)
	}

	// perks: the server generates the ID, ignoring any client-supplied one
	clientID := uuid.NewString()
	w = call(r, "owner", http.MethodPatch, path+"/perk", `{"perk_id":"`+clientID+`","name":"Tough","description":"hard","is_scar":false}`)
	expect(t, w, http.StatusOK)
	got := decodeUnit(t, w)
	if len(got.Perks) != 1 || got.Perks[0].Name != "Tough" || got.Perks[0].Description != "hard" {
		t.Fatalf("perk not added: %+v", got.Perks)
	}
	if got.Perks[0].ID.String() == clientID {
		t.Error("perk ID should be generated server-side, not taken from the request")
	}
	perkPath := path + "/perk/" + got.Perks[0].ID.String()

	// deleting an unknown perk is a 404
	expect(t, call(r, "owner", http.MethodDelete, path+"/perk/"+uuid.NewString(), ``), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodDelete, path+"/perk/not-a-uuid", ``), http.StatusNotFound)

	w = call(r, "owner", http.MethodDelete, perkPath, ``)
	expect(t, w, http.StatusOK)
	if got := decodeUnit(t, w); len(got.Perks) != 0 {
		t.Errorf("perk not deleted: %+v", got.Perks)
	}

	// delete the unit
	expect(t, call(r, "owner", http.MethodDelete, path, ``), http.StatusNoContent)
	expect(t, call(r, "", http.MethodGet, path, ``), http.StatusNotFound)
}

func TestUnitOwnership(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "intruder")
	wb := testutil.InsertWarband(t, "owner", "W")
	unit := testutil.InsertUnit(t, wb.ID, "Boss", 10)
	path := "/units/" + unit.ID.String()
	r := newRouter()

	// A user who doesn't own the warband can't create units in it...
	w := call(r, "intruder", http.MethodPost, "/units/create", `{"warband_id":"`+wb.ID.String()+`","unit_name":"Sneaky"}`)
	expect(t, w, http.StatusNotFound)

	// ...or modify existing ones. All return 404, hiding that the unit exists.
	tests := []struct{ name, method, path, body string }{
		{"update", "PATCH", path, `{"unit_name":"Hacked"}`},
		{"delete", "DELETE", path, ``},
		{"kills", "PATCH", path + "/kills", `{"amount":5}`},
		{"xp", "PATCH", path + "/xp", `{"amount":5}`},
		{"add perk", "PATCH", path + "/perk", `{"name":"x"}`},
		{"delete perk", "DELETE", path + "/perk/" + uuid.NewString(), ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expect(t, call(r, "intruder", tt.method, tt.path, tt.body), http.StatusNotFound)
		})
	}

	// nothing changed
	w = call(r, "", http.MethodGet, path, ``)
	expect(t, w, http.StatusOK)
	if got := decodeUnit(t, w); got.UnitName != "Boss" || got.Kills != 0 || got.Experience != 0 || len(got.Perks) != 0 {
		t.Errorf("unit was modified by non-owner: %+v", got)
	}
}

func TestUnitEndpoints_UnknownUnit(t *testing.T) {
	testutil.SetupDB(t)
	r := newRouter()
	path := "/units/" + uuid.NewString()

	expect(t, call(r, "", http.MethodGet, path, ``), http.StatusNotFound)
	for _, tt := range []struct{ method, suffix, body string }{
		{"PATCH", "", `{"unit_name":"x"}`},
		{"DELETE", "", ``},
		{"PATCH", "/kills", `{"amount":1}`},
		{"PATCH", "/xp", `{"amount":1}`},
		{"PATCH", "/perk", `{"name":"x"}`},
		{"DELETE", "/perk/" + uuid.NewString(), ``},
	} {
		t.Run(tt.method+" "+tt.suffix, func(t *testing.T) {
			expect(t, call(r, "owner", tt.method, path+tt.suffix, tt.body), http.StatusNotFound)
		})
	}
}

// A malformed (non-UUID) ID can never match a record, so path IDs give 404
// and a malformed warband_id in a body gives 400, never a 500 or raw DB error.
func TestMalformedIDs(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	r := newRouter()

	for _, tt := range []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/units/nope", ``, http.StatusNotFound},
		{"PATCH", "/units/nope", `{"unit_name":"x"}`, http.StatusNotFound},
		{"DELETE", "/units/nope", ``, http.StatusNotFound},
		{"PATCH", "/units/nope/kills", `{"amount":1}`, http.StatusNotFound},
		{"PATCH", "/units/nope/xp", `{"amount":1}`, http.StatusNotFound},
		{"PATCH", "/units/nope/perk", `{"name":"x"}`, http.StatusNotFound},
		{"DELETE", "/units/nope/perk/" + uuid.NewString(), ``, http.StatusNotFound},
		{"GET", "/warbands/nope", ``, http.StatusNotFound},
		{"PATCH", "/warbands/nope", `{"name":"x"}`, http.StatusNotFound},
		{"DELETE", "/warbands/nope", ``, http.StatusNotFound},
		{"POST", "/units/create", `{"warband_id":"nope","unit_name":"x"}`, http.StatusBadRequest},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			w := call(r, "owner", tt.method, tt.path, tt.body)
			expect(t, w, tt.want)
			if strings.Contains(w.Body.String(), "SQLSTATE") {
				t.Errorf("response leaks database error: %s", w.Body.String())
			}
		})
	}
}
