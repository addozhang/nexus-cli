// Package format implements per-format coordinate grammars and their
// translation onto Nexus Search API parameters. Grammars are part of the
// external spec; changing them requires a spec change.
//
// Grammar summary:
//
//	maven:            groupId:artifactId[:version]
//	npm/pypi/cargo/go: name[@version]
//	docker:           image[:tag]
package format

import (
	"strings"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// Ref is a parsed component reference.
type Ref struct {
	Group   string // maven only
	Name    string
	Version string
}

// SearchParams is the adapter's translation of user input onto Search API
// constraints. Exactly one of Q or structured fields is populated.
type SearchParams struct {
	Group   string
	Name    string
	Version string
	Q       string
}

// Adapter defines one format's grammar.
type Adapter interface {
	// Name returns the Nexus format identifier.
	Name() string
	// ParseSearch translates free-form search input.
	ParseSearch(query string) (SearchParams, error)
	// ParseComponent translates a component coordinate; when requireVersion
	// is true a missing version segment is an error.
	ParseComponent(coord string, requireVersion bool) (Ref, error)
}

var registry = map[string]Adapter{}

func register(a Adapter) { registry[a.Name()] = a }

// registryInit wires the built-in format adapters. Kept as an explicit init
// so the package import alone guarantees a populated registry.
func init() {
	register(mavenAdapter{})
	register(simpleAdapter{"npm"})
	register(simpleAdapter{"pypi"})
	register(simpleAdapter{"cargo"})
	register(simpleAdapter{"go"})
	register(dockerAdapter{})
}

// Get returns the adapter for a format name, or an error naming valid ones.
func Get(name string) (Adapter, error) {
	a, ok := registry[name]
	if !ok {
		return nil, nxerrors.New(nxerrors.ClassFlag, "unsupported format %q (supported: %s)", name, strings.Join(Names(), ", "))
	}
	return a, nil
}

// Names lists registered formats in stable order.
func Names() []string {
	return []string{"maven", "npm", "pypi", "cargo", "go", "docker"}
}
