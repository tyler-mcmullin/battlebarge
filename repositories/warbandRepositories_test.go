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

func perks(scars ...bool) []models.Perk {
	out := []models.Perk{}
	for _, s := range scars {
		out = append(out, models.Perk{ID: uuid.New(), Name: "p", IsScar: s})
	}
	return out
}

func TestCalculateCrusadePoints(t *testing.T) {
	tests := []struct {
		name  string
		units []models.Unit
		want  int
	}{
		{"no units", nil, 0},
		{"unit without perks", []models.Unit{{Perks: perks()}}, 0},
		{"perks add a point each", []models.Unit{{Perks: perks(false, false, false)}}, 3},
		{"scars subtract a point each", []models.Unit{{Perks: perks(true, true)}}, -2},
		{"perks and scars across units", []models.Unit{
			{Perks: perks(false, false, true)},
			{Perks: perks(true)},
			{Perks: perks(false)},
		}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := repositories.CalculateCrusadePoints(tt.units); got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

func newWarband(userID, name string) models.Warband {
	now := time.Now().UTC().Truncate(time.Microsecond)
	return models.Warband{
		ID: uuid.New(), UserID: userID, Name: name, Faction: "Orks", Description: "desc",
		RequisitionPoints: 5, SupplyLimit: 1000, CreatedAt: now, UpdatedAt: now,
	}
}

func TestCreateAndGetWarband(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")

	w := newWarband("owner", "Da Boyz")
	if err := repositories.CreateWarband(w); err != nil {
		t.Fatalf("CreateWarband: %v", err)
	}

	got, err := repositories.GetWarbandByID(w.ID.String())
	if err != nil {
		t.Fatalf("GetWarbandByID: %v", err)
	}
	if got.Name != "Da Boyz" || got.Faction != "Orks" || got.RequisitionPoints != 5 || got.SupplyLimit != 1000 {
		t.Errorf("unexpected warband: %+v", got)
	}
	if got.Units == nil || got.NumUnits != 0 || got.TotalPointsCost != 0 || got.CrusadePoints != 0 {
		t.Errorf("empty warband computed fields wrong: %+v", got)
	}
}

func TestGetWarbandByID_NotFound(t *testing.T) {
	testutil.SetupDB(t)

	_, err := repositories.GetWarbandByID(uuid.NewString())
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("err = %v, want pgx.ErrNoRows", err)
	}
}

func TestGetWarbandByID_ComputedFields(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	w := testutil.InsertWarband(t, "owner", "W")
	u1 := testutil.InsertUnit(t, w.ID, "u1", 100)
	u2 := testutil.InsertUnit(t, w.ID, "u2", 50)

	for _, p := range []struct {
		unit uuid.UUID
		scar bool
	}{{u1.ID, false}, {u1.ID, false}, {u2.ID, true}} {
		if _, err := repositories.AddUnitPerk(p.unit.String(), models.AddPerkRequest{ID: uuid.New(), Name: "x", IsScar: p.scar}); err != nil {
			t.Fatalf("AddUnitPerk: %v", err)
		}
	}

	got, err := repositories.GetWarbandByID(w.ID.String())
	if err != nil {
		t.Fatalf("GetWarbandByID: %v", err)
	}
	if got.NumUnits != 2 || got.TotalPointsCost != 150 || got.CrusadePoints != 1 {
		t.Errorf("num=%d points=%d crusade=%d, want 2/150/1", got.NumUnits, got.TotalPointsCost, got.CrusadePoints)
	}
}

func TestGetAllWarbands(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "a")
	testutil.InsertUser(t, "b")

	older := newWarband("a", "older")
	older.CreatedAt = older.CreatedAt.Add(-time.Hour)
	newer := newWarband("a", "newer")
	other := newWarband("b", "other")
	for _, w := range []models.Warband{older, newer, other} {
		if err := repositories.CreateWarband(w); err != nil {
			t.Fatalf("CreateWarband: %v", err)
		}
	}
	testutil.InsertUnit(t, newer.ID, "u", 10)

	got, err := repositories.GetAllWarbands("a")
	if err != nil {
		t.Fatalf("GetAllWarbands: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d warbands, want 2 (only user a's)", len(got))
	}
	if got[0].Name != "newer" || got[1].Name != "older" {
		t.Errorf("order = %q, %q; want newest first", got[0].Name, got[1].Name)
	}
	if got[0].NumUnits != 1 || got[0].TotalPointsCost != 10 {
		t.Errorf("computed fields not populated: %+v", got[0])
	}

	none, err := repositories.GetAllWarbands("nobody")
	if err != nil || none == nil || len(none) != 0 {
		t.Errorf("GetAllWarbands for unknown user = %v, %v; want empty non-nil slice", none, err)
	}
}

func TestUpdateWarband(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "intruder")
	w := newWarband("owner", "Original")
	if err := repositories.CreateWarband(w); err != nil {
		t.Fatal(err)
	}

	newName := "Renamed"
	got, err := repositories.UpdateWarband(w.ID.String(), "owner", models.UpdateWarbandRequest{Name: &newName})
	if err != nil {
		t.Fatalf("UpdateWarband: %v", err)
	}
	if got.Name != "Renamed" {
		t.Errorf("Name = %q, want Renamed", got.Name)
	}
	if got.Faction != "Orks" || got.SupplyLimit != 1000 {
		t.Errorf("nil fields should be unchanged: %+v", got)
	}

	_, err = repositories.UpdateWarband(w.ID.String(), "intruder", models.UpdateWarbandRequest{Name: &newName})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("non-owner err = %v, want pgx.ErrNoRows", err)
	}
}

