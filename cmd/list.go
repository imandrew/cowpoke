package cmd

import (
	"context"
	"errors"
	"fmt"

	"cowpoke/internal/commands"

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
	// Get the initialized app instance
	app := GetApp()
	if app == nil {
		return errors.New("application not initialized")
	}

	// Create list command with injected dependencies
	listCommand := commands.NewListCommand(
		app.ConfigRepo,
		app.Logger,
	)

	// Execute the list command
	result, err := listCommand.Execute(context.Background(), commands.ListRequest{})
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	if result.Count == 0 {
		fmt.Fprintln(
			cmd.OutOrStdout(),
			"No Rancher servers configured. Use 'cowpoke add' to add servers.",
		)
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configured Rancher servers (%d):\n\n", result.Count)
	for i, server := range result.Servers {
		fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n", i+1, server.URL)
		fmt.Fprintf(cmd.OutOrStdout(), "   ID: %s\n", server.ID())
		fmt.Fprintf(cmd.OutOrStdout(), "   Username: %s\n", server.Username)
		fmt.Fprintf(cmd.OutOrStdout(), "   Auth Type: %s\n", server.AuthType)
		if i < len(result.Servers)-1 {
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return nil
}
