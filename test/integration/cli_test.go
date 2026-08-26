// Package integration runs the full command tree against an httptest.Server
// posing as a Nexus instance, verifying spec scenarios end to end (in
// process, no binary build required).
package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"path/filepath"
	"strings"
	"testing"

	"github.com/addozhang/nexus-cli/internal/auth"
	"github.com/addozhang/nexus-cli/internal/cli"
	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// fixtureServer is a minimal fake Nexus covering the endpoints nx reads.
type fixtureServer struct {
	*httptest.Server
	repos      []map[string]any
	searchPage func(r *http.Request) (items []map[string]any, token string)
}

func newFixtureServer(t *testing.T) *fixtureServer {
	t.Helper()
	f := &fixtureServer{}
	mux := http.NewServeMux()

	mux.HandleFunc("/service/rest/v1/status", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/service/rest/v1/repositories", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, f.repos)
	})
	mux.HandleFunc("/service/rest/v1/search", func(w http.ResponseWriter, r *http.Request) {
		items, token := f.searchPage(r)
		resp := map[string]any{"items": items}
		if token != "" {
			resp["continuationToken"] = token
		}
		writeJSON(w, resp)
	})

	f.Server = httptest.NewServer(mux)
	t.Cleanup(f.Close)
	return f
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// seedInstance registers one instance against the running server and marks it
// default, bypassing the interactive auth add flow.
func seedInstance(t *testing.T, url string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nx", "credentials")
	t.Setenv("XDG_CONFIG_HOME", filepath.Dir(filepath.Dir(path)))
	store, err := auth.Load(path)
	if err != nil {
		t.Fatalf("seed store: %v", err)
	}
	if err := store.Add("test", url, "user", "token", true); err != nil {
		t.Fatalf("seed add: %v", err)
	}
}

func runNX(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runNXIn(t, strings.NewReader(""), args...)
}

func runNXIn(t *testing.T, stdin *strings.Reader, args ...string) (string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	root := cli.NewRootCommand()
	root.SetOut(&out)
	root.SetErr(&errOut)
	root.SetIn(stdin)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), err
}

func exitCodeOf(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	return cli.ExitCode(err)
}

func TestRepoList(t *testing.T) {
	f := newFixtureServer(t)
	f.repos = []map[string]any{
		{"name": "maven-central", "format": "maven", "type": "proxy", "url": "http://x/maven-central", "exposed": true},
		{"name": "npm-internal", "format": "npm", "type": "hosted", "url": "http://x/npm-internal", "exposed": true},
	}
	seedInstance(t, f.URL)

	out, err := runNX(t, "repo", "list")
	if err != nil {
		t.Fatalf("repo list: %v", err)
	}
	if !bytes.Contains([]byte(out), []byte(`schemaVersion: "1"`)) {
		t.Errorf("output missing schemaVersion first field:\n%s", out[:min(200, len(out))])
	}
	for _, want := range []string{"maven-central", "npm-internal"} {
		if !bytes.Contains([]byte(out), []byte(want)) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	outJSON, err := runNX(t, "repo", "list", "-o", "json")
	if err != nil {
		t.Fatalf("repo list -o json: %v", err)
	}
	var parsed struct {
		SchemaVersion string `json:"schemaVersion"`
		Repositories  []struct {
			Name    string `json:"name"`
			Format  string `json:"format"`
			Type    string `json:"type"`
			URL     string `json:"url"`
			Exposed *bool  `json:"exposed"`
		} `json:"repositories"`
	}
	if err := json.Unmarshal([]byte(outJSON), &parsed); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, outJSON)
	}
	if parsed.SchemaVersion != "1" || len(parsed.Repositories) != 2 || parsed.Repositories[0].Exposed == nil {
		t.Errorf("unexpected JSON payload: %s", outJSON)
	}

	out, _ = runNX(t, "repo", "list", "--format", "npm")
	if bytes.Contains([]byte(out), []byte("maven-central")) || !bytes.Contains([]byte(out), []byte("npm-internal")) {
		t.Errorf("format filter failed:\n%s", out)
	}

	out, _ = runNX(t, "repo", "list", "--format", "docker", "--type", "proxy")
	if bytes.Contains([]byte(out), []byte("name:")) {
		t.Errorf("empty filter result should render empty list:\n%s", out)
	}

	_, err = runNX(t, "repo", "list", "--type", "bogus")
	if code := exitCodeOf(t, err); code != 10 {
		t.Errorf("invalid type should exit 10, got %d (%v)", code, err)
	}
}

