// keyring.go keeps the OS keyring seam and every Store operation that touches
// it: secure instances store their token under their alias in the system
// keyring instead of the credentials file.
package auth

import (
	"errors"
	"sort"

	"github.com/zalando/go-keyring"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// keyringService namespaces nx entries inside the OS keyring.
const keyringService = "nx"

// Injectable seams for tests; production binds the real keyring.
var (
	keyringGet    = keyring.Get
	keyringSet    = keyring.Set
	keyringDelete = keyring.Delete
)

// Token returns the token for alias, reading from the OS keyring when the
// instance was registered with --secure-storage.
func (s *Store) Token(alias string) (string, error) {
	inst, ok := s.instances[alias]
	if !ok {
		return "", nxerrors.New(nxerrors.ClassInstance, "unknown instance %q (known: %s)", alias, s.AliasList())
	}
	if !inst.Secure {
		return inst.Token, nil
	}
	token, err := keyringGet(keyringService, alias)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", nxerrors.New(nxerrors.ClassAuth,
			"no token for instance %q in the OS keyring; re-run `nx auth add %s --secure-storage` to store it again",
			alias, alias)
	}
	if err != nil {
		return "", nxerrors.Wrap(nxerrors.ClassAuth, err,
			"read token for instance %q from the OS keyring; re-run `nx auth add %s` without --secure-storage to keep the token in the credentials file",
			alias, alias)
	}
	return token, nil
}

// SecureInstances returns the aliases whose tokens live in the OS keyring,
// sorted.
func (s *Store) SecureInstances() []string {
	secure := make([]string, 0)
	for alias, inst := range s.instances {
		if inst.Secure {
			secure = append(secure, alias)
		}
	}
	sort.Strings(secure)
	return secure
}

// storeToken writes the token to the keyring under the instance alias.
func storeToken(alias, token string) error {
	if err := keyringSet(keyringService, alias, token); err != nil {
		return nxerrors.Wrap(nxerrors.ClassAuth, err,
			"store token for instance %q in the OS keyring; re-run `nx auth add %s` without --secure-storage to keep the token in the credentials file",
			alias, alias)
	}
	return nil
}

// deleteToken removes the keyring entry for alias; best effort.
func deleteToken(alias string) {
	_ = keyringDelete(keyringService, alias)
}
