package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"cowpoke/internal/kubeconfig"
	"cowpoke/internal/logging"
	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync kubeconfigs from all Rancher servers",
	Long:  `Download kubeconfigs from all configured Rancher servers and merge them into a kubeconfig file.
	
By default, the merged kubeconfig is written to ~/.kube/config. Use the --output flag to specify a different location.`,
	RunE:  runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.Flags().StringP("output", "o", "", "Output directory or file path for merged kubeconfig (default: ~/.kube/config)")
}

func runSync(cmd *cobra.Command, args []string) error {
	// Create context with timeout for the entire sync operation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	logger := logging.Default().WithOperation("sync")
	logger.InfoContext(ctx, "Starting sync operation")

	// Get configuration manager
	configManager, err := utils.GetConfigManager()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get config manager", "error", err)
		return err
	}

	// Load servers
	servers, err := configManager.GetServers()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to load servers", "error", err)
		return fmt.Errorf("failed to load servers: %w", err)
	}

	if len(servers) == 0 {
		logger.InfoContext(ctx, "No Rancher servers configured")
		fmt.Println("No Rancher servers configured. Use 'cowpoke add' to add servers.")
		return nil
	}

	logger.InfoContext(ctx, "Found servers to sync", "count", len(servers))

	// Set up kubeconfig manager
	kubeconfigDir, err := utils.GetKubeconfigDir()
	if err != nil {
		return err
	}
	kubeconfigManager := kubeconfig.NewManager(kubeconfigDir)

	// Create sync processor and process servers
	processor := NewSyncProcessor(kubeconfigManager, logger)
	kubeconfigPaths, err := processor.ProcessServers(ctx, servers)
	if err != nil {
		return err
	}

	if len(kubeconfigPaths) == 0 {
		logger.WarnContext(ctx, "No kubeconfigs were downloaded successfully")
		fmt.Println("No kubeconfigs were downloaded successfully.")
		return nil
	}

	logger.InfoContext(ctx, "Starting kubeconfig merge", "count", len(kubeconfigPaths))

	// Determine output path from flag or use default
	outputPath, err := cmd.Flags().GetString("output")
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get output flag", "error", err)
		return err
	}

	var finalOutputPath string
	if outputPath == "" {
		// Use default ~/.kube/config
		finalOutputPath, err = utils.GetDefaultKubeconfigPath()
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get default kubeconfig path", "error", err)
			return err
		}
	} else {
		// Check if it's a directory or file path
		if filepath.Ext(outputPath) == "" {
			// It's a directory, append config filename
			finalOutputPath = filepath.Join(outputPath, "config")
		} else {
			// It's a file path
			finalOutputPath = outputPath
		}
	}

	err = kubeconfigManager.MergeKubeconfigs(kubeconfigPaths, finalOutputPath)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to merge kubeconfigs", "error", err, "output_path", finalOutputPath)
		return fmt.Errorf("failed to merge kubeconfigs: %w", err)
	}

	logger.InfoContext(ctx, "Sync operation completed successfully", 
		"merged_count", len(kubeconfigPaths), 
		"output_path", finalOutputPath)
	fmt.Printf("Successfully merged %d kubeconfigs into: %s\n", len(kubeconfigPaths), finalOutputPath)
	return nil
}