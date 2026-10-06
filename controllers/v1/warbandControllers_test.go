package v1_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"

	"battlebarge/testutil"
)

func TestWarbandEndpoints_Unauthenticated(t *testing.T) {
	r := newRouter()
	id := uuid.NewString()

	// These reject before touching the database, so no DB is needed.
	tests := []struct{ method, path, body string }{
		{"POST", "/warbands/create", `{"name":"x"}`},
		{"PATCH", "/warbands/" + id, `{"name":"x"}`},
		{"DELETE", "/warbands/" + id, ``},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			expect(t, call(r, "", tt.method, tt.path, tt.body), http.StatusUnauthorized)
		})
	}
}

func TestCreateWarband_InvalidBody(t *testing.T) {
	r := newRouter()

	for name, body := range map[string]string{
		"malformed json": `{oops`,
		"missing name":   `{"faction":"Orks"}`,
	} {
		t.Run(name, func(t *testing.T) {
			expect(t, call(r, "owner", http.MethodPost, "/warbands/create", body), http.StatusBadRequest)
		})
	}
}

func TestUpdateWarband_InvalidBody(t *testing.T) {
	r := newRouter()
	expect(t, call(r, "owner", http.MethodPatch, "/warbands/"+uuid.NewString(), `{oops`), http.StatusBadRequest)
}

func TestWarbandLifecycle(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	r := newRouter()

	// create, with only the required field: optional fields default
	w := call(r, "owner", http.MethodPost, "/warbands/create", `{"name":"Da Boyz"}`)
	expect(t, w, http.StatusCreated)
	created := decodeWarband(t, w)
	if created.Name != "Da Boyz" || created.UserID != "owner" || created.Faction != "" || created.SupplyLimit != 0 {
		t.Errorf("unexpected created warband: %+v", created)
	}
	path := "/warbands/" + created.ID.String()

	// create with optional fields
	w = call(r, "owner", http.MethodPost, "/warbands/create",
		`{"name":"Second","faction":"Orks","description":"d","requisition_points":3,"supply_limit":500}`)
	expect(t, w, http.StatusCreated)
	if got := decodeWarband(t, w); got.Faction != "Orks" || got.RequisitionPoints != 3 || got.SupplyLimit != 500 {
		t.Errorf("optional fields not applied: %+v", got)
	}

	// get is public: no auth needed
	w = call(r, "", http.MethodGet, path, ``)
	expect(t, w, http.StatusOK)
	if got := decodeWarband(t, w); got.ID != created.ID {
		t.Errorf("fetched wrong warband: %+v", got)
	}

	// list returns only the caller's warbands
	testutil.InsertUser(t, "other")
	call(r, "other", http.MethodPost, "/warbands/create", `{"name":"Not Mine"}`)
	w = call(r, "owner", http.MethodGet, "/warbands", ``)
	expect(t, w, http.StatusOK)
	if n := countJSONArray(t, w.Body.Bytes()); n != 2 {
		t.Errorf("owner sees %d warbands, want 2", n)
	}

	// update
	w = call(r, "owner", http.MethodPatch, path, `{"faction":"Goffs"}`)
	expect(t, w, http.StatusOK)
	if got := decodeWarband(t, w); got.Faction != "Goffs" || got.Name != "Da Boyz" {
		t.Errorf("update wrong: %+v", got)
	}

	// another user cannot update or delete it, and gets 404 rather than 403
	expect(t, call(r, "other", http.MethodPatch, path, `{"faction":"x"}`), http.StatusNotFound)
	expect(t, call(r, "other", http.MethodDelete, path, ``), http.StatusNotFound)

	// delete
	expect(t, call(r, "owner", http.MethodDelete, path, ``), http.StatusNoContent)
	expect(t, call(r, "", http.MethodGet, path, ``), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodDelete, path, ``), http.StatusNotFound)
}

func TestGetWarband_NotFound(t *testing.T) {
	testutil.SetupDB(t)
	r := newRouter()

	expect(t, call(r, "", http.MethodGet, "/warbands/"+uuid.NewString(), ``), http.StatusNotFound)
}
