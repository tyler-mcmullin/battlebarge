package repositories_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"battlebarge/models"
	"battlebarge/repositories"
	"battlebarge/testutil"
)

func pgCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func TestCreateAndGetCampaign(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")

	now := time.Now().UTC().Truncate(time.Microsecond)
	c := models.Campaign{
		ID: uuid.New(), OwnerID: "owner", Name: "Crusade", Description: "d",
		Settings:  models.CampaignSettings{PointsPerWin: 3, PointsPerLoss: 1, StartingRequisition: 5},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repositories.CreateCampaign(c); err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}

	got, err := repositories.GetCampaignByID(c.ID.String())
	if err != nil {
		t.Fatalf("GetCampaignByID: %v", err)
	}
	if got.Name != "Crusade" || got.OwnerID != "owner" || got.Settings != c.Settings {
		t.Errorf("unexpected campaign: %+v", got)
	}
	if got.Chapters == nil || got.Teams == nil || got.Warbands == nil {
		t.Errorf("child slices should be empty non-nil: %+v", got)
	}

	if _, err := repositories.GetCampaignByID(uuid.NewString()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("missing campaign err = %v, want pgx.ErrNoRows", err)
	}
}

func TestGetCampaignsForUser(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	testutil.InsertUser(t, "stranger")

	owned := testutil.InsertCampaign(t, "owner", "owned")
	joined := testutil.InsertCampaign(t, "stranger", "joined")
	testutil.InsertCampaign(t, "stranger", "unrelated")
	team := testutil.InsertTeam(t, joined.ID, "Red")
	wb := testutil.InsertWarband(t, "owner", "W")
	if err := repositories.JoinCampaign(joined.ID.String(), wb.ID.String(), team.ID.String(), nil); err != nil {
		t.Fatal(err)
	}

	got, err := repositories.GetCampaignsForUser("owner")
	if err != nil {
		t.Fatalf("GetCampaignsForUser: %v", err)
	}
	ids := map[uuid.UUID]bool{}
	for _, c := range got {
		ids[c.ID] = true
	}
	if len(got) != 2 || !ids[owned.ID] || !ids[joined.ID] {
		t.Errorf("owner should see owned + joined campaigns only, got %d: %+v", len(got), got)
	}

	none, err := repositories.GetCampaignsForUser("player")
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("player = %v, %v; want empty non-nil slice", none, err)
	}
}

func TestUpdateAndDeleteCampaign(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "intruder")
	c := testutil.InsertCampaign(t, "owner", "Before")
	id := c.ID.String()

	name := "After"
	pts := 4
	got, err := repositories.UpdateCampaign(id, "owner", models.UpdateCampaignRequest{Name: &name, PointsPerWin: &pts})
	if err != nil {
		t.Fatalf("UpdateCampaign: %v", err)
	}
	if got.Name != "After" || got.Settings.PointsPerWin != 4 {
		t.Errorf("update not applied: %+v", got)
	}

	if _, err := repositories.UpdateCampaign(id, "intruder", models.UpdateCampaignRequest{Name: &name}); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("non-owner update err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.DeleteCampaign(id, "intruder"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("non-owner delete err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.DeleteCampaign(id, "owner"); err != nil {
		t.Fatalf("DeleteCampaign: %v", err)
	}
	if _, err := repositories.GetCampaignByID(id); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("campaign still present: %v", err)
	}
}

func TestIsCampaignOwner(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")

	for _, tt := range []struct {
		name       string
		campaignID string
		userID     string
		want       bool
	}{
		{"owner", c.ID.String(), "owner", true},
		{"other user", c.ID.String(), "someone", false},
		{"unknown campaign", uuid.NewString(), "owner", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repositories.IsCampaignOwner(tt.campaignID, tt.userID)
			if err != nil || got != tt.want {
				t.Errorf("got %v, %v; want %v", got, err, tt.want)
			}
		})
	}
}

