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
	syncCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")
}

func runSync(cmd *cobra.Command, _ []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	logging.SetVerbose(verbose)

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	configManager, err := utils.GetConfigManager()
	if err != nil {
		return err
	}

	servers, err := configManager.GetServers()
	if err != nil {
		return fmt.Errorf("failed to load servers: %w", err)
	}

	if len(servers) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Rancher servers configured. Use 'cowpoke add' to add servers.")
		return nil
	}

	kubeconfigDir, err := utils.GetKubeconfigDir()
	if err != nil {
		return err
	}
	kubeconfigManager := kubeconfig.NewManager(kubeconfigDir)

	logger := logging.Get().With("operation", "sync")
	processor := NewSyncProcessor(kubeconfigManager, logger)
	kubeconfigPaths, err := processor.ProcessServers(ctx, servers)
	if err != nil {
		return err
	}

	if len(kubeconfigPaths) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No kubeconfigs were downloaded successfully.")
		return nil
	}

	outputPath, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}

	var finalOutputPath string
	if outputPath == "" {
		finalOutputPath, err = utils.GetDefaultKubeconfigPath()
		if err != nil {
			return err
		}
	} else {
		if filepath.Ext(outputPath) == "" {
			finalOutputPath = filepath.Join(outputPath, "config")
		} else {
			finalOutputPath = outputPath
		}
	}

	err = kubeconfigManager.MergeKubeconfigs(kubeconfigPaths, finalOutputPath)
	if err != nil {
		return fmt.Errorf("failed to merge kubeconfigs: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Successfully merged %d kubeconfigs into: %s\n",
		len(kubeconfigPaths), finalOutputPath)
	return nil
}
