package cli

import (
	"slices"

	"github.com/spf13/cobra"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/nexus"
	"github.com/addozhang/nexus-cli/internal/output"
	"github.com/addozhang/nexus-cli/internal/schema"
)

var repoTypes = []string{"hosted", "proxy", "group"}

func newRepoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Query repositories across formats",
	}
	list := &cobra.Command{
		Use:   "list",
		Short: "List repositories on an instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			formatFilter, _ := cmd.Flags().GetString("format")
			typeFilter, _ := cmd.Flags().GetString("type")
			if typeFilter != "" && !slices.Contains(repoTypes, typeFilter) {
				return nxerrors.New(nxerrors.ClassFlag,
					"invalid repository type %q (supported: hosted, proxy, group)", typeFilter)
			}

			d, err := resolveDeps(cmd)
			if err != nil {
				return err
			}
			client, err := d.client()
			if err != nil {
				return err
			}
			raw, err := client.Repositories(cmd.Context())
			if err != nil {
				return err
			}
			filtered := filterRepos(raw, formatFilter, typeFilter)
			return output.Write(cmd.OutOrStdout(), schema.MapRepoList(filtered), d.format)
		},
	}
	list.Flags().String("format", "", "filter by repository format (e.g. maven, npm)")
	list.Flags().String("type", "", "filter by repository type: hosted, proxy, group")
	cmd.AddCommand(list)
	return cmd
}

func filterRepos(raw []nexus.RawRepository, formatFilter, typeFilter string) []nexus.RawRepository {
	out := make([]nexus.RawRepository, 0, len(raw))
	for _, r := range raw {
		if formatFilter != "" && r.Format != formatFilter {
			continue
		}
		if typeFilter != "" && r.Type != typeFilter {
			continue
		}
		out = append(out, r)
	}
	return out
}
