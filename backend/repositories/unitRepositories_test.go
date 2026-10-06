package repositories_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"battlebarge/models"
	"battlebarge/repositories"
	"battlebarge/testutil"
)

// setupUnit creates a user, warband and unit, returning the unit.
func setupUnit(t *testing.T) models.Unit {
	t.Helper()
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	w := testutil.InsertWarband(t, "owner", "W")
	return testutil.InsertUnit(t, w.ID, "Boss", 80)
}

func TestCreateUnit(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	w := testutil.InsertWarband(t, "owner", "W")

	now := time.Now().UTC().Truncate(time.Microsecond)
	unit := models.Unit{
		ID: uuid.New(), WarbandID: w.ID, UnitName: "Nob", NarrativeName: "Grukk", Bio: "mean",
		Points: 90, Kills: 2, Experience: 7,
		Perks:     []models.Perk{{ID: uuid.New(), Name: "Tough", Description: "d", IsScar: false}},
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repositories.CreateUnit(unit); err != nil {
		t.Fatalf("CreateUnit: %v", err)
	}

	got, err := repositories.GetUnitByID(unit.ID.String())
	if err != nil {
		t.Fatalf("GetUnitByID: %v", err)
	}
	if got.UnitName != "Nob" || got.NarrativeName != "Grukk" || got.Bio != "mean" ||
		got.Points != 90 || got.Kills != 2 || got.Experience != 7 || got.WarbandID != w.ID {
		t.Errorf("unexpected unit: %+v", got)
	}
	if len(got.Perks) != 1 || got.Perks[0].Name != "Tough" || got.Perks[0].Description != "d" {
		t.Errorf("perks not saved with unit: %+v", got.Perks)
	}
}

func TestCreateUnit_UnknownWarband(t *testing.T) {
	testutil.SetupDB(t)

	now := time.Now()
	unit := models.Unit{ID: uuid.New(), WarbandID: uuid.New(), UnitName: "x", Perks: []models.Perk{}, CreatedAt: now, UpdatedAt: now}
	if err := repositories.CreateUnit(unit); err == nil {
		t.Error("expected foreign key error for unknown warband")
	}
}

func TestGetUnitByID(t *testing.T) {
	unit := setupUnit(t)

	got, err := repositories.GetUnitByID(unit.ID.String())
	if err != nil {
		t.Fatalf("GetUnitByID: %v", err)
	}
	if got.Perks == nil {
		t.Error("Perks should be an empty slice, not nil (serializes as [] not null)")
	}

	_, err = repositories.GetUnitByID(uuid.NewString())
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("missing unit err = %v, want pgx.ErrNoRows", err)
	}
}

func TestGetUnitsByWarbandID(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	w1 := testutil.InsertWarband(t, "owner", "W1")
	w2 := testutil.InsertWarband(t, "owner", "W2")
	a := testutil.InsertUnit(t, w1.ID, "a", 1)
	time.Sleep(2 * time.Millisecond)
	b := testutil.InsertUnit(t, w1.ID, "b", 2)
	testutil.InsertUnit(t, w2.ID, "other", 3)

	for _, name := range []string{"one", "two"} {
		if _, err := repositories.AddUnitPerk(a.ID.String(), models.AddPerkRequest{ID: uuid.New(), Name: name}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := repositories.GetUnitsByWarbandID(w1.ID.String())
	if err != nil {
		t.Fatalf("GetUnitsByWarbandID: %v", err)
	}
	if len(got) != 2 || got[0].ID != a.ID || got[1].ID != b.ID {
		t.Fatalf("want units [a b] oldest first, got %+v", got)
	}
	if len(got[0].Perks) != 2 || got[0].Perks[0].Name != "one" || got[0].Perks[1].Name != "two" {
		t.Errorf("perks not attached to the right unit in order: %+v", got[0].Perks)
	}
	if got[1].Perks == nil || len(got[1].Perks) != 0 {
		t.Errorf("unit without perks should have empty non-nil slice, got %#v", got[1].Perks)
	}

	empty, err := repositories.GetUnitsByWarbandID(uuid.NewString())
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("unknown warband = %v, %v; want empty non-nil slice", empty, err)
	}
}

func TestUpdateUnit(t *testing.T) {
	unit := setupUnit(t)

	name := "Warboss"
	pts := 120
	got, err := repositories.UpdateUnit(unit.ID.String(), models.UpdateUnitRequest{UnitName: &name, Points: &pts})
	if err != nil {
		t.Fatalf("UpdateUnit: %v", err)
	}
	if got.UnitName != "Warboss" || got.Points != 120 {
		t.Errorf("update not applied: %+v", got)
	}

	_, err = repositories.UpdateUnit(uuid.NewString(), models.UpdateUnitRequest{UnitName: &name})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("missing unit err = %v, want pgx.ErrNoRows", err)
	}
}

func TestUpdateUnit_NilFieldsUnchanged(t *testing.T) {
	unit := setupUnit(t)

	bio := "new bio"
	got, err := repositories.UpdateUnit(unit.ID.String(), models.UpdateUnitRequest{Bio: &bio})
	if err != nil {
		t.Fatal(err)
	}
	if got.UnitName != "Boss" || got.Points != 80 || got.Bio != "new bio" {
		t.Errorf("nil fields should be left alone: %+v", got)
	}
}

