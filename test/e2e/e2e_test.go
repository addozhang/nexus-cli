//go:build e2e

// Package e2e runs the nx command tree against a real Nexus Repository
// container provisioned by `make e2e-up` (test/e2e/up.sh). It validates the
// wire-format assumptions the httptest fixtures cannot: embedded search
// assets, timestamp formats, the exposed field, and per-format search
// behavior.
//
// Run with: make e2e-test   (requires the container from make e2e-up)
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/addozhang/nexus-cli/internal/auth"
	"github.com/addozhang/nexus-cli/internal/cli"
)

const (
	envURL  = "NX_E2E_URL"
	envUser = "NX_E2E_USER"
	envPass = "NX_E2E_PASS"

	defaultPass   = "nx-e2e-Pass1!"
	instanceAlias = "e2e"
)

var configHome string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "nx-e2e-config")
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e setup: %v\n", err)
		os.Exit(1)
	}
	configHome = dir
	os.Setenv("XDG_CONFIG_HOME", dir)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// requireInstance skips unless a Nexus is reachable and registers it under
// the shared alias once.
func requireInstance(t *testing.T) string {
	t.Helper()
	url := os.Getenv(envURL)
	if url == "" {
		t.Skipf("set %s to run e2e tests (make e2e-up && make e2e-test)", envURL)
	}
	user := os.Getenv(envUser)
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv(envPass)
	if pass == "" {
		pass = defaultPass
	}

	path := filepath.Join(configHome, "nx", "credentials")
	store, err := auth.Load(path)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if _, ok := store.Get(instanceAlias); !ok {
		if err := store.Add(instanceAlias, url, user, pass, true); err != nil {
			t.Fatalf("register instance: %v", err)
		}
	}
	return url
}

