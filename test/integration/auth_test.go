package integration

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/addozhang/nexus-cli/internal/auth"
)

// newConfigHome points XDG_CONFIG_HOME at a fresh temp dir and returns it.
func newConfigHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return dir
}

func TestAuthAddVerifiesBeforeSaving(t *testing.T) {
	f := newFixtureServer(t)
	dir := newConfigHome(t)

	stdin := strings.NewReader(f.URL + "\nuser\nsecret\n")
	out, err := runNXIn(t, stdin, "auth", "add", "prod")
	if err != nil {
		t.Fatalf("auth add: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte(`saved`)) || !bytes.Contains([]byte(out), []byte(`default`)) {
		t.Errorf("unexpected output: %s", out)
	}

	store, err := auth.Load(filepath.Join(dir, "nx", "credentials"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	inst, ok := store.Get("prod")
	if !ok || inst.Token != "secret" || inst.Username != "user" {
		t.Fatalf("stored instance wrong: %+v (ok=%v)", inst, ok)
	}
}

func TestAuthAddUnreachableDoesNotSave(t *testing.T) {
	newConfigHome(t)

	stdin := strings.NewReader("http://127.0.0.1:1\nuser\nsecret\n")
	_, err := runNXIn(t, stdin, "auth", "add", "bad")
	if code := exitCodeOf(t, err); code != 10 {
		t.Fatalf("unreachable add should exit 10, got %d (%v)", code, err)
	}
}

func TestAuthListHidesTokens(t *testing.T) {
	f := newFixtureServer(t)
	seedInstance(t, f.URL)

	out, err := runNX(t, "auth", "list")
	if err != nil {
		t.Fatalf("auth list: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte("test:")) {
		t.Errorf("alias missing from list: %s", out)
	}
	if bytes.Contains([]byte(out), []byte("as user token")) {
		t.Errorf("token leaked in auth list output: %s", out)
	}
}

func TestAuthRemoveAndDefaultLifecycle(t *testing.T) {
	f := newFixtureServer(t)
	dir := newConfigHome(t)

	path := filepath.Join(dir, "nx", "credentials")
	store, err := auth.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Add("a", f.URL, "u", "t", true, false)
	_ = store.Add("b", f.URL, "u", "t", false, false)

	_, err = runNX(t, "repo", "list") // default = a; server reachable
	if err != nil {
		t.Fatalf("default resolution should work: %v", err)
	}

	if _, err := runNX(t, "auth", "remove", "a"); err != nil {
		t.Fatalf("auth remove: %v", err)
	}
	_, err = runNX(t, "repo", "list")
	if code := exitCodeOf(t, err); code != 10 {
		t.Errorf("after removing default, flag-less query should fail with 10, got %d", code)
	}

	if _, err := runNX(t, "auth", "default", "b"); err != nil {
		t.Fatalf("auth default: %v", err)
	}
	if _, err := runNX(t, "repo", "list"); err != nil {
		t.Errorf("query after setting default b should succeed: %v", err)
	}

	_, err = runNX(t, "auth", "remove", "ghost")
	if code := exitCodeOf(t, err); code != 10 {
		t.Errorf("removing unknown alias should exit 10, got %d", code)
	}
}
