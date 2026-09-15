package cli

import (
	"fmt"
	"os"
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

func Test_SecureStorageWanted_FlagOrEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		flag bool
		want bool
	}{
		{"neither", "", false, false},
		{"flag only", "", true, true},
		{"env 1", "1", false, true},
		{"env 0 ignored", "0", false, false},
		{"env true ignored", "true", false, false},
		{"env wins with flag off", "1", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.env == "" {
				t.Setenv("NX_SECURE_STORAGE", "")
				os.Unsetenv("NX_SECURE_STORAGE")
			} else {
				t.Setenv("NX_SECURE_STORAGE", tt.env)
			}
			if got := secureStorageWanted(tt.flag); got != tt.want {
				t.Errorf("secureStorageWanted(%v) with env %q = %v, want %v", tt.flag, tt.env, got, tt.want)
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
