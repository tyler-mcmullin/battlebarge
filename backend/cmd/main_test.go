package main

import (
	"os"
	"path/filepath"
	"testing"
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

// A file at the repo root (above backend/) must not be picked up, so a root
// level file meant for something else never leaks into the backend.
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
	for in, want := range map[string]int{"": 0, "  ": 0, "a": 1, "a, b ,,c": 3} {
		got := splitList(in)
		if got == nil || len(got) != want {
			t.Errorf("splitList(%q) = %#v, want %d non-nil entries", in, got, want)
		}
	}
}
