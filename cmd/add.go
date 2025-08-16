package cmd

import (
	"context"
	"errors"
	"fmt"

	"cowpoke/internal/commands"

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
	addCmd.Flags().StringP("authtype", "a", "local", "Authentication type")

	_ = addCmd.MarkFlagRequired("url")
	_ = addCmd.MarkFlagRequired("username")
}

func runAdd(cmd *cobra.Command, _ []string) error {
	app := GetApp()
	if app == nil {
		return errors.New("application not initialized")
	}

	url, _ := cmd.Flags().GetString("url")
	username, _ := cmd.Flags().GetString("username")
	authType, _ := cmd.Flags().GetString("authtype")

	addCommand := commands.NewAddCommand(
		app.ConfigRepo,
		app.Logger,
	)
	err := addCommand.Execute(context.Background(), commands.AddRequest{
		URL:      url,
		Username: username,
		AuthType: authType,
	})
	if err != nil {
		return fmt.Errorf("failed to add server: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Successfully added Rancher server: %s\n", url)
	return nil
}
