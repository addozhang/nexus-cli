package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	nxerrors "github.com/addozhang/nexus-cli/internal/errors"
	"github.com/addozhang/nexus-cli/internal/format"
	"github.com/addozhang/nexus-cli/internal/nexus"
	"github.com/addozhang/nexus-cli/internal/output"
	"github.com/addozhang/nexus-cli/internal/schema"
)

// newFormatCmds builds one top-level command per registered format, each with
// search / versions / info subcommands sharing the same shape.
func newFormatCmds() []*cobra.Command {
	cmds := make([]*cobra.Command, 0, 6)
	for _, name := range format.Names() {
		adapter, err := format.Get(name)
		if err != nil {
			panic(err) // registry invariant: every name from Names() resolves
		}
		fc := &cobra.Command{
			Use:   name,
			Short: "Query " + name + " components",
		}
		fc.AddCommand(newSearchCmd(adapter), newVersionsCmd(adapter), newInfoCmd(adapter))
		cmds = append(cmds, fc)
	}
	return cmds
}

func newSearchCmd(a format.Adapter) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search " + a.Name() + " components",
		Long:  "Search " + a.Name() + " components on an instance.\nQuery grammar for " + a.Name() + ": see docs/schema.md.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			params, err := a.ParseSearch(args[0])
			if err != nil {
				return err
			}

			d, err := resolveDeps(cmd)
			if err != nil {
				return err
			}
			client, err := d.client()
			if err != nil {
				return err
			}
			raw, err := client.Search(cmd.Context(), nexus.SearchParams{
				Format:  a.Name(),
				Group:   params.Group,
				Name:    params.Name,
				Version: params.Version,
				Q:       params.Q,
			}, limit)
			if err != nil {
				return err
			}
			return output.Write(cmd.OutOrStdout(), schema.MapSearchResults(raw), d.format)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 0, "maximum number of components to return (default "+strconv.Itoa(nexus.DefaultSearchLimit)+")")
	return cmd
}

func newVersionsCmd(a format.Adapter) *cobra.Command {
	return &cobra.Command{
		Use:   "versions <component>",
		Short: "List versions of a " + a.Name() + " component",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := a.ParseComponent(args[0], false)
			if err != nil {
				return err
			}

			d, err := resolveDeps(cmd)
			if err != nil {
				return err
			}
			client, err := d.client()
			if err != nil {
				return err
			}
			raw, err := client.Search(cmd.Context(), nexus.SearchParams{
				Format: a.Name(),
				Group:  ref.Group,
				Name:   ref.Name,
			}, 0)
			if err != nil {
				return err
			}
			return output.Write(cmd.OutOrStdout(), schema.MapVersionList(ref.Group, ref.Name, raw), d.format)
		},
	}
}

func newInfoCmd(a format.Adapter) *cobra.Command {
	return &cobra.Command{
		Use:   "info <component-version>",
		Short: "Show details of a specific " + a.Name() + " component version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := a.ParseComponent(args[0], true)
			if err != nil {
				return err
			}

			d, err := resolveDeps(cmd)
			if err != nil {
				return err
			}
			client, err := d.client()
			if err != nil {
				return err
			}
			raw, err := client.Search(cmd.Context(), nexus.SearchParams{
				Format:  a.Name(),
				Group:   ref.Group,
				Name:    ref.Name,
				Version: ref.Version,
			}, 0)
			if err != nil {
				return err
			}
			info, err := schema.MapInfoResults(raw)
			if err != nil {
				return nxerrors.New(nxerrors.ClassResponse, "not found on instance: %s (format %s)",
					args[0], a.Name())
			}
			return output.Write(cmd.OutOrStdout(), info, d.format)
		},
	}
}
