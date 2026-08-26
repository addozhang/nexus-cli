package schema_test

import (
	"testing"

	"github.com/addozhang/nexus-cli/internal/schema"
)

func Test_CompareVersions_NumericAwareOrdering(t *testing.T) {
	tests := []struct {
		a, b string
		want int // sign only
	}{
		{"1.10.0", "1.9.0", 1},
		{"1.9.0", "1.10.0", -1},
		{"2.0.0", "2.0.0", 0},
		{"4.17.21", "4.17.20", 1},
		{"1.0.0-rc1", "1.0.0-rc2", -1},
		{"10", "9", 1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := schema.CompareVersions(tt.a, tt.b)
			sign := 0
			switch {
			case got > 0:
				sign = 1
			case got < 0:
				sign = -1
			}
			if sign != tt.want {
				t.Errorf("CompareVersions(%q,%q) sign = %d, want %d", tt.a, tt.b, sign, tt.want)
			}
		})
	}
}
