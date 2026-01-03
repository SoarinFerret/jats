package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/config"
)

var (
	cfgFile string
	serverURL string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jats",
	Short: "JATS CLI - Just Another To-do System command line interface",
	Long: `JATS CLI provides command-line access to your JATS task management system.

Manage tasks, track time, and organize your work from the command line.
Similar to ticktask but integrated with JATS server API.

Examples:
  jats add "Fix the authentication bug" urgent,backend
  jats list --status open
  jats log 123 30m
  jats close 123`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jats.toml)")
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "", "JATS server URL (overrides config)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		// Continue with defaults
		cfg = &config.Config{
			ServerURL: "http://localhost:8081",
		}
	}

	// Override server URL from flag if provided
	if serverURL != "" {
		cfg.ServerURL = serverURL
	}

	// Store config in context for commands to access
	config.SetCurrent(cfg)
}