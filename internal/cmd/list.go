package cmd

import (
	"fmt"

	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra CLI pattern for subcommand
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured Rancher servers",
	Long:  `List all Rancher servers that have been added to the configuration.`,
	RunE:  runList,
}

//nolint:gochecknoinits // Cobra CLI pattern for command registration
func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	configManager, err := utils.GetConfigManager()
	if err != nil {
		return err
	}

	servers, err := configManager.GetServers()
	if err != nil {
		return fmt.Errorf("failed to load servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Rancher servers configured. Use 'cowpoke add' to add servers.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configured Rancher servers (%d):\n\n", len(servers))
	for i, server := range servers {
		fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n", i+1, server.URL)
		fmt.Fprintf(cmd.OutOrStdout(), "   Username: %s\n", server.Username)
		fmt.Fprintf(cmd.OutOrStdout(), "   Auth Type: %s\n", server.AuthType)
		if i < len(servers)-1 {
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return nil
}
