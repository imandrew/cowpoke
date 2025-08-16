package cmd

import (
	"context"
	"errors"
	"fmt"

	"cowpoke/internal/commands"

	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals // Cobra CLI pattern for subcommand
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a Rancher server from the configuration",
	Long:  `Remove a Rancher server by its URL or ID from the configuration.`,
	RunE:  runRemove,
}

//nolint:gochecknoinits // Cobra CLI pattern for command registration
func init() {
	rootCmd.AddCommand(removeCmd)

	removeCmd.Flags().StringP("url", "u", "", "Rancher server URL to remove")
	removeCmd.Flags().StringP("id", "i", "", "Rancher server ID to remove")
}

func runRemove(cmd *cobra.Command, _ []string) error {
	// Get the initialized app instance
	app := GetApp()
	if app == nil {
		return errors.New("application not initialized")
	}

	// Extract flags
	removeURL, _ := cmd.Flags().GetString("url")
	removeID, _ := cmd.Flags().GetString("id")

	// Validate that exactly one flag is provided
	if removeURL == "" && removeID == "" {
		return errors.New("either --url or --id must be specified")
	}
	if removeURL != "" && removeID != "" {
		return errors.New("only one of --url or --id can be specified")
	}

	// Create remove command with injected dependencies
	removeCommand := commands.NewRemoveCommand(
		app.ConfigRepo,
		app.Logger,
	)

	// Execute the remove command
	err := removeCommand.Execute(context.Background(), commands.RemoveRequest{
		ServerURL: removeURL,
		ServerID:  removeID,
	})
	if err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	// Display appropriate success message
	if removeURL != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed Rancher server: %s\n", removeURL)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Successfully removed Rancher server with ID: %s\n", removeID)
	}
	return nil
}
