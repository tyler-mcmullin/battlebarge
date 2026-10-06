package repositories_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"battlebarge/models"
	"battlebarge/repositories"
	"battlebarge/testutil"
)

func TestCreateAndGetUser(t *testing.T) {
	testutil.SetupDB(t)

	now := time.Now().UTC().Truncate(time.Microsecond)
	user := models.User{ID: "uid-1", Email: "a@example.com", Username: "alice", CreatedAt: now, UpdatedAt: now}

	if err := repositories.CreateUser(user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	got, err := repositories.GetUserByID("uid-1")
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.ID != user.ID || got.Email != user.Email || got.Username != user.Username {
		t.Errorf("got %+v, want %+v", got, user)
	}
	if !got.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, now)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	testutil.SetupDB(t)

	_, err := repositories.GetUserByID("missing")
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("err = %v, want pgx.ErrNoRows", err)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	testutil.SetupDB(t)

	now := time.Now()
	first := models.User{ID: "u1", Email: "a@example.com", Username: "same", CreatedAt: now, UpdatedAt: now}
	second := models.User{ID: "u2", Email: "b@example.com", Username: "same", CreatedAt: now, UpdatedAt: now}

	if err := repositories.CreateUser(first); err != nil {
		t.Fatalf("CreateUser first: %v", err)
	}

	// RegisterUser maps SQLSTATE 23505 to "username already taken".
	err := repositories.CreateUser(second)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		t.Errorf("err = %v, want unique violation (23505)", err)
	}
}
