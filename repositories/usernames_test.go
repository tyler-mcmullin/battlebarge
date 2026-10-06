package repositories_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"battlebarge/models"
	"battlebarge/repositories"
	"battlebarge/testutil"
)

// Usernames are unique ignoring case, but keep the capitalization they were
// registered with.
func TestUsernamesAreUniqueIgnoringCase(t *testing.T) {
	testutil.SetupDB(t)

	now := time.Now()
	user := func(id, username string) models.User {
		return models.User{ID: id, Email: id + "@example.com", Username: username, CreatedAt: now, UpdatedAt: now}
	}

	if err := repositories.CreateUser(user("u1", "Alice")); err != nil {
		t.Fatal(err)
	}

	for _, variant := range []string{"alice", "ALICE", "aLiCe", "Alice"} {
		err := repositories.CreateUser(user("x-"+variant, variant))
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
			t.Errorf("username %q: err = %v, want a unique violation", variant, err)
		}
	}

	// different names are fine, and the stored capitalization is kept
	if err := repositories.CreateUser(user("u2", "Alicia")); err != nil {
		t.Errorf("a different username should be allowed: %v", err)
	}
	got, err := repositories.GetUserByID("u1")
	if err != nil || got.Username != "Alice" {
		t.Errorf("stored username = %q (err %v), want it kept as Alice", got.Username, err)
	}
}
