package schema_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/addozhang/nexus-cli/internal/nexus"
	"github.com/addozhang/nexus-cli/internal/schema"
)

func Test_MapRepoList(t *testing.T) {
	exposed := true
	raw := []nexus.RawRepository{
		{Name: "maven-central", Format: "maven", Type: "proxy", URL: "http://x/m", Exposed: &exposed},
		{Name: "docker", Format: "docker", Type: "hosted", URL: "http://x/d"},
	}
	got := schema.MapRepoList(raw)
	if got.SchemaVersion != "1" || len(got.Repositories) != 2 {
		t.Fatalf("bad payload: %+v", got)
	}
	if got.Repositories[0].Name != "maven-central" || !*got.Repositories[0].Exposed {
		t.Errorf("entry 0 wrong: %+v", got.Repositories[0])
	}
	if got.Repositories[1].Exposed != nil {
		t.Error("absent exposed should map to nil")
	}
	b, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(b[:len(`{"schemaVersion":"1"`)]) != `{"schemaVersion":"1"` {
		t.Errorf("schemaVersion not first JSON key: %s", b)
	}
}

func Test_MapSearchResults(t *testing.T) {
	raw := []nexus.RawComponent{{ID: "c1", Group: "g", Name: "n", Version: "1", Repository: "r", Format: "maven"}}
	got := schema.MapSearchResults(raw)
	if len(got.Results) != 1 || got.Results[0].ID != "c1" || got.SchemaVersion != "1" {
		t.Fatalf("bad payload: %+v", got)
	}
}

func Test_MapVersionList_AggregatesAndSorts(t *testing.T) {
	mod := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	newer := mod.Add(time.Hour)
	raw := []nexus.RawComponent{
		{ID: "1", Name: "lib", Version: "1.9", Repository: "a"},
		{ID: "2", Name: "lib", Version: "1.10", Repository: "b",
			Assets: []nexus.RawAsset{{LastModified: (*nexus.TimeValue)(nil)}},
		},
	}
	// inject timestamps through proper pointers
	tv := nexus.TimeValue(mod)
	tv2 := nexus.TimeValue(newer)
	raw[1].Assets = []nexus.RawAsset{{LastModified: &tv}, {LastModified: &tv2}}

	got := schema.MapVersionList("", "lib", raw)
	if len(got.Versions) != 2 {
		t.Fatalf("expected dedupe to 2 versions, got %+v", got.Versions)
	}
	if got.Versions[0].Version != "1.10" || got.Versions[1].Version != "1.9" {
		t.Errorf("sort order wrong: %+v", got.Versions)
	}
	if got.Versions[0].LastModified == nil || !got.Versions[0].LastModified.Equal(newer) {
		t.Errorf("lastModified should be max of asset times: %v", got.Versions[0].LastModified)
	}
	if got.Versions[0].Repository != "b" {
		t.Errorf("repository of first occurrence wrong: %+v", got.Versions[0])
	}
	if got.Versions[1].LastModified != nil {
		t.Error("version without assets should have null lastModified")
	}
}

func Test_MapInfoResults_AssetsAndNotFound(t *testing.T) {
	size := int64(42)
	downloaded := nexus.TimeValue(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	raw := []nexus.RawComponent{{
		ID: "c1", Name: "lib", Version: "1.0", Repository: "r", Format: "npm",
		Assets: []nexus.RawAsset{{
			Path:           "/p/lib.tgz",
			Size:           &size,
			Checksums:      map[string]string{"sha256": "aa"},
			LastDownloaded: &downloaded,
		}},
	}}
	got, err := schema.MapInfoResults(raw)
	if err != nil {
		t.Fatalf("MapInfoResults: %v", err)
	}
	a := got.Components[0].Assets[0]
	if a.Size == nil || *a.Size != 42 || a.Checksums["sha256"] != "aa" || a.LastDownloaded == nil {
		t.Errorf("asset mapping wrong: %+v", a)
	}

	if _, err := schema.MapInfoResults(nil); err == nil {
		t.Error("empty input should produce not-found error")
	}
}
