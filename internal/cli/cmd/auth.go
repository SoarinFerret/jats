package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
	"github.com/soarinferret/jats/internal/cli/config"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Commands for managing authentication with JATS server.`,
}

var (
	password string
)

var loginCmd = &cobra.Command{
	Use:   "login [username]",
	Short: "Log in to JATS server",
	Long: `Log in to JATS server and save authentication token.

Examples:
  jats auth login              # Prompt for username and password
  jats auth login admin        # Login as admin, prompt for password
  jats auth login admin --password secret  # Login with password (for testing)`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var username string
		
		if len(args) > 0 {
			username = args[0]
		} else {
			fmt.Print("Username: ")
			fmt.Scanln(&username)
		}
		
		if username == "" {
			return fmt.Errorf("username is required")
		}

		var finalPassword string
		if password != "" {
			// Password provided via flag
			finalPassword = password
		} else {
			// Prompt for password
			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				return fmt.Errorf("failed to read password: %w", err)
			}
			fmt.Println() // Add newline after password input
			
			finalPassword = strings.TrimSpace(string(passwordBytes))
			if finalPassword == "" {
				return fmt.Errorf("password is required")
			}
		}

		c := client.New()
		resp, err := c.Login(username, finalPassword)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		// Save updated config with token
		cfg := config.GetCurrent()
		if cfg != nil {
			if err := config.Save(cfg, ""); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to save config: %v\n", err)
			}
		}

		fmt.Printf("✓ Logged in successfully as %s (%s)\n", resp.User.Username, resp.User.Email)
		return nil
	},
}

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out from JATS server",
	Long:  `Log out and remove saved authentication token.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.GetCurrent()
		if cfg != nil {
			cfg.Username = ""
			cfg.Token = ""
			
			if err := config.Save(cfg, ""); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
		}

		fmt.Println("✓ Logged out successfully")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Show current authentication status and server configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.GetCurrent()
		if cfg == nil {
			fmt.Println("No configuration found")
			return nil
		}

		fmt.Printf("Server URL: %s\n", cfg.ServerURL)
		
		if cfg.Username != "" && cfg.Token != "" {
			fmt.Printf("Logged in as: %s\n", cfg.Username)
			fmt.Println("Status: Authenticated (session token)")
		} else {
			fmt.Println("Status: Not logged in")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)
	
	loginCmd.Flags().StringVarP(&password, "password", "p", "", "Password (for testing - use interactive prompt for security)")
}