func component(id, name, version, repo string, assets bool) map[string]any {
	c := map[string]any{
		"id": id, "group": "", "name": name, "version": version,
		"repository": repo, "format": "npm",
	}
	if assets {
		c["assets"] = []map[string]any{{
			"path":           "/" + name + "/-/" + name + "-" + version + ".tgz",
			"size":           1234,
			"checksums":      map[string]any{"sha1": "abc", "sha256": "def"},
			"lastModified":   "2026-01-02T03:04:05.000+0000",
			"lastDownloaded": nil,
		}}
	}
	return c
}

func TestSearchPaginationAndLimit(t *testing.T) {
	f := newFixtureServer(t)
	page2Seen := false
	f.searchPage = func(r *http.Request) ([]map[string]any, string) {
		q := r.URL.Query()
		all := []map[string]any{
			component("1", "lodash", "4.17.21", "npm-internal", false),
			component("2", "lodash", "4.17.20", "npm-internal", false),
			component("3", "lodash", "4.17.15", "npm-internal", false),
			component("4", "other", "1.0.0", "npm-internal", false),
		}
		var matched []map[string]any
		for _, c := range all {
			if q.Get("name") != "" && c["name"] != q.Get("name") {
				continue
			}
			if q.Get("version") != "" && c["version"] != q.Get("version") {
				continue
			}
			matched = append(matched, c)
		}
		if len(matched) == 0 {
			return nil, ""
		}
		if q.Get("continuationToken") == "" && len(matched) > 2 {
			return matched[:2], "tok2"
		}
		if q.Get("continuationToken") == "tok2" {
			page2Seen = true
			return matched[2:], ""
		}
		return matched, ""
	}
	seedInstance(t, f.URL)

	out, err := runNX(t, "npm", "search", "lodash@4.17.20")
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if page2Seen {
		t.Error("single-match search must not follow continuation")
	}
	if !bytes.Contains([]byte(out), []byte("4.17.20")) || bytes.Contains([]byte(out), []byte("- id: \"1\"")) {
		t.Errorf("version-constrained search wrong:\n%s", out)
	}

	out, err = runNX(t, "npm", "search", "lodash")
	if err != nil || !page2Seen {
		t.Errorf("free search should follow pagination; err=%v followed=%v", err, page2Seen)
	}
	if !bytes.Contains([]byte(out), []byte("4.17.15")) {
		t.Errorf("page 2 items missing from output:\n%s", out)
	}

	out, err = runNX(t, "npm", "search", "lodash", "--limit", "2")
	if err != nil {
		t.Fatalf("limited search: %v", err)
	}
	if got := bytes.Count([]byte(out), []byte("- id:")); got != 2 {
		t.Errorf("limit 2 returned %d entries:\n%s", got, out)
	}
}

func TestVersionsAggregation(t *testing.T) {
	f := newFixtureServer(t)
	f.searchPage = func(_ *http.Request) ([]map[string]any, string) {
		return []map[string]any{
			component("1", "left-pad", "1.3.0", "npm-internal", true),
			component("2", "left-pad", "1.3.0", "npm-mirror", false),
			component("3", "left-pad", "1.2.0", "npm-internal", false),
		}, ""
	}
	seedInstance(t, f.URL)

	out, err := runNX(t, "npm", "versions", "left-pad")
	if err != nil {
		t.Fatalf("versions: %v", err)
	}
	if got := bytes.Count([]byte(out), []byte("- version:")); got != 2 {
		t.Errorf("expected 2 distinct versions sorted newest first, got %d:\n%s", got, out)
	}
	idx130 := bytes.Index([]byte(out), []byte("version: 1.3.0"))
	idx120 := bytes.Index([]byte(out), []byte("version: 1.2.0"))
	if idx130 > idx120 {
		t.Errorf("versions not sorted newest first:\n%s", out)
	}
	if !bytes.Contains([]byte(out), []byte("2026-01-02T03:04:05Z")) && !bytes.Contains([]byte(out), []byte("2026-01-02T03:04:05")) {
		t.Errorf("lastModified missing:\n%s", out)
	}
}

