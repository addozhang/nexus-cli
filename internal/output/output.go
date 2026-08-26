// Package output renders schema types as YAML (default) or JSON.
//
// Rendered output always begins with schemaVersion: "1" because every schema
// type carries SchemaVersion as its first struct field; renderers never inject
// or mutate it. Diagnostics never pass through this package — stdout carries
// only rendered schema output.
package output

import (
	"encoding/json"
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v3"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

// Format selects the rendering target.
type Format string

// Supported rendering formats.
const (
	FormatYAML Format = "yaml"
	FormatJSON Format = "json"
)

// Parse validates a user-supplied -o value into a Format.
func Parse(s string) (Format, error) {
	switch Format(s) {
	case FormatYAML, "":
		return FormatYAML, nil
	case FormatJSON:
		return FormatJSON, nil
	default:
		return "", nxerrors.New(nxerrors.ClassFlag, "unsupported output format %q (supported: yaml, json)", s)
	}
}

// Write renders v in the requested format followed by a trailing newline.
func Write(w io.Writer, v any, f Format) error {
	switch f {
	case FormatJSON:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nxerrors.Wrap(nxerrors.ClassResponse, err, "render JSON")
		}
		if _, err := fmt.Fprintln(w, string(b)); err != nil {
			return nxerrors.Wrap(nxerrors.ClassNetwork, err, "write output")
		}
		return nil
	case FormatYAML:
		b, err := yaml.Marshal(v)
		if err != nil {
			return nxerrors.Wrap(nxerrors.ClassResponse, err, "render YAML")
		}
		if _, err := w.Write(b); err != nil {
			return nxerrors.Wrap(nxerrors.ClassNetwork, err, "write output")
		}
		return nil
	default:
		return nxerrors.New(nxerrors.ClassFlag, "unsupported output format %q", f)
	}
}
