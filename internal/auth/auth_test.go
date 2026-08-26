package auth_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/addozhang/nexus-cli/internal/auth"
	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

func newTestStore(t *testing.T) *auth.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nx", "credentials")
	s, err := auth.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return s
}

func Test_Store_AddSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nx", "credentials")
	s, err := auth.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := s.Add("prod", "https://nexus.example.com:8443/", "deployer", "tok", true); err != nil {
		t.Fatalf("Add: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("credentials file missing: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}

	reloaded, err := auth.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	inst, ok := reloaded.Get("prod")
	if !ok {
		t.Fatal("alias prod not found after reload")
	}
	if inst.URL != "https://nexus.example.com:8443" {
		t.Errorf("URL = %q, want trailing slash stripped", inst.URL)
	}
	if inst.Token != "tok" || inst.Username != "deployer" {
		t.Errorf("credentials not persisted: %+v", inst)
	}
	if alias, ok := reloaded.DefaultAlias(); !ok || alias != "prod" {
		t.Errorf("DefaultAlias() = %q,%v; want prod,true", alias, ok)
	}
}

func Test_Store_SetDefaultClearsOthers(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("a", "http://a", "u", "t", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("b", "http://b", "u", "t", false); err != nil {
		t.Fatal(err)
	}
	if err := s.SetDefault("b"); err != nil {
		t.Fatalf("SetDefault: %v", err)
	}
	a, _ := s.Get("a")
	b, _ := s.Get("b")
	if a.Default || !b.Default {
		t.Errorf("default marker wrong after SetDefault: a=%v b=%v", a.Default, b.Default)
	}
}

func Test_Store_RemoveUnknownAliasFailsWithInstanceClass(t *testing.T) {
	s := newTestStore(t)
	err := s.Remove("ghost")
	var nxErr *nxerrors.Error
	if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassInstance {
		t.Fatalf("want ClassInstance error, got %v", err)
	}
}

func Test_Store_MissingFileIsEmptyStore(t *testing.T) {
	s, err := auth.Load(filepath.Join(t.TempDir(), "nope", "credentials"))
	if err != nil {
		t.Fatalf("Load on missing file: %v", err)
	}
	if len(s.Aliases()) != 0 {
		t.Errorf("expected empty store, got %v", s.Aliases())
	}
	if list := s.AliasList(); list != "(none)" {
		t.Errorf("AliasList() = %q, want \"(none)\"", list)
	}
}

func Test_Store_RemoveDefaultClearsMarker(t *testing.T) {
	s := newTestStore(t)
	_ = s.Add("dev", "http://dev", "u", "t", true)
	if err := s.Remove("dev"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, ok := s.DefaultAlias(); ok {
		t.Error("removing the default instance should leave no default")
	}
}

func Test_Resolve_ExplicitFlagWinsOverEverything(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("dev", "http://dev", "u", "t", true); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("prod", "http://prod", "u2", "t2", false); err != nil {
		t.Fatal(err)
	}

	t.Setenv(auth.EnvURL, "http://env")
	t.Setenv(auth.EnvUsername, "envuser")
	t.Setenv(auth.EnvToken, "envtoken")

	got, err := auth.Resolve(s, "prod")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.BaseURL != "http://prod" || got.Alias != "prod" {
		t.Errorf("resolved %+v, want prod", got)
	}
}

func Test_Resolve_EnvOverrideWhenNoFlag(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("dev", "http://dev", "u", "t", true); err != nil {
		t.Fatal(err)
	}

	t.Setenv(auth.EnvURL, "http://env/")
	t.Setenv(auth.EnvUsername, "envuser")
	t.Setenv(auth.EnvToken, "envtoken")

	got, err := auth.Resolve(s, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.BaseURL != "http://env" || got.Alias != "" {
		t.Errorf("resolved %+v, want env override without alias", got)
	}
}

func Test_Resolve_PartialEnvIgnored(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("dev", "http://dev", "u", "t", true); err != nil {
		t.Fatal(err)
	}
	t.Setenv(auth.EnvURL, "http://env")
	t.Setenv(auth.EnvUsername, "envuser")
	os.Unsetenv(auth.EnvToken)

	got, err := auth.Resolve(s, "")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.BaseURL != "http://dev" {
		t.Errorf("partial env should be ignored, got %q", got.BaseURL)
	}
}

func Test_Resolve_NoDefaultErrorsListingAliases(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add("a", "http://a", "u", "t", false); err != nil {
		t.Fatal(err)
	}
	_, err := auth.Resolve(s, "")
	if err == nil {
		t.Fatal("expected error with no default and no flag")
	}
	if !strings.Contains(err.Error(), "a") {
		t.Errorf("error should list known aliases, got %q", err.Error())
	}
}

func Test_Resolve_UnknownFlagAliasErrorsWithInstanceClass(t *testing.T) {
	s := newTestStore(t)
	_, err := auth.Resolve(s, "staging")
	var nxErr *nxerrors.Error
	if !errors.As(err, &nxErr) || nxErr.Class() != nxerrors.ClassInstance {
		t.Fatalf("want ClassInstance error, got %v", err)
	}
}
