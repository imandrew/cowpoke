package cmd

import (
	"fmt"

	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured Rancher servers",
	Long:  `List all Rancher servers that have been added to the configuration.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	configManager, err := utils.GetConfigManager()
	if err != nil {
		return err
	}

	servers, err := configManager.GetServers()
	if err != nil {
		return fmt.Errorf("failed to load servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Println("No Rancher servers configured. Use 'cowpoke add' to add servers.")
		return nil
	}

	fmt.Printf("Configured Rancher servers (%d):\n\n", len(servers))
	for i, server := range servers {
		fmt.Printf("%d. %s\n", i+1, server.URL)
		fmt.Printf("   Username: %s\n", server.Username)
		fmt.Printf("   Auth Type: %s\n", server.AuthType)
		if i < len(servers)-1 {
			fmt.Println()
		}
	}

	return nil
}
