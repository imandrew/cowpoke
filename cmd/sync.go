package cmd

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cowpoke/internal/commands"

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
	syncCmd.Flags().
		StringSlice("exclude", []string{}, "Exclude clusters matching regex pattern (can be specified multiple times)")
}

func runSync(cmd *cobra.Command, _ []string) error {
	app := GetApp()
	if app == nil {
		return errors.New("application not initialized")
	}

	output, _ := cmd.Flags().GetString("output")
	cleanupTempFiles, _ := cmd.Flags().GetBool("cleanup-temp-files")
	insecureSkipTLS, _ := cmd.Flags().GetBool("insecure")
	excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")

	// Debug logging for exclude patterns
	if len(excludePatterns) > 0 {
		app.Logger.Info("Exclude patterns received from CLI",
			"patterns", excludePatterns,
			"count", len(excludePatterns))
	} else {
		app.Logger.Info("No exclude patterns specified")
	}

	rancherClient := app.CreateRancherClient(insecureSkipTLS)
	syncOrchestrator := app.CreateSyncOrchestrator(rancherClient)

	syncCommand := commands.NewSyncCommand(
		app.ConfigRepo,
		app.ConfigProvider,
		app.PasswordReader,
		app.Logger,
	)

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()
	err := syncCommand.Execute(ctx, commands.SyncRequest{
		Output:           output,
		InsecureSkipTLS:  insecureSkipTLS,
		CleanupTempFiles: cleanupTempFiles,
		Verbose:          app.Config.Verbose,
		ExcludePatterns:  excludePatterns,
	}, syncOrchestrator, app.KubeconfigHandler)
	if err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Sync completed successfully")
	return nil
}
