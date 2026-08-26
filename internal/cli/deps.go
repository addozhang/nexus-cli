package cli

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/addozhang/nexus-cli/internal/auth"
	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/nexus"
	"github.com/addozhang/nexus-cli/internal/output"
)

// deps carries everything a command needs per invocation.
type deps struct {
	instance *auth.Resolved
	format   output.Format
	insecure bool
}

// client builds the Nexus HTTP client honoring the TLS posture.
func (d *deps) client() (*nexus.Client, error) {
	tlsCfg, err := tlsConfig(d.insecure)
	if err != nil {
		return nil, err
	}
	return nexus.New(d.instance.BaseURL, d.instance.Username, d.instance.Token, tlsCfg), nil
}

func tlsConfig(insecure bool) (*tls.Config, error) {
	if insecure {
		fmt.Fprintln(os.Stderr, "warning: TLS certificate verification disabled by --insecure")
		return &tls.Config{InsecureSkipVerify: true}, nil //nolint:gosec // explicit last-resort flag
	}
	if pemPath := os.Getenv("SSL_CERT_FILE"); pemPath != "" {
		pem, err := os.ReadFile(pemPath)
		if err != nil {
			return nil, nxerrors.Wrap(nxerrors.ClassTLS, err, "read SSL_CERT_FILE %s", pemPath)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, nxerrors.New(nxerrors.ClassTLS, "no certificates found in SSL_CERT_FILE %s", pemPath)
		}
		return &tls.Config{RootCAs: pool}, nil
	}
	return nil, nil
}

// resolveDeps resolves instance and output format from persistent flags.
func resolveDeps(cmd *cobra.Command) (*deps, error) {
	store, err := loadStore()
	if err != nil {
		return nil, err
	}
	aliasFlag, _ := cmd.Flags().GetString("instance")
	resolved, err := auth.Resolve(store, aliasFlag)
	if err != nil {
		return nil, err
	}
	formatValue, _ := cmd.Flags().GetString("output")
	f, err := output.Parse(formatValue)
	if err != nil {
		return nil, err
	}
	insecure, _ := cmd.Flags().GetBool("insecure")
	return &deps{instance: resolved, format: f, insecure: insecure}, nil
}

func loadStore() (*auth.Store, error) {
	path, err := auth.DefaultPath()
	if err != nil {
		return nil, err
	}
	return auth.Load(path)
}

// ExitCode maps an error to the process exit code: 10 for nx-level errors,
// 1 for anything else (cobra usage errors etc.).
func ExitCode(err error) int {
	var coder interface{ ExitCode() int }
	if errors.As(err, &coder) {
		return coder.ExitCode()
	}
	return 1
}
