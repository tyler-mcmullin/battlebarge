package repositories_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"battlebarge/db"
	"battlebarge/models"
	"battlebarge/repositories"
	"battlebarge/testutil"
)

func TestLimit_WarbandsPerUser(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "other")
	for i := 0; i < repositories.MaxWarbandsPerUser; i++ {
		testutil.InsertWarband(t, "owner", "w")
	}

	if err := repositories.CreateWarband(newWarband("owner", "over")); !errors.Is(err, repositories.ErrLimitReached) {
		t.Errorf("err = %v, want ErrLimitReached", err)
	}
	// the cap is per user
	if err := repositories.CreateWarband(newWarband("other", "fine")); err != nil {
		t.Errorf("another user should be unaffected: %v", err)
	}
}

func TestLimit_UnitsPerWarband(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	full := testutil.InsertWarband(t, "owner", "full")
	empty := testutil.InsertWarband(t, "owner", "empty")
	for i := 0; i < repositories.MaxUnitsPerWarband; i++ {
		testutil.InsertUnit(t, full.ID, "u", 1)
	}

	now := time.Now()
	mk := func(wb uuid.UUID) models.Unit {
		return models.Unit{ID: uuid.New(), WarbandID: wb, UnitName: "x", Perks: []models.Perk{}, CreatedAt: now, UpdatedAt: now}
	}
	if err := repositories.CreateUnit(mk(full.ID)); !errors.Is(err, repositories.ErrLimitReached) {
		t.Errorf("err = %v, want ErrLimitReached", err)
	}
	if err := repositories.CreateUnit(mk(empty.ID)); err != nil {
		t.Errorf("another warband should be unaffected: %v", err)
	}
}

func TestLimit_PerksPerUnit(t *testing.T) {
	unit := setupUnit(t)
	id := unit.ID.String()
	for i := 0; i < repositories.MaxPerksPerUnit; i++ {
		if _, err := repositories.AddUnitPerk(id, models.AddPerkRequest{ID: uuid.New(), Name: "p"}); err != nil {
			t.Fatalf("perk %d: %v", i, err)
		}
	}

	_, err := repositories.AddUnitPerk(id, models.AddPerkRequest{ID: uuid.New(), Name: "over"})
	if !errors.Is(err, repositories.ErrLimitReached) {
		t.Errorf("err = %v, want ErrLimitReached", err)
	}
	got, _ := repositories.GetUnitByID(id)
	if len(got.Perks) != repositories.MaxPerksPerUnit {
		t.Errorf("%d perks stored, want exactly %d", len(got.Perks), repositories.MaxPerksPerUnit)
	}
}

func TestLimit_CampaignsPerUser(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	for i := 0; i < repositories.MaxCampaignsPerUser; i++ {
		testutil.InsertCampaign(t, "owner", "c")
	}

	now := time.Now()
	err := repositories.CreateCampaign(models.Campaign{ID: uuid.New(), OwnerID: "owner", Name: "over", CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, repositories.ErrLimitReached) {
		t.Errorf("err = %v, want ErrLimitReached", err)
	}
}

func TestLimit_ChaptersTeamsAndMembersPerCampaign(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	cid := c.ID.String()
	now := time.Now()

	for i := 0; i < repositories.MaxChaptersPerCampaign; i++ {
		_, err := repositories.AddChapter(models.CampaignChapter{ID: uuid.New(), CampaignID: c.ID, Title: "c", CreatedAt: now, UpdatedAt: now}, true)
		if err != nil {
			t.Fatalf("chapter %d: %v", i, err)
		}
	}
	_, err := repositories.AddChapter(models.CampaignChapter{ID: uuid.New(), CampaignID: c.ID, Title: "over", CreatedAt: now, UpdatedAt: now}, true)
	if !errors.Is(err, repositories.ErrLimitReached) {
		t.Errorf("chapters: err = %v, want ErrLimitReached", err)
	}

	var team models.CampaignTeam
	for i := 0; i < repositories.MaxTeamsPerCampaign; i++ {
		team = testutil.InsertTeam(t, c.ID, fmt.Sprintf("t%d", i))
	}
	err = repositories.CreateTeam(models.CampaignTeam{ID: uuid.New(), CampaignID: c.ID, Name: "over", CreatedAt: now, UpdatedAt: now})
	if !errors.Is(err, repositories.ErrLimitReached) {
		t.Errorf("teams: err = %v, want ErrLimitReached", err)
	}

	for i := 0; i < repositories.MaxWarbandsPerCampaign; i++ {
		wb := testutil.InsertWarband(t, "owner", "w")
		if err := repositories.JoinCampaign(cid, wb.ID.String(), team.ID.String()); err != nil {
			t.Fatalf("member %d: %v", i, err)
		}
	}
	extra := testutil.InsertWarband(t, "owner", "extra")
	if err := repositories.JoinCampaign(cid, extra.ID.String(), team.ID.String()); !errors.Is(err, repositories.ErrLimitReached) {
		t.Errorf("members: err = %v, want ErrLimitReached", err)
	}
}

// A missing parent is reported as not found, not as a limit.
func TestLimit_MissingParent(t *testing.T) {
	testutil.SetupDB(t)
	now := time.Now()

	_, err := repositories.AddChapter(models.CampaignChapter{ID: uuid.New(), CampaignID: uuid.New(), Title: "x", CreatedAt: now, UpdatedAt: now}, true)
	if errors.Is(err, repositories.ErrLimitReached) || err == nil {
		t.Errorf("err = %v, want a not-found style error", err)
	}
}

// The cap must hold when many requests arrive at once: only the free slots get
// filled, however many requests race for them.
func TestLimit_HoldsUnderConcurrency(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	c := testutil.InsertCampaign(t, "owner", "C")
	for i := 0; i < repositories.MaxTeamsPerCampaign-2; i++ { // two slots left
		testutil.InsertTeam(t, c.ID, fmt.Sprintf("t%d", i))
	}

	// Open the pool's connections up front. Otherwise requests queue behind
	// connection setup and run one after another, which hides any race.
	warm := make([]*pgxpool.Conn, 0, 32)
	for i := 0; i < 32; i++ {
		conn, err := db.PGClient.Acquire(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		warm = append(warm, conn)
	}
	for _, conn := range warm {
		conn.Release()
	}

	const racers = 64
	results := make(chan error, racers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			now := time.Now()
			results <- repositories.CreateTeam(models.CampaignTeam{
				ID: uuid.New(), CampaignID: c.ID, Name: fmt.Sprintf("racer-%d", i), CreatedAt: now, UpdatedAt: now,
			})
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)

	ok, limited := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, repositories.ErrLimitReached):
			limited++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 2 || limited != racers-2 {
		t.Errorf("%d succeeded and %d were limited; want exactly 2 and %d", ok, limited, racers-2)
	}
	teams, _ := repositories.GetTeamsByCampaignID(c.ID.String())
	if len(teams) != repositories.MaxTeamsPerCampaign {
		t.Errorf("%d teams stored, want %d", len(teams), repositories.MaxTeamsPerCampaign)
	}
}
