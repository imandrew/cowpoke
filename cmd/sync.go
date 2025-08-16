package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cowpoke/internal/commands"
	"cowpoke/internal/domain"

	"github.com/spf13/cobra"
)

const (
	// syncTimeout is the maximum time allowed for the entire sync operation.
	syncTimeout = 10 * time.Minute
)

//nolint:gochecknoglobals // Cobra CLI pattern for subcommand
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync kubeconfigs from all Rancher servers",
	Long: `Download kubeconfigs from all configured Rancher servers and merge them into a kubeconfig file.
	
By default, the merged kubeconfig is written to ~/.kube/config. Use the --output flag to specify a different location.`,
	RunE: runSync,
}

//nolint:gochecknoinits // Cobra CLI pattern for command registration
func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().
		StringP("output", "o", "", "Output directory or file path for merged kubeconfig (default: ~/.kube/config)")
	syncCmd.Flags().
		Bool("cleanup-temp-files", false, "Remove temporary kubeconfig files after merging")
	syncCmd.Flags().
		Bool("insecure", false, "Skip TLS certificate verification for Rancher servers")
}

func runSync(cmd *cobra.Command, _ []string) error {
	// Get the initialized app instance
	app := GetApp()
	if app == nil {
		return errors.New("application not initialized")
	}

	// Extract flags
	output, _ := cmd.Flags().GetString("output")
	cleanupTempFiles, _ := cmd.Flags().GetBool("cleanup-temp-files")
	insecureSkipTLS, _ := cmd.Flags().GetBool("insecure")

	// Create Rancher services with appropriate TLS configuration
	var rancherServices domain.RancherServices
	if insecureSkipTLS {
		rancherServices = app.RancherServiceFactory.CreateInsecureServices(app.ConfigProvider)
	} else {
		rancherServices = app.RancherServiceFactory.CreateSecureServices(app.ConfigProvider)
	}

	// Create sync command with simplified dependencies
	syncCommand := commands.NewSyncCommand(
		app.ConfigRepo,
		app.ConfigProvider,
		app.PasswordReader,
		app.Logger,
	)

	// Execute with timeout
	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	// Execute the sync command with RancherServices
	err := syncCommand.Execute(ctx, commands.SyncRequest{
		Output:           output,
		InsecureSkipTLS:  insecureSkipTLS,
		CleanupTempFiles: cleanupTempFiles,
		Verbose:          app.Config.Verbose,
	}, rancherServices)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Sync completed successfully")
	return nil
}
