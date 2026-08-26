package nexus

import (
	"context"
	"net/url"
)

// RawRepository mirrors GET /service/rest/v1/repositories entries. Fields we
// do not surface are ignored.
type RawRepository struct {
	Name    string `json:"name"`
	Format  string `json:"format"`
	Type    string `json:"type"`
	URL     string `json:"url"`
	Exposed *bool  `json:"exposed,omitempty"`
}

// Repositories lists all repositories visible to the credentials. Format
// names are normalized to user-facing names ("maven2" -> "maven").
func (c *Client) Repositories(ctx context.Context) ([]RawRepository, error) {
	raw := []RawRepository{}
	if err := c.get(ctx, "/repositories", nil, &raw); err != nil {
		return nil, err
	}
	for i := range raw {
		if raw[i].Format == "maven2" {
			raw[i].Format = "maven"
		}
	}
	return raw, nil
}

// RawAsset mirrors an asset entry embedded in search/component payloads.
type RawAsset struct {
	Path string `json:"path"`
	// Nexus reports size under "fileSize" and checksums under "checksum".
	Size           *int64            `json:"fileSize,omitempty"`
	Checksums      map[string]string `json:"checksum,omitempty"`
	LastModified   *TimeValue        `json:"lastModified,omitempty"`
	LastDownloaded *TimeValue        `json:"lastDownloaded,omitempty"`
}

// RawComponent mirrors a component entry from the search API.
type RawComponent struct {
	ID         string     `json:"id"`
	Group      string     `json:"group,omitempty"`
	Name       string     `json:"name"`
	Version    string     `json:"version"`
	Repository string     `json:"repository"`
	Format     string     `json:"format"`
	Assets     []RawAsset `json:"assets,omitempty"`
}

// SearchParams carries constrained query values; empty fields are omitted.
type SearchParams struct {
	Format  string
	Group   string
	Name    string
	Version string
	Q       string
}

func (p SearchParams) values() url.Values {
	v := url.Values{}
	// The Search API indexes maven components under the wire name "maven2";
	// nx exposes it to users as "maven" and translates here.
	setIf := func(key, val string) {
		if val != "" {
			v.Set(key, val)
		}
	}
	setIf("format", WireFormat(p.Format))
	setIf("group", p.Group)
	setIf("name", p.Name)
	setIf("version", p.Version)
	setIf("q", p.Q)
	return v
}

// WireFormat translates user-facing format names onto Nexus wire names.
func WireFormat(f string) string {
	if f == "maven" {
		return "maven2"
	}
	return f
}

type rawSearchPage struct {
	Items             []RawComponent `json:"items"`
	ContinuationToken string         `json:"continuationToken,omitempty"`
}

// Search executes the paginated search endpoint, following continuation
// tokens until exhaustion or until limit components are collected.
// limit <= 0 means DefaultSearchLimit.
func (c *Client) Search(ctx context.Context, p SearchParams, limit int) ([]RawComponent, error) {
	if limit <= 0 {
		limit = DefaultSearchLimit
	}
	vals := p.values()
	collected := make([]RawComponent, 0, limit)
	for {
		page := rawSearchPage{}
		if err := c.get(ctx, "/search", vals, &page); err != nil {
			return nil, err
		}
		collected = append(collected, page.Items...)
		if page.ContinuationToken == "" || len(collected) >= limit {
			break
		}
		vals.Set("continuationToken", page.ContinuationToken)
	}
	if len(collected) > limit {
		collected = collected[:limit]
	}
	return collected, nil
}

// GetComponent fetches full detail (including assets) for one component id.
func (c *Client) GetComponent(ctx context.Context, id string) (*RawComponent, error) {
	out := &RawComponent{}
	if err := c.get(ctx, "/components/"+url.PathEscape(id), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}
