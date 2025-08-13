package cmd

import (
	"fmt"

	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a Rancher server from the configuration",
	Long:  `Remove a Rancher server by its URL from the configuration.`,
	RunE:  runRemove,
}

var removeURL string

func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringVarP(&removeURL, "url", "u", "", "Rancher server URL to remove (required)")
	_ = removeCmd.MarkFlagRequired("url")
}

func runRemove(cmd *cobra.Command, args []string) error {
	configManager, err := utils.GetConfigManager()
	if err != nil {
		return err
	}

	err = configManager.RemoveServerByURL(removeURL)
	if err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	fmt.Printf("Successfully removed Rancher server: %s\n", removeURL)
	return nil
}
