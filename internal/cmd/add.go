package cmd

import (
	"fmt"

	"cowpoke/internal/config"
	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra CLI pattern for subcommand
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new Rancher server to the configuration",
	Long:  `Add a new Rancher server with the specified URL, username, and authentication type.`,
	RunE:  runAdd,
}

//nolint:gochecknoinits // Cobra CLI pattern for command registration
func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringP("url", "u", "", "Rancher server URL (required)")
	addCmd.Flags().StringP("username", "n", "", "Username for authentication (required)")
	addCmd.Flags().StringP("authtype", "a", "local", "Authentication type (default: local)")

	_ = addCmd.MarkFlagRequired("url")
	_ = addCmd.MarkFlagRequired("username")
}

func runAdd(cmd *cobra.Command, _ []string) error {
	url, _ := cmd.Flags().GetString("url")
	username, _ := cmd.Flags().GetString("username")
	authType, _ := cmd.Flags().GetString("authtype")

	configManager, err := utils.GetConfigManager()
	if err != nil {
		return err
	}

	server := config.RancherServer{
		Name:     url,
		URL:      url,
		Username: username,
		AuthType: authType,
	}

	err = configManager.AddServer(server)
	if err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully added Rancher server: %s\n", url)
	return nil
}
