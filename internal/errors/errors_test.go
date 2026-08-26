package errors_test

import (
	"errors"
	"fmt"
	"testing"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
)

const wantExit = 10

func Test_Error_ExitCodeIsAlwaysTen(t *testing.T) {
	tests := []struct {
		name string
		err  *nxerrors.Error
	}{
		{"auth", nxerrors.New(nxerrors.ClassAuth, "rejected")},
		{"network", nxerrors.New(nxerrors.ClassNetwork, "timeout")},
		{"tls", nxerrors.New(nxerrors.ClassTLS, "untrusted cert")},
		{"response", nxerrors.New(nxerrors.ClassResponse, "malformed")},
		{"coordinate", nxerrors.New(nxerrors.ClassCoordinate, "bad coordinate")},
		{"instance", nxerrors.New(nxerrors.ClassInstance, "unknown alias")},
		{"flag", nxerrors.New(nxerrors.ClassFlag, "bad flag")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var coder interface{ ExitCode() int }
			if !errors.As(tt.err, &coder) {
				t.Fatalf("Error does not implement ExitCode()")
			}
			if got := coder.ExitCode(); got != wantExit {
				t.Errorf("ExitCode() = %d, want %d", got, wantExit)
			}
		})
	}
}

func Test_Error_WrapPreservesCause(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := nxerrors.Wrap(nxerrors.ClassNetwork, cause, "query failed")
	if !errors.Is(err, cause) {
		t.Fatalf("Unwrap chain broken: %v", err)
	}
	if err.Class() != nxerrors.ClassNetwork {
		t.Errorf("Class() = %q, want %q", err.Class(), nxerrors.ClassNetwork)
	}
	want := "network: query failed: connection refused"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func Test_Error_WrapNilCauseReturnsNil(t *testing.T) {
	if err := nxerrors.Wrap(nxerrors.ClassNetwork, nil, "noop"); err != nil {
		t.Errorf("Wrap(nil) = %v, want nil", err)
	}
}
