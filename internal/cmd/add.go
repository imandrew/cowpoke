package cmd

import (
	"fmt"

	"cowpoke/internal/config"
	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new Rancher server to the configuration",
	Long:  `Add a new Rancher server with the specified URL, username, and authentication type.`,
	RunE:  runAdd,
}

var (
	url      string
	username string
	authType string
)

func init() {
	rootCmd.AddCommand(addCmd)

	addCmd.Flags().StringVarP(&url, "url", "u", "", "Rancher server URL (required)")
	addCmd.Flags().StringVarP(&username, "username", "n", "", "Username for authentication (required)")
	addCmd.Flags().StringVarP(&authType, "authtype", "a", "local", "Authentication type (default: local)")

	_ = addCmd.MarkFlagRequired("url")
	_ = addCmd.MarkFlagRequired("username")
}

func runAdd(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Successfully added Rancher server: %s\n", url)
	return nil
}
