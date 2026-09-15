package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/nexus"
)

func newAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage registered Nexus instances",
	}
	cmd.AddCommand(newAuthAddCmd(), newAuthListCmd(), newAuthRemoveCmd(), newAuthDefaultCmd())
	return cmd
}

// envSecureStorage opts into keyring storage for `auth add` when set to "1".
const envSecureStorage = "NX_SECURE_STORAGE"

func newAuthAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <alias>",
		Short: "Register an instance (alias, URL, username, token) and verify connectivity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			stdin := cmd.InOrStdin()
			reader := bufio.NewReader(stdin)

			_, _ = fmt.Fprint(cmd.OutOrStdout(), "Instance URL (e.g. https://nexus.example.com:8443): ")
			url := strings.TrimSpace(readLine(reader))
			if url == "" {
				return nxerrors.New(nxerrors.ClassFlag, "instance URL is required")
			}

			_, _ = fmt.Fprint(cmd.OutOrStdout(), "Username: ")
			username := strings.TrimSpace(readLine(reader))

			_, _ = fmt.Fprint(cmd.OutOrStdout(), "Token: ")
			token := readSecret(stdin, reader)
			if token == "" {
				return nxerrors.New(nxerrors.ClassFlag, "token is required")
			}

			store, err := loadStore()
			if err != nil {
				return err
			}
			client := nexus.New(url, username, token, nil)
			if err := client.Status(cmd.Context()); err != nil {
				return nxerrors.Wrap(nxerrors.ClassNetwork, err,
					"verification against %s failed; credentials were NOT saved", url)
			}

			secureStorage, _ := cmd.Flags().GetBool("secure-storage")
			secure := secureStorageWanted(secureStorage)
			def := len(store.Aliases()) == 0
			if err := store.Add(alias, url, username, token, def, secure); err != nil {
				return err
			}
			extra := ""
			if def {
				extra = " (set as default — first instance registered)"
			}
			if secure {
				extra += " — token stored in the OS keyring"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Instance %q saved.%s\n", alias, extra)
			return nil
		},
	}
	cmd.Flags().Bool("secure-storage", false,
		"store the token in the OS keyring instead of the credentials file (also "+envSecureStorage+"=1)")
	return cmd
}

// secureStorageWanted combines the --secure-storage flag with its
// NX_SECURE_STORAGE=1 environment fallback.
func secureStorageWanted(flag bool) bool {
	return flag || os.Getenv(envSecureStorage) == "1"
}

func newAuthListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List registered instances (tokens are never printed)",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := loadStore()
			if err != nil {
				return err
			}
			for _, alias := range store.Aliases() {
				inst, _ := store.Get(alias)
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), inst.String(alias))
			}
			return nil
		},
	}
}

func newAuthRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <alias>",
		Short: "Remove a registered instance",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := loadStore()
			if err != nil {
				return err
			}
			return store.Remove(args[0])
		},
	}
}

func newAuthDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default <alias>",
		Short: "Set the default instance used when --instance is not given",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := loadStore()
			if err != nil {
				return err
			}
			if err := store.SetDefault(args[0]); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Default instance set to %q\n", args[0])
			return nil
		},
	}
}

// readLine reads one input line from r without discarding buffered bytes.
func readLine(r *bufio.Reader) string {
	line, err := r.ReadString('\n')
	if err != nil && line == "" {
		return ""
	}
	return strings.TrimRight(line, "\r\n")
}

// readSecret reads a token without echo on a TTY; falls back to plain read
// from the shared buffered reader for scripted/piped input.
func readSecret(stdin io.Reader, reader *bufio.Reader) string {
	if f, ok := stdin.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		b, err := term.ReadPassword(int(f.Fd()))
		fmt.Println()
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(b))
	}
	return readLine(reader)
}
