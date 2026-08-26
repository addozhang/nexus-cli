package cli

import (
	"github.com/spf13/cobra"
)

// NewRootCommand assembles the nx command tree.
func NewRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:          "nx",
		Short:        "Read-only query CLI for Sonatype Nexus Repository 3",
		SilenceUsage: true,
	}
	root.PersistentFlags().StringP("output", "o", "yaml", "output format: yaml or json")
	root.PersistentFlags().String("instance", "", "instance alias to query (defaults to the stored default instance)")
	root.PersistentFlags().Bool("insecure", false, "disable TLS certificate verification (last resort; prints a warning)")

	root.AddCommand(newAuthCmd(), newRepoCmd())
	for _, fc := range newFormatCmds() {
		root.AddCommand(fc)
	}

	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the nx version",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Println("nx version dev")
			return nil
		},
	}
}
