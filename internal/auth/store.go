// Package auth manages the alias-based instance registry: TOML credential
// storage at ~/.config/nx/credentials and instance resolution at query time.
package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// Instance is one registered Nexus instance.
type Instance struct {
	URL      string `toml:"url"`
	Username string `toml:"username"`
	Token    string `toml:"token"`
	Default  bool   `toml:"default"`
}

// storeFile is the on-disk layout of the credentials file.
type storeFile struct {
	Instances map[string]Instance `toml:"instances"`
}

// Store wraps the parsed credentials file.
type Store struct {
	instances map[string]Instance
	path      string
}

// DefaultPath returns ~/.config/nx/credentials honouring XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); dir != "" {
		return filepath.Join(dir, "nx", "credentials"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", nxerrors.Wrap(nxerrors.ClassNetwork, err, "resolve home directory")
	}
	return filepath.Join(home, ".config", "nx", "credentials"), nil
}

// Load reads the credentials file. A missing file yields an empty store.
func Load(path string) (*Store, error) {
	s := &Store{instances: map[string]Instance{}, path: path}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, nxerrors.Wrap(nxerrors.ClassNetwork, err, "read credentials %s", path)
	}
	var f storeFile
	if _, err := toml.Decode(string(b), &f); err != nil {
		return nil, nxerrors.Wrap(nxerrors.ClassResponse, err, "malformed credentials file %s", path)
	}
	if f.Instances != nil {
		s.instances = f.Instances
	}
	return s, nil
}

// Save persists the store with mode 0600, creating the parent directory with
// mode 0700 when absent.
func (s *Store) Save() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nxerrors.Wrap(nxerrors.ClassNetwork, err, "create config directory %s", dir)
	}
	var buf strings.Builder
	if err := toml.NewEncoder(&buf).Encode(storeFile{Instances: s.instances}); err != nil {
		return nxerrors.Wrap(nxerrors.ClassResponse, err, "encode credentials")
	}
	if err := os.WriteFile(s.path, []byte(buf.String()), 0o600); err != nil {
		return nxerrors.Wrap(nxerrors.ClassNetwork, err, "write credentials %s", s.path)
	}
	return nil
}

// Add registers or replaces an instance. When def is true other instances'
// default markers are cleared.
func (s *Store) Add(alias, url, username, token string, def bool) error {
	if alias == "" {
		return nxerrors.New(nxerrors.ClassFlag, "alias is required")
	}
	if def {
		s.clearDefaults()
	}
	s.instances[alias] = Instance{URL: normalizeURL(url), Username: username, Token: token, Default: def}
	return s.Save()
}

// Remove deletes an instance; removing the default clears the marker.
func (s *Store) Remove(alias string) error {
	if _, ok := s.instances[alias]; !ok {
		return nxerrors.New(nxerrors.ClassInstance, "unknown instance %q (known: %s)", alias, s.AliasList())
	}
	delete(s.instances, alias)
	return s.Save()
}

// SetDefault marks alias as the only default instance.
func (s *Store) SetDefault(alias string) error {
	inst, ok := s.instances[alias]
	if !ok {
		return nxerrors.New(nxerrors.ClassInstance, "unknown instance %q (known: %s)", alias, s.AliasList())
	}
	s.clearDefaults()
	inst.Default = true
	s.instances[alias] = inst
	return s.Save()
}

func (s *Store) clearDefaults() {
	for k, v := range s.instances {
		if v.Default {
			v.Default = false
			s.instances[k] = v
		}
	}
}

// Get returns a stored instance by alias.
func (s *Store) Get(alias string) (Instance, bool) {
	inst, ok := s.instances[alias]
	return inst, ok
}

// Aliases returns all aliases sorted.
func (s *Store) Aliases() []string {
	out := make([]string, 0, len(s.instances))
	for k := range s.instances {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// AliasList renders sorted aliases for error messages.
func (s *Store) AliasList() string {
	aliases := s.Aliases()
	if len(aliases) == 0 {
		return "(none)"
	}
	return strings.Join(aliases, ", ")
}

// DefaultAlias returns the stored default alias, if any.
func (s *Store) DefaultAlias() (string, bool) {
	for k, v := range s.instances {
		if v.Default {
			return k, true
		}
	}
	return "", false
}

// String renders one instance line for auth list output.
func (i Instance) String(alias string) string {
	marker := ""
	if i.Default {
		marker = " (default)"
	}
	return fmt.Sprintf("%s: %s as %s%s", alias, i.URL, i.Username, marker)
}

func normalizeURL(u string) string { return strings.TrimRight(strings.TrimSpace(u), "/") }
