package v1_test

import (
	"net/http"
	"strings"
	"testing"
)

// Only input validation is covered: a successful registration needs a Firebase
// client, which tests don't create.
func TestRegisterUser_InvalidBody(t *testing.T) {
	r := newRouter()

	w := call(r, "", http.MethodPost, "/auth/register", `{not json`)
	expect(t, w, http.StatusBadRequest)
}

// Fields are limited to the users table's varchar sizes; longer values are
// rejected up front instead of failing the insert (and a Firebase rollback).
func TestRegisterUser_FieldTooLong(t *testing.T) {
	r := newRouter()

	for name, body := range map[string]string{
		"username over 50": `{"email":"a@example.com","username":"` + strings.Repeat("u", 51) + `","password":"secret123"}`,
		"email over 255":   `{"email":"` + strings.Repeat("e", 250) + `@example.com","username":"alice","password":"secret123"}`,
	} {
		t.Run(name, func(t *testing.T) {
			expect(t, call(r, "", http.MethodPost, "/auth/register", body), http.StatusBadRequest)
		})
	}
}
