package cmd

import (
	"context"
	"fmt"
	"os"

	"cowpoke/internal/app"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//nolint:gochecknoglobals // Cobra CLI pattern for persistent flag variables
var (
	cfgFile string
	verbose bool

	application *app.App
)

// VersionInfo holds build information.
type VersionInfo struct {
	Version string
	Commit  string
	Date    string
	BuiltBy string
}

//nolint:gochecknoglobals // Package-level version info for CLI commands
var versionInfo = VersionInfo{
	Version: "dev",
	Commit:  "none",
	Date:    "unknown",
	BuiltBy: "unknown",
}

// SetVersionInfo updates the build information.
func SetVersionInfo(v, c, d, b string) {
	versionInfo.Version = v
	versionInfo.Commit = c
	versionInfo.Date = d
	versionInfo.BuiltBy = b
}

// GetVersionInfo returns the current version information.
func GetVersionInfo() VersionInfo {
	return versionInfo
}

// GetApp returns the initialized application instance.
func GetApp() *app.App {
	return application
}

//nolint:gochecknoglobals // Cobra CLI pattern for root command
var rootCmd = &cobra.Command{
	Use:   "cowpoke",
	Short: "A CLI tool for managing Rancher servers and downloading kubeconfigs",
	Long: `Cowpoke is a CLI tool that helps you manage multiple Rancher servers
and download kubeconfigs from all clusters across all servers.`,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

//nolint:gochecknoinits // Cobra CLI pattern for flag initialization
func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/cowpoke/config.yaml)")
	rootCmd.PersistentFlags().
		BoolVarP(&verbose, "verbose", "v", false, "Enable verbose logging")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home + "/.config/cowpoke")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	viper.AutomaticEnv()

	// Read config file silently (ignore error if config file doesn't exist)
	_ = viper.ReadInConfig()

	// Initialize the application with dependency injection
	var opts []app.Option
	if verbose {
		opts = append(opts, app.WithVerbose(true))
	}

	var err error
	application, err = app.NewApp(context.Background(), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}
}
