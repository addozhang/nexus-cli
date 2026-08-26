// Package schema defines the self-owned output types and mappers from raw
// Nexus API responses. Field names are part of the external contract tracked
// in docs/schema.md; every type carries SchemaVersion as its first field so
// rendered output begins with it.
package schema

import (
	"sort"
	"time"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/nexus"
)

// SchemaVersion is the current output schema version.
const SchemaVersion = "1"

// normalizeFormat maps Nexus wire format names onto the user-facing names of
// this schema ("maven2" is reported as "maven"); others pass through.
func normalizeFormat(f string) string {
	if f == "maven2" {
		return "maven"
	}
	return f
}

// Repo is a single repository entry (stable fields).
type Repo struct {
	Name    string `json:"name" yaml:"name"`
	Format  string `json:"format" yaml:"format"`
	Type    string `json:"type" yaml:"type"`
	URL     string `json:"url" yaml:"url"`
	Exposed *bool  `json:"exposed" yaml:"exposed"`
}

// RepoList is the nx repo list payload.
type RepoList struct {
	SchemaVersion string `json:"schemaVersion" yaml:"schemaVersion"`
	Repositories  []Repo `json:"repositories" yaml:"repositories"`
}

// MapRepoList maps raw repository responses into the stable RepoList shape.
func MapRepoList(raw []nexus.RawRepository) *RepoList {
	list := &RepoList{SchemaVersion: SchemaVersion, Repositories: make([]Repo, 0, len(raw))}
	for _, r := range raw {
		list.Repositories = append(list.Repositories, Repo{
			Name:    r.Name,
			Format:  normalizeFormat(r.Format),
			Type:    r.Type,
			URL:     r.URL,
			Exposed: r.Exposed,
		})
	}
	return list
}

// ComponentResult is one search hit.
type ComponentResult struct {
	ID         string `json:"id" yaml:"id"`
	Group      string `json:"group" yaml:"group"`
	Name       string `json:"name" yaml:"name"`
	Version    string `json:"version" yaml:"version"`
	Repository string `json:"repository" yaml:"repository"`
	Format     string `json:"format" yaml:"format"`
}

// SearchResults is the <format> search payload.
type SearchResults struct {
	SchemaVersion string            `json:"schemaVersion" yaml:"schemaVersion"`
	Results       []ComponentResult `json:"results" yaml:"results"`
}

// MapSearchResults maps raw search components into SearchResults.
func MapSearchResults(raw []nexus.RawComponent) *SearchResults {
	out := &SearchResults{SchemaVersion: SchemaVersion, Results: make([]ComponentResult, 0, len(raw))}
	for _, c := range raw {
		out.Results = append(out.Results, ComponentResult{
			ID:         c.ID,
			Group:      c.Group,
			Name:       c.Name,
			Version:    c.Version,
			Repository: c.Repository,
			Format:     normalizeFormat(c.Format),
		})
	}
	return out
}

// VersionEntry is one available version of a component.
type VersionEntry struct {
	Version      string     `json:"version" yaml:"version"`
	Repository   string     `json:"repository" yaml:"repository"`
	LastModified *time.Time `json:"lastModified" yaml:"lastModified"`
}

// VersionList is the <format> versions payload.
type VersionList struct {
	SchemaVersion string         `json:"schemaVersion" yaml:"schemaVersion"`
	Group         string         `json:"group" yaml:"group"`
	Name          string         `json:"name" yaml:"name"`
	Versions      []VersionEntry `json:"versions" yaml:"versions"`
}

// MapVersionList aggregates distinct versions from search hits for one
// component. lastModified is the newest asset modification time reported for
// that version; nil when no assets carry a timestamp.
func MapVersionList(group, name string, raw []nexus.RawComponent) *VersionList {
	type entry struct {
		repo string
		last *time.Time
	}
	seen := map[string]entry{}
	order := []string{}
	for _, c := range raw {
		if _, ok := seen[c.Version]; !ok {
			seen[c.Version] = entry{repo: c.Repository}
			order = append(order, c.Version)
		}
		e := seen[c.Version]
		for _, a := range c.Assets {
			last := nexus.AsTime(a.LastModified)
			if last == nil {
				continue
			}
			if e.last == nil || last.After(*e.last) {
				t := *last
				e.last = &t
			}
		}
		seen[c.Version] = e
	}
	sort.Slice(order, func(i, j int) bool { return CompareVersions(order[i], order[j]) > 0 })
	list := &VersionList{
		SchemaVersion: SchemaVersion,
		Group:         group,
		Name:          name,
		Versions:      make([]VersionEntry, 0, len(order)),
	}
	for _, v := range order {
		e := seen[v]
		last := e.last
		list.Versions = append(list.Versions, VersionEntry{Version: v, Repository: e.repo, LastModified: last})
	}
	return list
}

