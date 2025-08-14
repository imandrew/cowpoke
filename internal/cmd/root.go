package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//nolint:gochecknoglobals // Cobra CLI pattern for persistent flag variable
var cfgFile string

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

	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/cowpoke/config.yaml)")
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

	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
