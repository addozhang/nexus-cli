package auth

import (
	"os"
	"strings"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// Resolved is the outcome of instance resolution: everything the HTTP client
// needs to talk to one Nexus instance.
type Resolved struct {
	Alias    string // empty when resolved via environment override
	BaseURL  string
	Username string
	Token    string
}

// Environment variable names for the ephemeral connection override.
const (
	EnvURL      = "NX_URL"
	EnvUsername = "NX_USERNAME"
	EnvToken    = "NX_TOKEN"
)

// Resolve applies the precedence chain:
//
//  1. explicit --instance <alias>
//  2. complete NX_URL / NX_USERNAME / NX_TOKEN environment override
//     (all-or-nothing; partial sets are ignored)
//  3. stored default instance
//  4. failure listing available aliases
func Resolve(store *Store, aliasFlag string) (*Resolved, error) {
	if aliasFlag != "" {
		inst, ok := store.Get(aliasFlag)
		if !ok {
			return nil, nxerrors.New(nxerrors.ClassInstance, "unknown instance %q (known: %s)", aliasFlag, store.AliasList())
		}
		token, err := store.Token(aliasFlag)
		if err != nil {
			return nil, err
		}
		return &Resolved{Alias: aliasFlag, BaseURL: inst.URL, Username: inst.Username, Token: token}, nil
	}

	if url, user, token, ok := envOverride(); ok {
		return &Resolved{BaseURL: url, Username: user, Token: token}, nil
	}

	alias, ok := store.DefaultAlias()
	if !ok {
		return nil, nxerrors.New(
			nxerrors.ClassInstance,
			"no default instance configured; use --instance <alias> or `nx auth default <alias>` (known: %s)",
			store.AliasList(),
		)
	}
	inst, _ := store.Get(alias)
	token, err := store.Token(alias)
	if err != nil {
		return nil, err
	}
	return &Resolved{Alias: alias, BaseURL: inst.URL, Username: inst.Username, Token: token}, nil
}

func envOverride() (string, string, string, bool) {
	url, user, token := os.Getenv(EnvURL), os.Getenv(EnvUsername), os.Getenv(EnvToken)
	if url == "" || user == "" || token == "" {
		return "", "", "", false
	}
	return strings.TrimRight(url, "/"), user, token, true
}
