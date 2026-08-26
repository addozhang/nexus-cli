// Package nexus implements the read-only HTTP client for the Nexus
// Repository 3 REST API (v1). It knows nothing about credentials storage or
// output schema; callers translate its raw types via internal/schema.
package nexus

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"crypto/tls"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

const apiBase = "/service/rest/v1"

// DefaultSearchLimit caps component results when --limit is not given.
const DefaultSearchLimit = 50

// Client issues authenticated read requests against one Nexus instance.
type Client struct {
	baseURL  string
	username string
	token    string
	http     *http.Client
}

// New builds a Client for baseURL with basic-auth credentials. A nil tlsCfg
// means system defaults.
func New(baseURL, username, token string, tlsCfg *tls.Config) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsCfg != nil {
		transport.TLSClientConfig = tlsCfg
	}
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		token:    token,
		http:     &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

func (c *Client) get(ctx context.Context, path string, params url.Values, out any) error {
	u := c.baseURL + apiBase + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nxerrors.Wrap(nxerrors.ClassNetwork, err, "build request for %s", path)
	}
	req.SetBasicAuth(c.username, c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return translateTransport(err, path)
	}
	defer func() { _ = resp.Body.Close() }()

	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nxerrors.New(nxerrors.ClassAuth, "authentication rejected for %s (HTTP %d)", path, resp.StatusCode)
	case resp.StatusCode == http.StatusNotFound:
		return nxerrors.New(nxerrors.ClassResponse, "not found: %s", path)
	case resp.StatusCode >= 400:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nxerrors.New(nxerrors.ClassResponse, "unexpected HTTP %d from %s: %s", resp.StatusCode, path, strings.TrimSpace(string(body)))
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nxerrors.Wrap(nxerrors.ClassResponse, err, "malformed response from %s", path)
	}
	return nil
}

// Status verifies connectivity and credential acceptance via /status.
func (c *Client) Status(ctx context.Context) error {
	return c.get(ctx, "/status", nil, nil)
}

func translateTransport(err error, path string) error {
	msg := err.Error()
	if strings.Contains(msg, "certificate") || strings.Contains(msg, "x509") || strings.Contains(msg, "tls:") {
		return nxerrors.Wrap(nxerrors.ClassTLS, err, "TLS failure talking to instance (%s)", path)
	}
	return nxerrors.Wrap(nxerrors.ClassNetwork, err, "request to %s failed", path)
}
