package cmd

import (
	"fmt"

	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra CLI pattern for subcommand
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a Rancher server from the configuration",
	Long:  `Remove a Rancher server by its URL from the configuration.`,
	RunE:  runRemove,
}

//nolint:gochecknoinits // Cobra CLI pattern for command registration
func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringP("url", "u", "", "Rancher server URL to remove (required)")
	_ = removeCmd.MarkFlagRequired("url")
}

func runRemove(cmd *cobra.Command, _ []string) error {
	removeURL, _ := cmd.Flags().GetString("url")

	configManager, err := utils.GetConfigManager()
	if err != nil {
		return err
	}

	err = configManager.RemoveServerByURL(removeURL)
	if err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed Rancher server: %s\n", removeURL)
	return nil
}