func TestInfoDetailAndNotFound(t *testing.T) {
	f := newFixtureServer(t)
	f.searchPage = func(r *http.Request) ([]map[string]any, string) {
		q := r.URL.Query()
		if q.Get("version") == "4.17.21" && q.Get("format") == "npm" {
			return []map[string]any{component("c1", "lodash", "4.17.21", "npm-internal", true)}, ""
		}
		return []map[string]any{}, ""
	}
	seedInstance(t, f.URL)

	out, err := runNX(t, "npm", "info", "lodash@4.17.21", "-o", "json")
	if err != nil {
		t.Fatalf("info: %v", err)
	}
	var parsed struct {
		SchemaVersion string `json:"schemaVersion"`
		Components    []struct {
			Assets []struct {
				Path           string            `json:"path"`
				Size           *int64            `json:"size"`
				Checksums      map[string]string `json:"checksums"`
				LastDownloaded any               `json:"lastDownloaded"`
			} `json:"assets"`
		} `json:"components"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("bad JSON: %v\n%s", err, out)
	}
	if parsed.SchemaVersion != "1" || len(parsed.Components) != 1 || len(parsed.Components[0].Assets) != 1 {
		t.Fatalf("unexpected info payload: %s", out)
	}
	asset := parsed.Components[0].Assets[0]
	if asset.LastDownloaded != nil {
		t.Errorf("never-downloaded asset must be explicit null, got %v", asset.LastDownloaded)
	}
	if asset.Checksums["sha256"] != "def" || asset.Size == nil || *asset.Size != 1234 {
		t.Errorf("asset metadata wrong: %+v", asset)
	}

	_, err = runNX(t, "npm", "info", "lodash")
	if code := exitCodeOf(t, err); code != 10 {
		t.Errorf("info without version should exit 10, got %d", code)
	}
	var nxErr *nxerrors.Error
	if !asClassError(err, &nxErr) || nxErr.Class() != nxerrors.ClassCoordinate {
		t.Errorf("missing version should be ClassCoordinate, got %v", err)
	}

	_, err = runNX(t, "npm", "info", "ghost@9.9.9")
	if code := exitCodeOf(t, err); code != 10 {
		t.Errorf("absent coordinate should exit 10, got %d", code)
	}
}

func TestInstanceResolutionErrors(t *testing.T) {
	f := newFixtureServer(t)
	// no instances stored
	_, err := runNX(t, "repo", "list")
	if code := exitCodeOf(t, err); code != 10 {
		t.Errorf("no default configured should exit 10, got %d", code)
	}

	// unknown alias
	seedInstance(t, "http://unused")
	_, err = runNX(t, "repo", "list", "--instance", "staging")
	var nxErr *nxerrors.Error
	if !asClassError(err, &nxErr) || nxErr.Class() != nxerrors.ClassInstance {
		t.Errorf("unknown alias should be ClassInstance, got %v", err)
	}

	// env override works even with no default instance
	t.Setenv(auth.EnvURL, f.URL)
	t.Setenv(auth.EnvUsername, "envuser")
	t.Setenv(auth.EnvToken, "envtoken")
	if _, err := runNX(t, "repo", "list"); err != nil {
		t.Errorf("full env override should resolve without stored instances: %v", err)
	}
}

func asClassError(err error, target **nxerrors.Error) bool {
	return errors.As(err, target)
}
