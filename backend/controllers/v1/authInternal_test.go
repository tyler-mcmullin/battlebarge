package v1

import (
	"errors"
	"testing"
	"time"

	"battlebarge/models"
	"battlebarge/repositories"
	"battlebarge/testutil"
)

// TestUserInsertConflict tests RegisterUser's error mapping directly, with the
// real errors Postgres gives for each kind of duplicate, because RegisterUser
// itself needs Firebase.
func TestUserInsertConflict(t *testing.T) {
	testutil.SetupDB(t)

	now := time.Now()
	user := func(id, email, username string) models.User {
		return models.User{ID: id, Email: email, Username: username, CreatedAt: now, UpdatedAt: now}
	}
	if err := repositories.CreateUser(user("u1", "a@example.com", "Alice")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		u    models.User
		want string
	}{
		{"same username", user("u2", "b@example.com", "Alice"), "username already taken"},
		{"username in another case", user("u3", "c@example.com", "ALICE"), "username already taken"},
		{"same email", user("u4", "a@example.com", "Bob"), "email already exists"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repositories.CreateUser(tt.u)
			msg, ok := userInsertConflict(err)
			if !ok || msg != tt.want {
				t.Errorf("userInsertConflict = %q, %v; want %q, true (err: %v)", msg, ok, tt.want, err)
			}
		})
	}

	if msg, ok := userInsertConflict(errors.New("connection reset")); ok {
		t.Errorf("an unrelated error was reported as a conflict: %q", msg)
	}
	if _, ok := userInsertConflict(nil); ok {
		t.Error("nil must not be a conflict")
	}
}