func TestIncrementUnitKillsAndXP(t *testing.T) {
	unit := setupUnit(t)
	id := unit.ID.String()

	tests := []struct {
		name string
		fn   func(string, int) (models.Unit, error)
		get  func(models.Unit) int
	}{
		{"kills", repositories.IncrementUnitKills, func(u models.Unit) int { return u.Kills }},
		{"xp", repositories.IncrementUnitXP, func(u models.Unit) int { return u.Experience }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			steps := []struct{ amount, want int }{
				{3, 3},
				{2, 5},
				{-2, 3},
				{-100, 0}, // clamps at zero
			}
			for _, s := range steps {
				got, err := tt.fn(id, s.amount)
				if err != nil {
					t.Fatalf("increment %d: %v", s.amount, err)
				}
				if tt.get(got) != s.want {
					t.Errorf("after %+d got %d, want %d", s.amount, tt.get(got), s.want)
				}
			}
			if _, err := tt.fn(uuid.NewString(), 1); !errors.Is(err, pgx.ErrNoRows) {
				t.Errorf("missing unit err = %v, want pgx.ErrNoRows", err)
			}
		})
	}
}

func TestAddAndDeleteUnitPerk(t *testing.T) {
	unit := setupUnit(t)
	id := unit.ID.String()

	desc := "scarred"
	perkA := models.AddPerkRequest{ID: uuid.New(), Name: "Tough"}
	perkB := models.AddPerkRequest{ID: uuid.New(), Name: "Scar", Description: &desc, IsScar: true}

	if _, err := repositories.AddUnitPerk(id, perkA); err != nil {
		t.Fatalf("AddUnitPerk A: %v", err)
	}
	got, err := repositories.AddUnitPerk(id, perkB)
	if err != nil {
		t.Fatalf("AddUnitPerk B: %v", err)
	}
	if len(got.Perks) != 2 {
		t.Fatalf("got %d perks, want 2", len(got.Perks))
	}
	if got.Perks[0].ID != perkA.ID || got.Perks[0].Description != "" || got.Perks[0].IsScar {
		t.Errorf("perk A wrong: %+v", got.Perks[0])
	}
	if got.Perks[1].ID != perkB.ID || got.Perks[1].Description != "scarred" || !got.Perks[1].IsScar {
		t.Errorf("perk B wrong: %+v", got.Perks[1])
	}
	if !got.UpdatedAt.After(unit.UpdatedAt) {
		t.Error("adding a perk should bump updated_at")
	}

	got, err = repositories.DeleteUnitPerk(id, perkA.ID.String())
	if err != nil {
		t.Fatalf("DeleteUnitPerk: %v", err)
	}
	if len(got.Perks) != 1 || got.Perks[0].ID != perkB.ID {
		t.Errorf("wrong perks left: %+v", got.Perks)
	}
}

func TestAddUnitPerk_UnknownUnit(t *testing.T) {
	testutil.SetupDB(t)

	_, err := repositories.AddUnitPerk(uuid.NewString(), models.AddPerkRequest{ID: uuid.New(), Name: "x"})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("err = %v, want pgx.ErrNoRows", err)
	}
}

func TestDeleteUnitPerk_NotFound(t *testing.T) {
	unit := setupUnit(t)
	other := testutil.InsertUnit(t, unit.WarbandID, "other", 1)
	perk := models.AddPerkRequest{ID: uuid.New(), Name: "mine"}
	if _, err := repositories.AddUnitPerk(other.ID.String(), perk); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		unitID string
		perkID string
	}{
		{"unknown perk", unit.ID.String(), uuid.NewString()},
		{"invalid perk id", unit.ID.String(), "not-a-uuid"},
		{"perk belongs to another unit", unit.ID.String(), perk.ID.String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repositories.DeleteUnitPerk(tt.unitID, tt.perkID)
			if !errors.Is(err, pgx.ErrNoRows) {
				t.Errorf("err = %v, want pgx.ErrNoRows", err)
			}
		})
	}

	// the other unit's perk must be untouched
	got, err := repositories.GetUnitByID(other.ID.String())
	if err != nil || len(got.Perks) != 1 {
		t.Errorf("other unit's perk was affected: %+v, %v", got.Perks, err)
	}
}

func TestDeleteUnit(t *testing.T) {
	unit := setupUnit(t)
	id := unit.ID.String()
	if _, err := repositories.AddUnitPerk(id, models.AddPerkRequest{ID: uuid.New(), Name: "x"}); err != nil {
		t.Fatal(err)
	}

	if err := repositories.DeleteUnit(id); err != nil {
		t.Fatalf("DeleteUnit: %v", err)
	}
	if _, err := repositories.GetUnitByID(id); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("unit still present: %v", err)
	}
	perks, err := repositories.GetPerksByUnitID(id)
	if err != nil || len(perks) != 0 {
		t.Errorf("perks should cascade-delete with unit: %v, %v", perks, err)
	}
	if err := repositories.DeleteUnit(id); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("second delete err = %v, want pgx.ErrNoRows", err)
	}
}
