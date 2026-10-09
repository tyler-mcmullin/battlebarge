package repositories_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"battlebarge/models"
	"battlebarge/repositories"
	"battlebarge/testutil"
)

func TestNewJoinCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		code, err := repositories.NewJoinCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 10 {
			t.Fatalf("code %q has length %d, want 10", code, len(code))
		}
		if strings.ContainsAny(code, "0O1IL") {
			t.Fatalf("code %q contains a look-alike character", code)
		}
		if code != strings.ToUpper(code) {
			t.Fatalf("code %q is not uppercase", code)
		}
		seen[code] = true
	}
	if len(seen) != 500 {
		t.Errorf("%d unique codes out of 500; codes should not repeat", len(seen))
	}
}

func TestNormalizeJoinCode(t *testing.T) {
	for in, want := range map[string]string{
		"ABCDE12345":     "ABCDE12345",
		"abcde12345":     "ABCDE12345",
		" abcde-12345 ":  "ABCDE12345",
		"AB CD E1-23 45": "ABCDE12345",
		"":               "",
		" - ":            "",
	} {
		if got := repositories.NormalizeJoinCode(in); got != want {
			t.Errorf("NormalizeJoinCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCreateCampaignStoresJoinCode(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")

	code, _ := repositories.NewJoinCode()
	now := time.Now()
	c := models.Campaign{ID: uuid.New(), OwnerID: "owner", Name: "C", JoinCode: code, CreatedAt: now, UpdatedAt: now}
	if err := repositories.CreateCampaign(c); err != nil {
		t.Fatal(err)
	}

	got, err := repositories.GetJoinCode(c.ID.String())
	if err != nil || got != code {
		t.Errorf("GetJoinCode = %q, %v; want %q", got, err, code)
	}

	// reading a campaign never loads the code, so public responses cannot carry it
	loaded, err := repositories.GetCampaignByID(c.ID.String())
	if err != nil || loaded.JoinCode != "" {
		t.Errorf("GetCampaignByID returned JoinCode %q (err %v); want it empty", loaded.JoinCode, err)
	}

	if _, err := repositories.GetJoinCode(uuid.NewString()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("unknown campaign err = %v, want pgx.ErrNoRows", err)
	}
}

func TestRotateJoinCode(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "intruder")
	c := testutil.InsertCampaign(t, "owner", "C")
	id := c.ID.String()

	if _, err := repositories.RotateJoinCode(id, "intruder"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("non-owner rotate err = %v, want pgx.ErrNoRows", err)
	}
	if got, _ := repositories.GetJoinCode(id); got != c.JoinCode {
		t.Errorf("a refused rotation changed the code to %q", got)
	}

	fresh, err := repositories.RotateJoinCode(id, "owner")
	if err != nil {
		t.Fatalf("RotateJoinCode: %v", err)
	}
	if fresh == c.JoinCode || len(fresh) != 10 {
		t.Errorf("new code %q should differ from %q and have 10 characters", fresh, c.JoinCode)
	}
	if got, _ := repositories.GetJoinCode(id); got != fresh {
		t.Errorf("stored code = %q, want %q", got, fresh)
	}
}

func TestJoinCampaignChecksTheCode(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	c := testutil.InsertCampaign(t, "owner", "C") // join code TESTCODE01
	team := testutil.InsertTeam(t, c.ID, "Red")
	cid, tid := c.ID.String(), team.ID.String()
	str := func(s string) *string { return &s }

	for _, wrong := range []string{"", "WRONGCODE1", "TESTCODE0", "TESTCODE011", "TESTCODE01 x"} {
		wb := testutil.InsertWarband(t, "player", "w")
		if err := repositories.JoinCampaign(cid, wb.ID.String(), tid, str(wrong)); !errors.Is(err, repositories.ErrInvalidJoinCode) {
			t.Errorf("code %q: err = %v, want ErrInvalidJoinCode", wrong, err)
		}
	}
	if members, _ := repositories.GetCampaignWarbands(cid); len(members) != 0 {
		t.Fatalf("%d warbands joined with a wrong code", len(members))
	}

	// right code, in any case or grouping
	for _, code := range []string{"TESTCODE01", "testcode01", "TEST-CODE01", " test code01 "} {
		wb := testutil.InsertWarband(t, "player", "w")
		if err := repositories.JoinCampaign(cid, wb.ID.String(), tid, str(code)); err != nil {
			t.Errorf("code %q should work: %v", code, err)
		}
	}

	// nil means the caller is already authorized (the campaign owner)
	wb := testutil.InsertWarband(t, "owner", "mine")
	if err := repositories.JoinCampaign(cid, wb.ID.String(), tid, nil); err != nil {
		t.Errorf("nil code should skip the check: %v", err)
	}

	// an unknown campaign is not found, whatever code is given
	err := repositories.JoinCampaign(uuid.NewString(), wb.ID.String(), tid, str("TESTCODE01"))
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("unknown campaign err = %v, want pgx.ErrNoRows", err)
	}
}

// TestJoinCampaign_WrongCodeBeatsOtherErrors checks that a wrong code is refused
// before anything else is revealed: not the team check, and not the member limit.
func TestJoinCampaign_WrongCodeBeatsOtherErrors(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	wb := testutil.InsertWarband(t, "owner", "w")
	wrong := "WRONGCODE1"

	err := repositories.JoinCampaign(c.ID.String(), wb.ID.String(), uuid.NewString(), &wrong) // team does not exist
	if !errors.Is(err, repositories.ErrInvalidJoinCode) {
		t.Errorf("err = %v, want ErrInvalidJoinCode, not a team-not-found error", err)
	}
}

func TestRotatingStopsTheOldCode(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	c := testutil.InsertCampaign(t, "owner", "C")
	team := testutil.InsertTeam(t, c.ID, "Red")
	cid := c.ID.String()
	existing := testutil.InsertWarband(t, "player", "already in")
	old := c.JoinCode
	if err := repositories.JoinCampaign(cid, existing.ID.String(), team.ID.String(), &old); err != nil {
		t.Fatal(err)
	}

	fresh, err := repositories.RotateJoinCode(cid, "owner")
	if err != nil {
		t.Fatal(err)
	}

	late := testutil.InsertWarband(t, "player", "late")
	if err := repositories.JoinCampaign(cid, late.ID.String(), team.ID.String(), &old); !errors.Is(err, repositories.ErrInvalidJoinCode) {
		t.Errorf("old code err = %v, want ErrInvalidJoinCode", err)
	}
	if err := repositories.JoinCampaign(cid, late.ID.String(), team.ID.String(), &fresh); err != nil {
		t.Errorf("new code should work: %v", err)
	}
	// rotating does not remove anyone who already joined
	if members, _ := repositories.GetCampaignWarbands(cid); len(members) != 2 {
		t.Errorf("%d members, want 2 (rotation must not remove existing ones)", len(members))
	}
}
