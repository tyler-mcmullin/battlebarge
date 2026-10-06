package middleware_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"battlebarge/middleware"
	"battlebarge/models"
	"battlebarge/testutil"
)

func init() { gin.SetMode(gin.TestMode) }

// do runs one request through a router with the given middleware and a
// handler that records whether it was reached.
func do(t *testing.T, header string, mw ...gin.HandlerFunc) (*httptest.ResponseRecorder, bool) {
	t.Helper()

	reached := false
	r := gin.New()
	r.Use(mw...)
	r.GET("/", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, reached
}

// Only the rejection paths are covered: accepting a token needs a Firebase
// client (db.AuthClient), which tests don't create.
func TestRequireAuth_Rejects(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"missing header", ""},
		{"no scheme", "sometoken"},
		{"wrong scheme", "Basic abc123"},
		{"bearer without token", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, reached := do(t, tt.header, middleware.RequireAuth())
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", w.Code)
			}
			if reached {
				t.Error("handler should not run when auth fails")
			}
		})
	}
}

func TestLoadUser_MissingUID(t *testing.T) {
	w, reached := do(t, "", middleware.LoadUser())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if reached {
		t.Error("handler should not run without a uid")
	}
}

func setUID(uid string) gin.HandlerFunc {
	return func(c *gin.Context) { c.Set(middleware.ContextUIDKey, uid) }
}

func TestLoadUser_UnknownUser(t *testing.T) {
	testutil.SetupDB(t)

	w, reached := do(t, "", setUID("ghost"), middleware.LoadUser())
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
	if reached {
		t.Error("handler should not run for an unknown user")
	}
}

