package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yanjiulab/lopa/internal/version"
)

func init() {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print lopa version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version.String("lopa"))
		},
	}

	rootCmd.AddCommand(versionCmd)
}