func runNX(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	root := cli.NewRootCommand()
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func mustRunJSON(t *testing.T, args ...string) map[string]any {
	t.Helper()
	out, err := runNX(t, append(append([]string{}, args...), "-o", "json")...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("%v: output is not valid JSON: %v\n%s", args, err, out)
	}
	if parsed["schemaVersion"] != "1" {
		t.Errorf("%v: schemaVersion = %v, want \"1\"", args, parsed["schemaVersion"])
	}
	return parsed
}

// --- repo list ---

func TestE2ERepoListAllFormats(t *testing.T) {
	requireInstance(t)

	parsed := mustRunJSON(t, "repo", "list")
	repos, _ := parsed["repositories"].([]any)
	if len(repos) == 0 {
		t.Fatal("expected seeded repositories, got empty list")
	}

	names := map[string]bool{}
	for _, r := range repos {
		entry, _ := r.(map[string]any)
		name, _ := entry["name"].(string)
		names[name] = true

		// Wire-assumption check: exposed must be present and non-nil on
		// real instances; fixtures invented this shape.
		if _, ok := entry["exposed"]; !ok {
			t.Errorf("repository %q: exposed field missing from real API response", name)
		}
		for _, stable := range []string{"format", "type", "url"} {
			if v, ok := entry[stable]; !ok || v == nil || v == "" {
				t.Errorf("repository %q: stable field %q missing/empty (%v)", name, stable, entry[stable])
			}
		}
	}
	for _, want := range []string{"e2e-maven", "e2e-npm", "e2e-pypi"} {
		if !names[want] {
			t.Errorf("expected repository %q in listing, got %v", want, names)
		}
	}
}

func TestE2ERepoListFilters(t *testing.T) {
	requireInstance(t)

	parsed := mustRunJSON(t, "repo", "list", "--format", "maven", "--type", "hosted")
	repos, _ := parsed["repositories"].([]any)
	// The instance also ships built-in maven hosted repos (maven-releases,
	// maven-snapshots); e2e-maven must be among the matches.
	found := false
	for _, r := range repos {
		entry, _ := r.(map[string]any)
		if entry["name"] == "e2e-maven" {
			found = true
		}
	}
	if !found {
		t.Errorf("e2e-maven missing from filtered listing: %v", repos)
	}
}

// --- maven (seeded with data) ---

func TestE2EMavenSearchFindsSeededComponent(t *testing.T) {
	requireInstance(t)

	parsed := mustRunJSON(t, "maven", "search", "com.example:e2e")
	results, _ := parsed["results"].([]any)
	if len(results) < 2 {
		t.Fatalf("expected >=2 versions of com.example:e2e, got %d:\n%v", len(results), results)
	}
	first, _ := results[0].(map[string]any)
	if first["name"] != "e2e" || first["group"] != "com.example" {
		t.Errorf("search result wrong: %v", first)
	}
	if _, ok := first["repository"]; !ok {
		t.Error("repository field missing from search result")
	}
}

func TestE2EMavenVersionsSortedNewestFirst(t *testing.T) {
	requireInstance(t)

	parsed := mustRunJSON(t, "maven", "versions", "com.example:e2e")
	versions, _ := parsed["versions"].([]any)
	if len(versions) < 2 {
		t.Fatalf("expected >=2 versions, got %v", versions)
	}
	first, _ := versions[0].(map[string]any)
	if first["version"] != "1.1.0" {
		t.Errorf("newest-first ordering violated: first version is %v", first["version"])
	}

	// Wire-assumption check: lastModified must be populated from real asset
	// timestamps, proving the non-RFC3339 "+0000" layout parses.
	entry, _ := versions[len(versions)-1].(map[string]any)
	if lm, ok := entry["lastModified"].(string); !ok || lm == "" {
		t.Errorf("lastModified should parse from real instance, got %v", entry["lastModified"])
	}
}

func TestE2EMavenInfoAssetsAndChecksums(t *testing.T) {
	requireInstance(t)

	raw, err := runNX(t, "maven", "info", "com.example:e2e:1.0.0", "-o", "json")
	if err != nil {
		t.Fatalf("info: %v\n%s", err, raw)
	}
	var parsed struct {
		SchemaVersion string `json:"schemaVersion"`
		Components    []struct {
			Name       string `json:"name"`
			Version    string `json:"version"`
			Repository string `json:"repository"`
			Assets     []struct {
				Path           string            `json:"path"`
				Size           *int64            `json:"size"`
				Checksums      map[string]string `json:"checksums"`
				Uploader       string            `json:"uploader"`
				BlobCreated    *string           `json:"blobCreated"`
				LastModified   *string           `json:"lastModified"`
				LastDownloaded *string           `json:"lastDownloaded"`
			} `json:"assets"`
		} `json:"components"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("bad JSON: %v\n%s", err, raw)
	}
	if len(parsed.Components) == 0 {
		t.Fatalf("no components returned:\n%s", raw)
	}
	c := parsed.Components[0]
	if c.Version != "1.0.0" || c.Repository != "e2e-maven" {
		t.Errorf("component wrong: %+v", c)
	}

	// Wire-assumption checks on assets.
	var sawJar bool
	for _, a := range c.Assets {
		if strings.HasSuffix(a.Path, ".jar") {
			sawJar = true
			if a.Size == nil || *a.Size == 0 {
				t.Errorf("jar asset size missing: %+v", a)
			}
			if a.Checksums == nil || a.Checksums["sha1"] == "" {
				t.Errorf("jar checksums missing: %+v", a)
			}
			// lastDownloaded may legitimately become non-null on a real
			// instance (any GET records a download); assert the key exists.
			if !strings.Contains(raw, `"lastDownloaded"`) {
				t.Error(`stable key "lastDownloaded" missing from asset JSON`)
			}
		}
	}

	// Wire-assumption check: real instances expose upload provenance.
	for _, a := range c.Assets {
		if a.Uploader == "" {
			t.Errorf("uploader missing on asset %q", a.Path)
		}
		if a.BlobCreated == nil || *a.BlobCreated == "" {
			t.Errorf("blobCreated missing on asset %q", a.Path)
		}
	}
	// Wire-assumption check: real instances expose upload provenance.
	for _, a := range c.Assets {
		if a.Uploader == "" {
			t.Errorf("uploader missing on asset %q", a.Path)
		}
		if a.BlobCreated == nil || *a.BlobCreated == "" {
			t.Errorf("blobCreated missing on asset %q", a.Path)
		}
	}
	if !sawJar {
		t.Errorf("jar asset not found among %+v", c.Assets)
	}
}

func TestE2EMavenInfoMissingVersionRejected(t *testing.T) {
	requireInstance(t)

	_, err := runNX(t, "maven", "info", "com.example:e2e")
	if code := exitCode(err); code != 10 {
		t.Errorf("info without version should exit 10, got %d (%v)", code, err)
	}
	_, err = runNX(t, "maven", "info", "com.example:ghost:9.9.9")
	if code := exitCode(err); code != 10 {
		t.Errorf("absent coordinate should exit 10, got %d (%v)", code, err)
	}
}

// --- other formats: endpoint compatibility + empty-result semantics ---

func TestE2EOtherFormatSearchesAreCompatible(t *testing.T) {
	requireInstance(t)

	for _, tc := range []struct{ format, query string }{
		{"npm", "nothing-matches-this-e2e-query"},
		{"pypi", "nothing-matches-this-e2e-query"},
		{"cargo", "nothing-matches-this-e2e-query"},
		{"go", "nothing-matches-this-e2e-query"},
		{"docker", "nothing-matches-this-e2e-query"},
	} {
		t.Run(tc.format, func(t *testing.T) {
			raw, err := runNX(t, tc.format, "search", tc.query, "-o", "json")
			if err != nil {
				t.Fatalf("%s search against real instance failed: %v\n%s", tc.format, err, raw)
			}
			var parsed struct {
				SchemaVersion string           `json:"schemaVersion"`
				Results       []map[string]any `json:"results"`
			}
			if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
				t.Fatalf("bad JSON from %s search: %v\n%s", tc.format, err, raw)
			}
			if len(parsed.Results) != 0 {
				t.Errorf("expected zero matches for %q, got %d", tc.query, len(parsed.Results))
			}
		})
	}
}

func TestE2ELimitFlagAgainstRealPagination(t *testing.T) {
	requireInstance(t)

	parsed := mustRunJSON(t, "maven", "search", "com.example:e2e", "--limit", "1")
	results, _ := parsed["results"].([]any)
	if len(results) != 1 {
		t.Errorf("--limit 1 returned %d results", len(results))
	}
}

// --- auth lifecycle ---

func TestE2EAuthListAndResolutionErrors(t *testing.T) {
	requireInstance(t)

	out, err := runNX(t, "auth", "list")
	if err != nil {
		t.Fatalf("auth list: %v", err)
	}
	if !strings.Contains(out, instanceAlias+":") {
		t.Errorf("alias missing from auth list:\n%s", out)
	}
	if strings.Contains(out, os.Getenv(envPass)) {
		t.Error("token material leaked into auth list output")
	}

	_, err = runNX(t, "repo", "list", "--instance", "no-such-alias")
	if code := exitCode(err); code != 10 {
		t.Errorf("unknown alias should exit 10, got %d", code)
	}
}

func TestE2EBadCredentialsExitTen(t *testing.T) {
	requireInstance(t)
	url := os.Getenv(envURL)

	t.Setenv(auth.EnvURL, url)
	t.Setenv(auth.EnvUsername, "admin")
	t.Setenv(auth.EnvToken, "wrong-password")

	_, err := runNX(t, "repo", "list")
	if code := exitCode(err); code != 10 {
		t.Errorf("bad credentials should exit 10, got %d (%v)", code, err)
	}
}

// --- env override path ---

func TestE2EEnvOverrideWorks(t *testing.T) {
	requireInstance(t)
	url := os.Getenv(envURL)
	user := cmpOr(os.Getenv(envUser), "admin")
	pass := cmpOr(os.Getenv(envPass), defaultPass)

	t.Setenv(auth.EnvURL, url)
	t.Setenv(auth.EnvUsername, user)
	t.Setenv(auth.EnvToken, pass)
	// Point XDG_CONFIG_HOME at an empty dir so no stored default exists;
	// resolution must come entirely from the environment.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	_, err := runNX(t, "repo", "list", "-o", "json")
	if err != nil {
		t.Fatalf("env override should work without stored instances: %v", err)
	}
}

func cmpOr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	return cli.ExitCode(err)
}