func TestChapters(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	other := testutil.InsertCampaign(t, "owner", "Other")
	id := c.ID.String()

	add := func(title string, order int, auto bool) models.CampaignChapter {
		now := time.Now()
		ch, err := repositories.AddChapter(models.CampaignChapter{
			ID: uuid.New(), CampaignID: c.ID, Title: title, SortOrder: order, CreatedAt: now, UpdatedAt: now,
		}, auto)
		if err != nil {
			t.Fatalf("AddChapter %q: %v", title, err)
		}
		return ch
	}

	first := add("one", 0, true)
	second := add("two", 0, true)
	explicit := add("zero", -1, false)
	if first.SortOrder != 1 || second.SortOrder != 2 || explicit.SortOrder != -1 {
		t.Errorf("sort orders = %d, %d, %d; want 1, 2, -1", first.SortOrder, second.SortOrder, explicit.SortOrder)
	}

	got, err := repositories.GetChaptersByCampaignID(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Title != "zero" || got[1].Title != "one" || got[2].Title != "two" {
		t.Errorf("chapters not ordered by sort_order: %+v", got)
	}

	title := "renamed"
	order := 10
	upd, err := repositories.UpdateChapter(id, first.ID.String(), models.UpdateChapterRequest{Title: &title, SortOrder: &order})
	if err != nil || upd.Title != "renamed" || upd.SortOrder != 10 {
		t.Errorf("UpdateChapter = %+v, %v", upd, err)
	}

	// a chapter can only be touched through its own campaign
	if _, err := repositories.UpdateChapter(other.ID.String(), first.ID.String(), models.UpdateChapterRequest{Title: &title}); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("cross-campaign update err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.DeleteChapter(other.ID.String(), first.ID.String()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("cross-campaign delete err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.DeleteChapter(id, first.ID.String()); err != nil {
		t.Fatalf("DeleteChapter: %v", err)
	}
	if err := repositories.DeleteChapter(id, first.ID.String()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("second delete err = %v, want pgx.ErrNoRows", err)
	}
}

func TestTeams(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	other := testutil.InsertCampaign(t, "owner", "Other")
	id := c.ID.String()

	// any number of teams
	for _, n := range []string{"Red", "Blue", "Green", "Yellow"} {
		now := time.Now()
		if err := repositories.CreateTeam(models.CampaignTeam{ID: uuid.New(), CampaignID: c.ID, Name: n, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatalf("CreateTeam %s: %v", n, err)
		}
	}
	teams, err := repositories.GetTeamsByCampaignID(id)
	if err != nil || len(teams) != 4 {
		t.Fatalf("got %d teams, %v; want 4", len(teams), err)
	}

	// names are unique within a campaign but reusable across campaigns
	now := time.Now()
	dup := models.CampaignTeam{ID: uuid.New(), CampaignID: c.ID, Name: "Red", CreatedAt: now, UpdatedAt: now}
	if err := repositories.CreateTeam(dup); pgCode(err) != "23505" {
		t.Errorf("duplicate name err = %v, want 23505", err)
	}
	dup.ID, dup.CampaignID = uuid.New(), other.ID
	if err := repositories.CreateTeam(dup); err != nil {
		t.Errorf("same name in another campaign should be allowed: %v", err)
	}

	// rename
	red := teams[0]
	renamed, err := repositories.RenameTeam(id, red.ID.String(), "Crimson")
	if err != nil || renamed.Name != "Crimson" || renamed.ID != red.ID {
		t.Errorf("RenameTeam = %+v, %v", renamed, err)
	}
	if _, err := repositories.RenameTeam(id, teams[1].ID.String(), "Crimson"); pgCode(err) != "23505" {
		t.Errorf("rename to taken name err = %v, want 23505", err)
	}
	if _, err := repositories.RenameTeam(other.ID.String(), red.ID.String(), "x"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("cross-campaign rename err = %v, want pgx.ErrNoRows", err)
	}

	// delete
	if err := repositories.DeleteTeam(other.ID.String(), red.ID.String()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("cross-campaign delete err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.DeleteTeam(id, red.ID.String()); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
}

func TestJoinMoveAndLeaveCampaign(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "player")
	c := testutil.InsertCampaign(t, "owner", "C")
	other := testutil.InsertCampaign(t, "owner", "Other")
	red := testutil.InsertTeam(t, c.ID, "Red")
	blue := testutil.InsertTeam(t, c.ID, "Blue")
	foreign := testutil.InsertTeam(t, other.ID, "Foreign")
	wb := testutil.InsertWarband(t, "player", "Da Boyz")
	cid, wid := c.ID.String(), wb.ID.String()

	if err := repositories.JoinCampaign(cid, wid, red.ID.String(), nil); err != nil {
		t.Fatalf("JoinCampaign: %v", err)
	}
	members, err := repositories.GetCampaignWarbands(cid)
	if err != nil || len(members) != 1 {
		t.Fatalf("members = %+v, %v", members, err)
	}
	if members[0].WarbandID != wb.ID || members[0].WarbandName != "Da Boyz" || members[0].TeamID != red.ID {
		t.Errorf("unexpected member: %+v", members[0])
	}

	// once per campaign
	if err := repositories.JoinCampaign(cid, wid, blue.ID.String(), nil); pgCode(err) != "23505" {
		t.Errorf("rejoin err = %v, want 23505", err)
	}
	// a team from another campaign is rejected
	wb2 := testutil.InsertWarband(t, "player", "Second")
	if err := repositories.JoinCampaign(cid, wb2.ID.String(), foreign.ID.String(), nil); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("foreign team err = %v, want pgx.ErrNoRows", err)
	}
	// a warband can be in several campaigns
	if err := repositories.JoinCampaign(other.ID.String(), wid, foreign.ID.String(), nil); err != nil {
		t.Errorf("joining a second campaign should work: %v", err)
	}

	// move
	if err := repositories.SetWarbandTeam(cid, wid, blue.ID.String()); err != nil {
		t.Fatalf("SetWarbandTeam: %v", err)
	}
	members, _ = repositories.GetCampaignWarbands(cid)
	if members[0].TeamID != blue.ID {
		t.Errorf("team = %v, want blue", members[0].TeamID)
	}
	if err := repositories.SetWarbandTeam(cid, wid, foreign.ID.String()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("move to foreign team err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.SetWarbandTeam(cid, wb2.ID.String(), red.ID.String()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("move non-member err = %v, want pgx.ErrNoRows", err)
	}

	// a team with warbands on it can't be deleted until they move
	if err := repositories.DeleteTeam(cid, blue.ID.String()); pgCode(err) != "23503" {
		t.Errorf("delete occupied team err = %v, want 23503", err)
	}
	if err := repositories.DeleteTeam(cid, red.ID.String()); err != nil {
		t.Errorf("delete empty team: %v", err)
	}

	// leave
	if err := repositories.LeaveCampaign(cid, wid); err != nil {
		t.Fatalf("LeaveCampaign: %v", err)
	}
	if err := repositories.LeaveCampaign(cid, wid); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("second leave err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.DeleteTeam(cid, blue.ID.String()); err != nil {
		t.Errorf("team should be deletable after warband left: %v", err)
	}
}

func TestCampaignCascades(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	team := testutil.InsertTeam(t, c.ID, "Red")
	wb := testutil.InsertWarband(t, "owner", "W")
	cid := c.ID.String()
	if err := repositories.JoinCampaign(cid, wb.ID.String(), team.ID.String(), nil); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if _, err := repositories.AddChapter(models.CampaignChapter{ID: uuid.New(), CampaignID: c.ID, Title: "x", CreatedAt: now, UpdatedAt: now}, true); err != nil {
		t.Fatal(err)
	}

	// Deleting a campaign removes teams (despite occupied), chapters, and memberships,
	// but not the warband itself.
	if err := repositories.DeleteCampaign(cid, "owner"); err != nil {
		t.Fatalf("DeleteCampaign: %v", err)
	}
	if _, err := repositories.GetWarbandByID(wb.ID.String()); err != nil {
		t.Errorf("warband should survive campaign deletion: %v", err)
	}
	for name, n := range map[string]func() (int, error){
		"teams":    func() (int, error) { x, e := repositories.GetTeamsByCampaignID(cid); return len(x), e },
		"chapters": func() (int, error) { x, e := repositories.GetChaptersByCampaignID(cid); return len(x), e },
		"members":  func() (int, error) { x, e := repositories.GetCampaignWarbands(cid); return len(x), e },
	} {
		if got, err := n(); err != nil || got != 0 {
			t.Errorf("%s left behind after campaign delete: %d, %v", name, got, err)
		}
	}
}

func TestDeletingWarbandLeavesCampaign(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	team := testutil.InsertTeam(t, c.ID, "Red")
	wb := testutil.InsertWarband(t, "owner", "W")
	if err := repositories.JoinCampaign(c.ID.String(), wb.ID.String(), team.ID.String(), nil); err != nil {
		t.Fatal(err)
	}

	if err := repositories.DeleteWarband(wb.ID.String(), "owner"); err != nil {
		t.Fatalf("DeleteWarband: %v", err)
	}
	members, err := repositories.GetCampaignWarbands(c.ID.String())
	if err != nil || len(members) != 0 {
		t.Errorf("members after warband delete = %+v, %v; want none", members, err)
	}
}
