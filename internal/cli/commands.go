package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"cowpoke/internal/config"
	"cowpoke/internal/kubeconfig"
	"cowpoke/internal/logging"
	syncprocessor "cowpoke/internal/cmd"
	"cowpoke/internal/utils"

	"github.com/spf13/cobra"
)

// Add command variables
var (
	url      string
	username string
	authType string
)

// Remove command variables
var removeURL string

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new Rancher server to the configuration",
	Long:  `Add a new Rancher server with the specified URL, username, and authentication type.`,
	RunE:  runAdd,
}

// listCmd represents the list command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured Rancher servers",
	Long:  `List all Rancher servers that have been added to the configuration.`,
	RunE:  runList,
}

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a Rancher server from the configuration",
	Long:  `Remove a Rancher server by its URL from the configuration.`,
	RunE:  runRemove,
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, commit, build date, and build information for cowpoke.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cowpoke version %s\n", version)
		fmt.Printf("  commit: %s\n", commit)
		fmt.Printf("  built: %s\n", date)
		fmt.Printf("  built by: %s\n", builtBy)
	},
}

// InitCommands initializes all commands and adds them to root
func InitCommands(rootCmd *cobra.Command) {
	// Add commands to root
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(versionCmd)

	// Initialize add command flags
	addCmd.Flags().StringVarP(&url, "url", "u", "", "Rancher server URL (required)")
	addCmd.Flags().StringVarP(&username, "username", "n", "", "Username for authentication (required)")
	addCmd.Flags().StringVarP(&authType, "authtype", "a", "local", "Authentication type (default: local)")
	addCmd.MarkFlagRequired("url")
	addCmd.MarkFlagRequired("username")

	// Initialize remove command flags
	removeCmd.Flags().StringVarP(&removeURL, "url", "u", "", "Rancher server URL to remove (required)")
	removeCmd.MarkFlagRequired("url")
}

// runAdd handles the add command execution
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

// runList handles the list command execution
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

// runRemove handles the remove command execution
func runRemove(cmd *cobra.Command, args []string) error {
	configManager, err := utils.GetConfigManager()
	if err != nil {
		return err
	}

	err = configManager.RemoveServerByURL(removeURL)
	if err != nil {
		return fmt.Errorf("failed to remove server: %w", err)
	}

	fmt.Printf("Successfully removed Rancher server: %s\n", removeURL)
	return nil
}

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync kubeconfigs from all Rancher servers",
	Long:  `Download kubeconfigs from all configured Rancher servers and merge them into a kubeconfig file.

By default, the merged kubeconfig is written to ~/.kube/config. Use the --output flag to specify a different location.`,
	RunE:  runSync,
}

// InitSyncCommand initializes the sync command and adds it to root
func InitSyncCommand(rootCmd *cobra.Command) {
	syncCmd.Flags().StringP("output", "o", "", "Output directory or file path for merged kubeconfig (default: ~/.kube/config)")
	rootCmd.AddCommand(syncCmd)
}

// runSync handles the sync command execution
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
	processor := syncprocessor.NewSyncProcessor(kubeconfigManager, logger)
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