func TestDeleteWarband(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	testutil.InsertUser(t, "intruder")
	w := testutil.InsertWarband(t, "owner", "W")
	u := testutil.InsertUnit(t, w.ID, "u", 1)

	if err := repositories.DeleteWarband(w.ID.String(), "intruder"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("non-owner delete err = %v, want pgx.ErrNoRows", err)
	}
	if err := repositories.DeleteWarband(w.ID.String(), "owner"); err != nil {
		t.Fatalf("DeleteWarband: %v", err)
	}
	if _, err := repositories.GetWarbandByID(w.ID.String()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("warband still present after delete: %v", err)
	}
	if _, err := repositories.GetUnitByID(u.ID.String()); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("units should cascade with warband: %v", err)
	}
	if err := repositories.DeleteWarband(w.ID.String(), "owner"); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("second delete err = %v, want pgx.ErrNoRows", err)
	}
}

func TestIsWarbandOwner(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	w := testutil.InsertWarband(t, "owner", "W")

	tests := []struct {
		name      string
		warbandID string
		userID    string
		want      bool
	}{
		{"owner", w.ID.String(), "owner", true},
		{"other user", w.ID.String(), "someone", false},
		{"unknown warband", uuid.NewString(), "owner", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repositories.IsWarbandOwner(tt.warbandID, tt.userID)
			if err != nil {
				t.Fatalf("IsWarbandOwner: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSaveWarband(t *testing.T) {
	testutil.SetupDB(t)
	testutil.InsertUser(t, "owner")
	w := newWarband("owner", "Before")
	if err := repositories.CreateWarband(w); err != nil {
		t.Fatal(err)
	}

	w.Name = "After"
	w.SupplyLimit = 2000
	w.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)
	if err := repositories.SaveWarband(w); err != nil {
		t.Fatalf("SaveWarband: %v", err)
	}

	got, err := repositories.GetWarbandByID(w.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "After" || got.SupplyLimit != 2000 {
		t.Errorf("save not persisted: %+v", got)
	}

	w.UserID = "someone-else"
	if err := repositories.SaveWarband(w); !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("wrong owner err = %v, want pgx.ErrNoRows", err)
	}
}
