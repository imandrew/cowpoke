package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, commit, build date, and build information for cowpoke.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cowpoke version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", date)
		fmt.Printf("  built by: %s\n", builtBy)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