func TestLoadUser_AttachesUser(t *testing.T) {
	testutil.SetupDB(t)
	want := testutil.InsertUser(t, "real")

	var got models.User
	r := gin.New()
	r.Use(setUID("real"), middleware.LoadUser())
	r.GET("/", func(c *gin.Context) {
		got = c.MustGet(middleware.ContextUserKey).(models.User)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got.ID != want.ID || got.Username != want.Username {
		t.Errorf("context user = %+v, want %+v", got, want)
	}
}

const frontend = "http://localhost:5173"

// corsRouter has real routes registered for GET only, like the API: the
// preflight OPTIONS requests match no route and must still be answered.
func corsRouter(origins ...string) (*gin.Engine, *bool) {
	reached := false
	r := gin.New()
	r.Use(middleware.CORS(origins))
	r.GET("/warbands", func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	return r, &reached
}

func corsRequest(r *gin.Engine, method, origin string, preflight bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/warbands", nil)
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if preflight {
		req.Header.Set("Access-Control-Request-Method", "PATCH")
		req.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCORS_PreflightFromAllowedOrigin(t *testing.T) {
	r, reached := corsRouter(frontend)

	w := corsRequest(r, http.MethodOptions, frontend, true)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
	h := w.Header()
	if got := h.Get("Access-Control-Allow-Origin"); got != frontend {
		t.Errorf("Allow-Origin = %q, want %q", got, frontend)
	}
	for _, m := range []string{"GET", "POST", "PATCH", "DELETE"} {
		if !strings.Contains(h.Get("Access-Control-Allow-Methods"), m) {
			t.Errorf("Allow-Methods %q missing %s", h.Get("Access-Control-Allow-Methods"), m)
		}
	}
	for _, name := range []string{"Authorization", "Content-Type"} {
		if !strings.Contains(h.Get("Access-Control-Allow-Headers"), name) {
			t.Errorf("Allow-Headers %q missing %s", h.Get("Access-Control-Allow-Headers"), name)
		}
	}
	if h.Get("Vary") != "Origin" {
		t.Errorf("Vary = %q, want Origin", h.Get("Vary"))
	}
	if *reached {
		t.Error("preflight must not reach the handler")
	}
}

func TestCORS_PreflightFromDisallowedOrigin(t *testing.T) {
	r, reached := corsRouter(frontend)

	w := corsRequest(r, http.MethodOptions, "http://evil.example", true)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", w.Code)
	}
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("disallowed origin must not get Allow-Origin")
	}
	if *reached {
		t.Error("preflight must not reach the handler")
	}
}

func TestCORS_ActualRequest(t *testing.T) {
	tests := []struct {
		name      string
		origins   []string
		origin    string
		wantAllow string
		wantVary  bool
	}{
		{"allowed origin", []string{frontend}, frontend, frontend, true},
		{"second allowed origin", []string{"https://a.example", frontend}, frontend, frontend, true},
		{"disallowed origin", []string{frontend}, "http://evil.example", "", true},
		{"origin differs by port", []string{frontend}, "http://localhost:3000", "", true},
		{"no origins configured", nil, frontend, "", true},
		{"no Origin header", []string{frontend}, "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, reached := corsRouter(tt.origins...)
			w := corsRequest(r, http.MethodGet, tt.origin, false)

			// CORS is enforced by the browser, so the request itself is still served.
			if w.Code != http.StatusOK || !*reached {
				t.Errorf("status = %d, reached = %v; want 200 and handler run", w.Code, *reached)
			}
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllow {
				t.Errorf("Allow-Origin = %q, want %q", got, tt.wantAllow)
			}
			if got := w.Header().Get("Vary") == "Origin"; got != tt.wantVary {
				t.Errorf("Vary set = %v, want %v", got, tt.wantVary)
			}
		})
	}
}

func TestCORS_WildcardNotSupported(t *testing.T) {
	r, _ := corsRouter("*")

	w := corsRequest(r, http.MethodGet, frontend, false)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error(`"*" must not act as a wildcard`)
	}
}

func TestParseOrigins(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"", []string{}},
		{"http://localhost:5173", []string{"http://localhost:5173"}},
		{" http://a.example , https://b.example/ ,, ", []string{"http://a.example", "https://b.example"}},
	}
	for _, tt := range tests {
		got := middleware.ParseOrigins(tt.raw)
		if len(got) != len(tt.want) {
			t.Errorf("ParseOrigins(%q) = %v, want %v", tt.raw, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("ParseOrigins(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		}
	}
}

func TestRequireAuth_AcceptsVerifiedToken(t *testing.T) {
	var gotToken string
	restore := middleware.SetTokenVerifier(func(ctx context.Context, idToken string) (*auth.Token, error) {
		gotToken = idToken
		return &auth.Token{UID: "user-123"}, nil
	})
	defer restore()

	var gotUID string
	r := gin.New()
	r.Use(middleware.RequireAuth())
	r.GET("/", func(c *gin.Context) {
		gotUID = c.GetString(middleware.ContextUIDKey)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "bearer the-token") // scheme is case-insensitive
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if gotToken != "the-token" || gotUID != "user-123" {
		t.Errorf("verifier got %q, handler saw uid %q; want the-token / user-123", gotToken, gotUID)
	}
}

func TestRequireAuth_RejectsTokenTheVerifierRejects(t *testing.T) {
	restore := middleware.SetTokenVerifier(func(ctx context.Context, idToken string) (*auth.Token, error) {
		return nil, errors.New("verifier says no, with internal detail")
	})
	defer restore()

	w, reached := do(t, "Bearer bad-token", middleware.RequireAuth())

	if w.Code != http.StatusUnauthorized || reached {
		t.Errorf("status = %d, reached = %v; want 401 and handler not run", w.Code, reached)
	}
	if strings.Contains(w.Body.String(), "internal detail") {
		t.Errorf("response leaks the verifier's error: %s", w.Body.String())
	}
}

func TestMaxBodySize(t *testing.T) {
	const limit = 100

	newRouter := func() (*gin.Engine, *error) {
		var readErr error
		r := gin.New()
		r.Use(middleware.MaxBodySize(limit))
		r.POST("/", func(c *gin.Context) {
			_, readErr = io.ReadAll(c.Request.Body)
			c.Status(http.StatusOK)
		})
		return r, &readErr
	}

	t.Run("under the limit", func(t *testing.T) {
		r, readErr := newRouter()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", limit))))
		if w.Code != http.StatusOK || *readErr != nil {
			t.Errorf("status = %d, read error = %v; want 200 and none", w.Code, *readErr)
		}
	})

	t.Run("declared length over the limit gets 413 without running the handler", func(t *testing.T) {
		r, readErr := newRouter()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("a", limit+1)))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusRequestEntityTooLarge {
			t.Errorf("status = %d, want 413", w.Code)
		}
		if *readErr != nil {
			t.Error("handler should not have run")
		}
	})

	t.Run("undeclared length cannot be read past the limit", func(t *testing.T) {
		r, readErr := newRouter()
		req := httptest.NewRequest(http.MethodPost, "/", io.NopCloser(strings.NewReader(strings.Repeat("a", limit*10))))
		req.ContentLength = -1 // chunked: no Content-Length to check up front
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if *readErr == nil {
			t.Error("reading a chunked body past the limit should fail")
		}
	})
}

func TestRateLimit(t *testing.T) {
	newRouter := func(limit rate.Limit, burst int) *gin.Engine {
		r := gin.New()
		r.Use(middleware.RateLimit(limit, burst))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
		return r
	}
	get := func(r *gin.Engine, ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip + ":4000"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	t.Run("allows the burst then returns 429 with Retry-After", func(t *testing.T) {
		r := newRouter(rate.Every(time.Hour), 3)
		for i := 1; i <= 3; i++ {
			if w := get(r, "10.0.0.1"); w.Code != http.StatusOK {
				t.Fatalf("request %d: status = %d, want 200", i, w.Code)
			}
		}
		w := get(r, "10.0.0.1")
		if w.Code != http.StatusTooManyRequests {
			t.Fatalf("status = %d, want 429", w.Code)
		}
		if secs, err := strconv.Atoi(w.Header().Get("Retry-After")); err != nil || secs < 1 {
			t.Errorf("Retry-After = %q, want a positive number of seconds", w.Header().Get("Retry-After"))
		}
	})

	t.Run("limits each client IP separately", func(t *testing.T) {
		r := newRouter(rate.Every(time.Hour), 1)
		get(r, "10.0.0.1")
		if w := get(r, "10.0.0.1"); w.Code != http.StatusTooManyRequests {
			t.Errorf("same IP: status = %d, want 429", w.Code)
		}
		if w := get(r, "10.0.0.2"); w.Code != http.StatusOK {
			t.Errorf("other IP: status = %d, want 200", w.Code)
		}
	})

	t.Run("refills over time", func(t *testing.T) {
		r := newRouter(rate.Limit(1000), 1) // one token per millisecond
		get(r, "10.0.0.1")
		if w := get(r, "10.0.0.1"); w.Code != http.StatusTooManyRequests {
			t.Fatalf("immediate retry: status = %d, want 429", w.Code)
		}
		time.Sleep(20 * time.Millisecond)
		if w := get(r, "10.0.0.1"); w.Code != http.StatusOK {
			t.Errorf("after refill: status = %d, want 200", w.Code)
		}
	})

	t.Run("rejected requests do not extend the lockout", func(t *testing.T) {
		r := newRouter(rate.Limit(50), 1) // one token per 20ms
		get(r, "10.0.0.1")
		for i := 0; i < 20; i++ { // hammer while limited
			get(r, "10.0.0.1")
		}
		time.Sleep(60 * time.Millisecond)
		if w := get(r, "10.0.0.1"); w.Code != http.StatusOK {
			t.Errorf("status = %d, want 200 once the bucket has refilled", w.Code)
		}
	})

	t.Run("a spoofed X-Forwarded-For does not get a fresh bucket when no proxy is trusted", func(t *testing.T) {
		r := gin.New()
		// what main.go does when TRUSTED_PROXIES is unset
		if err := r.SetTrustedProxies([]string{}); err != nil {
			t.Fatal(err)
		}
		r.Use(middleware.RateLimit(rate.Every(time.Hour), 1))
		r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })

		send := func(spoof string) int {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = "10.0.0.1:4000"
			req.Header.Set("X-Forwarded-For", spoof)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			return w.Code
		}
		if code := send("1.1.1.1"); code != http.StatusOK {
			t.Fatalf("first request: status = %d, want 200", code)
		}
		if code := send("2.2.2.2"); code != http.StatusTooManyRequests {
			t.Errorf("spoofed header: status = %d, want 429 (same real client)", code)
		}
	})
}
