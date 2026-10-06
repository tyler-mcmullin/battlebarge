package v1_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"battlebarge/testutil"
)

func decodeJoinCode(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	var out struct {
		JoinCode string `json:"join_code"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode join code: %v; body: %s", err, w.Body.String())
	}
	return out.JoinCode
}

func TestJoinCode_OnlyTheOwnerSeesIt(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	testutil.InsertUser(t, "stranger")
	r := newRouter()

	// the owner gets the code when creating the campaign
	w := call(r, "owner", http.MethodPost, "/campaigns/create", `{"name":"Crusade"}`)
	expect(t, w, http.StatusCreated)
	code := decodeCampaign(t, w).JoinCode
	if len(code) != 10 {
		t.Fatalf("create returned join_code %q, want a 10-character code", code)
	}
	path := "/campaigns/" + decodeCampaign(t, w).ID.String()

	// anyone can read the campaign, but the code is never in it
	for _, uid := range []string{"", "owner", "player", "stranger"} {
		w := call(r, uid, http.MethodGet, path, ``)
		expect(t, w, http.StatusOK)
		if strings.Contains(w.Body.String(), "join_code") || strings.Contains(w.Body.String(), code) {
			t.Errorf("public GET as %q exposes the join code: %s", uid, w.Body.String())
		}
	}

	// the owner's list shows it; a participant's list does not
	team := testutil.InsertTeam(t, decodeCampaign(t, w).ID, "Red")
	wb := testutil.InsertWarband(t, "player", "Da Boyz")
	joined := call(r, "player", http.MethodPost, path+"/warbands",
		`{"warband_id":"`+wb.ID.String()+`","team_id":"`+team.ID.String()+`","join_code":"`+code+`"}`)
	expect(t, joined, http.StatusCreated)
	if strings.Contains(joined.Body.String(), "join_code") {
		t.Errorf("the join response must not carry the code: %s", joined.Body.String())
	}

	w = call(r, "owner", http.MethodGet, "/campaigns", ``)
	expect(t, w, http.StatusOK)
	if !strings.Contains(w.Body.String(), `"join_code":"`+code+`"`) {
		t.Errorf("the owner's campaign list should include the code: %s", w.Body.String())
	}
	w = call(r, "player", http.MethodGet, "/campaigns", ``)
	expect(t, w, http.StatusOK)
	if strings.Contains(w.Body.String(), "join_code") || strings.Contains(w.Body.String(), code) {
		t.Errorf("a participant's campaign list exposes the code: %s", w.Body.String())
	}

	// the dedicated endpoint is owner-only
	w = call(r, "owner", http.MethodGet, path+"/join-code", ``)
	expect(t, w, http.StatusOK)
	if got := decodeJoinCode(t, w); got != code {
		t.Errorf("GET join-code = %q, want %q", got, code)
	}
	for _, uid := range []string{"player", "stranger"} {
		expect(t, call(r, uid, http.MethodGet, path+"/join-code", ``), http.StatusNotFound)
		expect(t, call(r, uid, http.MethodPost, path+"/join-code/rotate", ``), http.StatusNotFound)
	}
	expect(t, call(r, "owner", http.MethodGet, "/campaigns/nope/join-code", ``), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodPost, "/campaigns/nope/join-code/rotate", ``), http.StatusNotFound)
	expect(t, call(r, "owner", http.MethodGet, "/campaigns/"+uuid.NewString()+"/join-code", ``), http.StatusNotFound)
}

func TestJoining_NeedsTheCode(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	c := testutil.InsertCampaign(t, "owner", "C") // join code TESTCODE01
	team := testutil.InsertTeam(t, c.ID, "Red")
	wb := testutil.InsertWarband(t, "player", "Da Boyz")
	ownerWb := testutil.InsertWarband(t, "owner", "Mine")
	path := "/campaigns/" + c.ID.String() + "/warbands"
	r := newRouter()

	body := func(wbID, code string) string {
		b := `{"warband_id":"` + wbID + `","team_id":"` + team.ID.String() + `"`
		if code != "" {
			b += `,"join_code":"` + code + `"`
		}
		return b + `}`
	}
	members := func() int {
		return len(decodeCampaign(t, call(r, "", http.MethodGet, "/campaigns/"+c.ID.String(), ``)).Warbands)
	}

	// knowing only the campaign ID is not enough
	w := call(r, "player", http.MethodPost, path, body(wb.ID.String(), ""))
	expect(t, w, http.StatusForbidden)
	if !strings.Contains(w.Body.String(), "join code required") {
		t.Errorf("body = %s", w.Body.String())
	}
	w = call(r, "player", http.MethodPost, path, body(wb.ID.String(), "WRONGCODE1"))
	expect(t, w, http.StatusForbidden)
	if !strings.Contains(w.Body.String(), "invalid join code") {
		t.Errorf("body = %s", w.Body.String())
	}
	// a wrong code gets no hint about anything else, such as which teams exist
	w = call(r, "player", http.MethodPost, path, `{"warband_id":"`+wb.ID.String()+`","team_id":"`+uuid.NewString()+`","join_code":"WRONGCODE1"}`)
	expect(t, w, http.StatusForbidden)
	if members() != 0 {
		t.Fatalf("%d warbands joined without a valid code", members())
	}

	// the right code works, however it is typed
	expect(t, call(r, "player", http.MethodPost, path, body(wb.ID.String(), "test-code01")), http.StatusCreated)
	if members() != 1 {
		t.Errorf("%d members after joining with the code, want 1", members())
	}

	// the campaign owner does not need the code, and a wrong one is ignored
	expect(t, call(r, "owner", http.MethodPost, path, body(ownerWb.ID.String(), "")), http.StatusCreated)
	if members() != 2 {
		t.Errorf("%d members after the owner joined, want 2", members())
	}

	// ownership of the warband is still checked before the code
	other := testutil.InsertWarband(t, "owner", "owner's second")
	expect(t, call(r, "player", http.MethodPost, path, body(other.ID.String(), "TESTCODE01")), http.StatusNotFound)
}

func TestRotatingTheJoinCode(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	c := testutil.InsertCampaign(t, "owner", "C")
	team := testutil.InsertTeam(t, c.ID, "Red")
	base := "/campaigns/" + c.ID.String()
	r := newRouter()

	first := testutil.InsertWarband(t, "player", "first")
	second := testutil.InsertWarband(t, "player", "second")
	join := func(wbID, code string) int {
		return call(r, "player", http.MethodPost, base+"/warbands",
			`{"warband_id":"`+wbID+`","team_id":"`+team.ID.String()+`","join_code":"`+code+`"}`).Code
	}

	if code := join(first.ID.String(), "TESTCODE01"); code != http.StatusCreated {
		t.Fatalf("join with the original code: status = %d, want 201", code)
	}

	w := call(r, "owner", http.MethodPost, base+"/join-code/rotate", ``)
	expect(t, w, http.StatusOK)
	fresh := decodeJoinCode(t, w)
	if fresh == "TESTCODE01" || len(fresh) != 10 {
		t.Fatalf("rotated code = %q, want a different 10-character code", fresh)
	}

	if code := join(second.ID.String(), "TESTCODE01"); code != http.StatusForbidden {
		t.Errorf("old code after rotating: status = %d, want 403", code)
	}
	if code := join(second.ID.String(), fresh); code != http.StatusCreated {
		t.Errorf("new code: status = %d, want 201", code)
	}

	// the owner now sees the new code, and earlier members were not removed
	w = call(r, "owner", http.MethodGet, base+"/join-code", ``)
	expect(t, w, http.StatusOK)
	if got := decodeJoinCode(t, w); got != fresh {
		t.Errorf("GET join-code = %q, want %q", got, fresh)
	}
	if n := len(decodeCampaign(t, call(r, "", http.MethodGet, base, ``)).Warbands); n != 2 {
		t.Errorf("%d members, want 2", n)
	}
}
