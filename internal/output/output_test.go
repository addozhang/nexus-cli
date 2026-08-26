package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/addozhang/nexus-cli/internal/output"
)

type sample struct {
	SchemaVersion string `json:"schemaVersion" yaml:"schemaVersion"`
	Name          string `json:"name" yaml:"name"`
}

func Test_Parse_FormatValues(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    output.Format
		wantErr bool
	}{
		{"empty defaults to yaml", "", output.FormatYAML, false},
		{"yaml", "yaml", output.FormatYAML, false},
		{"json", "json", output.FormatJSON, false},
		{"table rejected", "table", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := output.Parse(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func Test_Write_YAMLFirstLineIsSchemaVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := output.Write(&buf, sample{SchemaVersion: "1", Name: "demo"}, output.FormatYAML); err != nil {
		t.Fatalf("Write: %v", err)
	}
	first := strings.SplitN(buf.String(), "\n", 2)[0]
	if first != `schemaVersion: "1"` {
		t.Errorf("first line = %q, want %q", first, `schemaVersion: "1"`)
	}
}

func Test_Write_JSONCarriesSchemaVersion(t *testing.T) {
	var buf bytes.Buffer
	if err := output.Write(&buf, sample{SchemaVersion: "1", Name: "demo"}, output.FormatJSON); err != nil {
		t.Fatalf("Write: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, buf.String())
	}
	if got["schemaVersion"] != "1" {
		t.Errorf("schemaVersion = %v, want \"1\"", got["schemaVersion"])
	}
}
