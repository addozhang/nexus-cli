package cli

import (
	"fmt"
	"testing"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/nexus"
)

func Test_ExitCode_NxErrorsAreTen(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil", nil, 1},
		{"plain", fmt.Errorf("boom"), 1},
		{"wrapped nx error", fmt.Errorf("context: %w", nxerrors.New(nxerrors.ClassAuth, "denied")), 10},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExitCode(tt.err); got != tt.want {
				t.Errorf("ExitCode = %d, want %d", got, tt.want)
			}
		})
	}
}

func Test_FilterRepos(t *testing.T) {
	repos := []nexus.RawRepository{
		{Name: "maven-central", Format: "maven", Type: "proxy"},
		{Name: "npm-internal", Format: "npm", Type: "hosted"},
		{Name: "docker-proxy", Format: "docker", Type: "proxy"},
	}
	got := filterRepos(repos, "docker", "proxy")
	if len(got) != 1 || got[0].Name != "docker-proxy" {
		t.Errorf("filter wrong: %+v", got)
	}
	if all := filterRepos(repos, "", ""); len(all) != len(repos) {
		t.Errorf("no filters should return everything, got %d", len(all))
	}
	if none := filterRepos(repos, "cargo", ""); len(none) != 0 {
		t.Errorf("unknown format filter should return empty, got %d", len(none))
	}
}
