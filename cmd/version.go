package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra CLI pattern for subcommand
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, commit, build date, and build information for cowpoke.`,
	Run: func(cmd *cobra.Command, _ []string) {
		info := GetVersionInfo()
		fmt.Fprintf(cmd.OutOrStdout(), "cowpoke version %s\n", info.Version)
		fmt.Fprintf(cmd.OutOrStdout(), "  commit: %s\n", info.Commit)
		fmt.Fprintf(cmd.OutOrStdout(), "  built: %s\n", info.Date)
		fmt.Fprintf(cmd.OutOrStdout(), "  built by: %s\n", info.BuiltBy)
	},
}

//nolint:gochecknoinits // Cobra CLI pattern for command registration
func init() {
	rootCmd.AddCommand(versionCmd)
}