// AssetInfo describes one archived asset of a component.
type AssetInfo struct {
	Path      string            `json:"path" yaml:"path"`
	Size      *int64            `json:"size" yaml:"size"`
	Checksums map[string]string `json:"checksums" yaml:"checksums"`
	// Experimental provenance fields; promoted to stable after one minor release.
	Uploader       string     `json:"uploader" yaml:"uploader"`
	UploaderIP     string     `json:"uploaderIp" yaml:"uploaderIp"`
	BlobCreated    *time.Time `json:"blobCreated" yaml:"blobCreated"`
	LastModified   *time.Time `json:"lastModified" yaml:"lastModified"`
	LastDownloaded *time.Time `json:"lastDownloaded" yaml:"lastDownloaded"`
}

// ComponentInfo is one component occurrence in one repository, with assets.
type ComponentInfo struct {
	ID         string      `json:"id" yaml:"id"`
	Group      string      `json:"group" yaml:"group"`
	Name       string      `json:"name" yaml:"name"`
	Version    string      `json:"version" yaml:"version"`
	Format     string      `json:"format" yaml:"format"`
	Repository string      `json:"repository" yaml:"repository"`
	Assets     []AssetInfo `json:"assets" yaml:"assets"`
}

// InfoResults is the <format> info payload; one entry per repository
// occurrence of the requested component-version.
type InfoResults struct {
	SchemaVersion string          `json:"schemaVersion" yaml:"schemaVersion"`
	Components    []ComponentInfo `json:"components" yaml:"components"`
}

// MapInfoResults maps exact-match search hits into InfoResults.
func MapInfoResults(raw []nexus.RawComponent) (*InfoResults, error) {
	out := &InfoResults{SchemaVersion: SchemaVersion, Components: []ComponentInfo{}}
	for _, c := range raw {
		info := ComponentInfo{
			ID:         c.ID,
			Group:      c.Group,
			Name:       c.Name,
			Version:    c.Version,
			Format:     normalizeFormat(c.Format),
			Repository: c.Repository,
			Assets:     []AssetInfo{},
		}
		for _, a := range c.Assets {
			checksums := map[string]string{}
			for alg, sum := range a.Checksums {
				checksums[alg] = sum
			}
			size := a.Size
			info.Assets = append(info.Assets, AssetInfo{
				Path:           a.Path,
				Size:           size,
				Checksums:      checksums,
				Uploader:       a.Uploader,
				UploaderIP:     a.UploaderIP,
				BlobCreated:    nexus.AsTime(a.BlobCreated),
				LastModified:   nexus.AsTime(a.LastModified),
				LastDownloaded: nexus.AsTime(a.LastDownloaded),
			})
		}
		out.Components = append(out.Components, info)
	}
	if len(out.Components) == 0 {
		return nil, nxerrors.New(nxerrors.ClassResponse, "not found on instance")
	}
	return out, nil
}

// CompareVersions orders version strings newest-first using numeric-aware
// segment comparison ("1.10.0" > "1.9.0"). It is a heuristic ordering, not a
// semver implementation; prerelease and build metadata sort lexicographically.
func CompareVersions(a, b string) int {
	as, bs := splitSegments(a), splitSegments(b)
	for i := 0; i < len(as) && i < len(bs); i++ {
		if as[i] != bs[i] {
			return compareSegment(as[i], bs[i])
		}
	}
	switch {
	case len(as) < len(bs):
		return -1
	case len(as) > len(bs):
		return 1
	default:
		return 0
	}
}

func splitSegments(v string) []string {
	segs := []string{}
	cur := ""
	for _, r := range v {
		isDigit := r >= '0' && r <= '9'
		if cur == "" {
			cur = string(r)
			continue
		}
		prevDigit := cur[len(cur)-1] >= '0' && cur[len(cur)-1] <= '9'
		if isDigit != prevDigit {
			segs = append(segs, cur)
			cur = string(r)
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		segs = append(segs, cur)
	}
	return segs
}

func compareSegment(a, b string) int {
	an, aok := atoi(a)
	bn, bok := atoi(b)
	switch {
	case aok && bok:
		return an - bn
	case aok:
		return 1
	case bok:
		return -1
	default:
		if a < b {
			return -1
		}
		if a > b {
			return 1
		}
		return 0
	}
}

func atoi(s string) (int, bool) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}
