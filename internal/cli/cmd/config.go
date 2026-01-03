package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration commands",
	Long:  `Commands for managing JATS CLI configuration.`,
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  server_url  - JATS server URL (e.g., http://localhost:8081)

Examples:
  jats config set server_url http://localhost:8080
  jats config set server_url https://jats.example.com`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		cfg := config.GetCurrent()
		if cfg == nil {
			cfg = &config.Config{}
		}

		switch key {
		case "server_url":
			cfg.ServerURL = value
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		if err := config.Save(cfg, ""); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		// Update current config
		config.SetCurrent(cfg)

		fmt.Printf("✓ Set %s = %s\n", key, value)
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value(s)",
	Long: `Get one or all configuration values.

Examples:
  jats config get              # Show all configuration
  jats config get server_url   # Show server URL only`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.GetCurrent()
		if cfg == nil {
			fmt.Println("No configuration found")
			return nil
		}

		if len(args) == 0 {
			// Show all configuration
			fmt.Printf("server_url = %s\n", cfg.ServerURL)
			if cfg.Username != "" {
				fmt.Printf("username = %s\n", cfg.Username)
			}
			fmt.Printf("authenticated = %t\n", cfg.Username != "" && cfg.Token != "")
			return nil
		}

		key := args[0]
		switch key {
		case "server_url":
			fmt.Println(cfg.ServerURL)
		case "username":
			fmt.Println(cfg.Username)
		case "authenticated":
			fmt.Printf("%t\n", cfg.Username != "" && cfg.Token != "")
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(setCmd)
	configCmd.AddCommand(getCmd)
}