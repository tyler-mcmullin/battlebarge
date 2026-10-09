package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// All files here are throwaway fakes in a temp directory; the project's real
// .env is never touched.

const testVar = "BATTLEBARGE_TEST_ENVFILE_VALUE"

// clearTestVar removes testVar from the environment now and restores whatever
// it was when the test ends (godotenv sets variables for the whole process).
func clearTestVar(t *testing.T) {
	t.Helper()
	t.Setenv(testVar, "")
	os.Unsetenv(testVar)
}

func writeEnvFile(t *testing.T, dir, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadEnvFile_FindsTheFileInTheWorkingDirectoryOrItsParent(t *testing.T) {
	// the real layout: backend/.env, running from backend/ (same directory)
	// or from backend/cmd/ (parent)
	for name, depth := range map[string]int{"same directory": 0, "parent": 1} {
		t.Run(name, func(t *testing.T) {
			clearTestVar(t)
			root := t.TempDir()
			writeEnvFile(t, root, testVar+"=from-file\n")

			dir := root
			for i := 0; i < depth; i++ {
				dir = filepath.Join(dir, "sub")
			}
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			t.Chdir(dir)

			loadEnvFile()

			if got := os.Getenv(testVar); got != "from-file" {
				t.Errorf("%s = %q, want the value from the file", testVar, got)
			}
		})
	}
}

// TestLoadEnvFile_DoesNotLookAboveTheParent checks that a file above the parent
// directory (the repo root) is not picked up, so a root level file meant for
// something else never leaks into the backend.
func TestLoadEnvFile_DoesNotLookAboveTheParent(t *testing.T) {
	clearTestVar(t)
	root := t.TempDir()
	writeEnvFile(t, root, testVar+"=from-a-file-two-levels-up\n")
	dir := filepath.Join(root, "backend", "cmd")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	loadEnvFile()

	if got := os.Getenv(testVar); got != "" {
		t.Errorf("%s = %q; a file two levels up must not be loaded", testVar, got)
	}
}

func TestLoadEnvFile_RealEnvironmentWins(t *testing.T) {
	root := t.TempDir()
	writeEnvFile(t, root, testVar+"=from-file\n")
	t.Chdir(root)
	t.Setenv(testVar, "from-real-environment")

	loadEnvFile()

	if got := os.Getenv(testVar); got != "from-real-environment" {
		t.Errorf("%s = %q; a variable that is already set must beat the file", testVar, got)
	}
}

func TestLoadEnvFile_MissingFileIsFine(t *testing.T) {
	clearTestVar(t)
	t.Chdir(t.TempDir())

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("a missing file must not stop startup, got panic: %v", r)
		}
	}()
	loadEnvFile()

	if got := os.Getenv(testVar); got != "" {
		t.Errorf("%s = %q, want it unset", testVar, got)
	}
}

func TestLoadEnvFile_MalformedFileStopsStartup(t *testing.T) {
	clearTestVar(t)
	root := t.TempDir()
	writeEnvFile(t, root, "THIS LINE HAS NO EQUALS SIGN AND AN UNCLOSED QUOTE \"\n")
	t.Chdir(root)

	defer func() {
		if recover() == nil {
			t.Error("a malformed file should stop startup instead of being ignored")
		}
	}()
	loadEnvFile()
}

func TestSplitList(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"  ", nil},
		{",,", nil},
		{"a", []string{"a"}},
		{"a, b ,,c", []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		got := splitList(tt.in)
		if len(got) != len(tt.want) {
			t.Errorf("splitList(%q) = %#v, want %#v", tt.in, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("splitList(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
		}
	}
}

// TestTrustedProxiesSetting checks how gin treats splitList's result: with no
// TRUSTED_PROXIES the X-Forwarded-For header must be ignored, and with one set
// it must be believed, but only from that proxy.
func TestTrustedProxiesSetting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	clientIP := func(trustedProxies, remoteAddr string) string {
		r := gin.New()
		if err := r.SetTrustedProxies(splitList(trustedProxies)); err != nil {
			t.Fatal(err)
		}
		var got string
		r.GET("/", func(c *gin.Context) { got = c.ClientIP() })

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = remoteAddr
		req.Header.Set("X-Forwarded-For", "203.0.113.9")
		r.ServeHTTP(httptest.NewRecorder(), req)
		return got
	}

	tests := []struct {
		name, trusted, remote, want string
	}{
		{"unset: the header is ignored", "", "10.0.0.1:4000", "10.0.0.1"},
		{"blank: the header is ignored", " , ", "10.0.0.1:4000", "10.0.0.1"},
		{"set, request from that proxy: the header is believed", "10.0.0.1", "10.0.0.1:4000", "203.0.113.9"},
		{"set as a range: believed from inside it", "10.0.0.0/24", "10.0.0.7:4000", "203.0.113.9"},
		{"set, request from anyone else: the header is ignored", "10.0.0.1", "198.51.100.5:4000", "198.51.100.5"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := clientIP(tt.trusted, tt.remote); got != tt.want {
				t.Errorf("ClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
