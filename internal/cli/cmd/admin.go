package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"text/tabwriter"

	"golang.org/x/term"
	"github.com/spf13/cobra"
	"github.com/soarinferret/jats/internal/cli/client"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Administrative commands",
	Long:  `Administrative commands for managing users and system settings. Requires admin permissions.`,
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
	Long:  `Commands for managing users in the JATS system.`,
}

// User management variables
var (
	userEmail    string
	userActive   bool
	userInactive bool
)

var userCreateCmd = &cobra.Command{
	Use:   "create <username> [email]",
	Short: "Create a new user",
	Long: `Create a new user in the JATS system.

Examples:
  jats admin user create alice alice@example.com
  jats admin user create bob --email bob@example.com
  jats admin user create charlie --inactive  # Create inactive user`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		
		email := userEmail
		if len(args) > 1 && email == "" {
			email = args[1]
		}
		
		if email == "" {
			return fmt.Errorf("email is required")
		}

		// Prompt for password
		fmt.Print("Password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println() // Add newline after password input
		
		password := strings.TrimSpace(string(passwordBytes))
		if password == "" {
			return fmt.Errorf("password is required")
		}

		c := client.New()
		
		createReq := map[string]interface{}{
			"username":  username,
			"email":     email,
			"password":  password,
			"is_active": !userInactive,
		}

		resp, err := c.Post("/api/v1/admin/users", createReq)
		if err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}

		fmt.Printf("✓ User '%s' created successfully (ID: %v)\n", username, resp["data"].(map[string]interface{})["id"])
		return nil
	},
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Long:  `List all users in the JATS system.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := client.New()
		
		resp, err := c.Get("/api/v1/admin/users")
		if err != nil {
			return fmt.Errorf("failed to get users: %w", err)
		}

		users, ok := resp["data"].([]interface{})
		if !ok {
			return fmt.Errorf("unexpected response format")
		}

		if len(users) == 0 {
			fmt.Println("No users found")
			return nil
		}

		// Create table writer
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tUSERNAME\tEMAIL\tACTIVE\tTOTP\tLAST LOGIN\tCREATED\n")
		fmt.Fprintf(w, "--\t--------\t-----\t------\t----\t----------\t-------\n")

		for _, userInterface := range users {
			user := userInterface.(map[string]interface{})
			
			id := user["id"]
			username := user["username"]
			email := user["email"]
			isActive := "No"
			if user["is_active"].(bool) {
				isActive = "Yes"
			}
			totpEnabled := "No"
			if user["totp_enabled"].(bool) {
				totpEnabled = "Yes"
			}
			
			lastLogin := "Never"
			if user["last_login_at"] != nil {
				lastLogin = user["last_login_at"].(string)
				if len(lastLogin) > 16 {
					lastLogin = lastLogin[:16] // Truncate for display
				}
			}
			
			createdAt := user["created_at"].(string)
			if len(createdAt) > 10 {
				createdAt = createdAt[:10] // Just the date
			}

			fmt.Fprintf(w, "%.0f\t%s\t%s\t%s\t%s\t%s\t%s\n",
				id.(float64), username, email, isActive, totpEnabled, lastLogin, createdAt)
		}
		
		w.Flush()
		return nil
	},
}

var userShowCmd = &cobra.Command{
	Use:   "show <user_id>",
	Short: "Show details for a specific user",
	Long:  `Show detailed information for a specific user.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		
		c := client.New()
		
		resp, err := c.Get("/api/v1/admin/users/" + userID)
		if err != nil {
			return fmt.Errorf("failed to get user: %w", err)
		}

		user := resp["data"].(map[string]interface{})
		
		fmt.Printf("User Details:\n")
		fmt.Printf("  ID: %.0f\n", user["id"].(float64))
		fmt.Printf("  Username: %s\n", user["username"])
		fmt.Printf("  Email: %s\n", user["email"])
		fmt.Printf("  Active: %t\n", user["is_active"])
		fmt.Printf("  TOTP Enabled: %t\n", user["totp_enabled"])
		fmt.Printf("  Created: %s\n", user["created_at"])
		fmt.Printf("  Updated: %s\n", user["updated_at"])
		
		if user["last_login_at"] != nil {
			fmt.Printf("  Last Login: %s\n", user["last_login_at"])
		} else {
			fmt.Printf("  Last Login: Never\n")
		}
		
		return nil
	},
}

var userUpdateCmd = &cobra.Command{
	Use:   "update <user_id>",
	Short: "Update a user",
	Long: `Update user information.

Examples:
  jats admin user update 123 --email newemail@example.com
  jats admin user update 123 --active
  jats admin user update 123 --inactive`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		
		updateReq := make(map[string]interface{})
		
		if userEmail != "" {
			updateReq["email"] = userEmail
		}
		
		if userActive && userInactive {
			return fmt.Errorf("cannot specify both --active and --inactive")
		}
		
		if userActive {
			updateReq["is_active"] = true
		} else if userInactive {
			updateReq["is_active"] = false
		}
		
		if len(updateReq) == 0 {
			return fmt.Errorf("no updates specified")
		}

		c := client.New()
		
		_, err := c.Put("/api/v1/admin/users/"+userID, updateReq)
		if err != nil {
			return fmt.Errorf("failed to update user: %w", err)
		}

		fmt.Printf("✓ User %s updated successfully\n", userID)
		return nil
	},
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <user_id>",
	Short: "Delete a user",
	Long:  `Delete a user from the JATS system. This action is irreversible.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		
		// Confirm deletion
		fmt.Printf("Are you sure you want to delete user %s? (y/N): ", userID)
		var confirmation string
		fmt.Scanln(&confirmation)
		
		if strings.ToLower(confirmation) != "y" && strings.ToLower(confirmation) != "yes" {
			fmt.Println("Deletion cancelled")
			return nil
		}

		c := client.New()
		
		err := c.Delete("/api/v1/admin/users/" + userID)
		if err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		fmt.Printf("✓ User %s deleted successfully\n", userID)
		return nil
	},
}

var userResetPasswordCmd = &cobra.Command{
	Use:   "reset-password <user_id>",
	Short: "Reset a user's password",
	Long:  `Reset a user's password. This will invalidate all existing sessions.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		userID := args[0]
		
		// Prompt for new password
		fmt.Print("New password: ")
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		fmt.Println() // Add newline after password input
		
		newPassword := strings.TrimSpace(string(passwordBytes))
		if newPassword == "" {
			return fmt.Errorf("password is required")
		}

		c := client.New()
		
		resetReq := map[string]string{
			"new_password": newPassword,
		}
		
		_, err = c.Post("/api/v1/admin/users/"+userID+"/reset-password", resetReq)
		if err != nil {
			return fmt.Errorf("failed to reset password: %w", err)
		}

		fmt.Printf("✓ Password reset successfully for user %s\n", userID)
		return nil
	},
}

func init() {
	// Add admin command to root
	rootCmd.AddCommand(adminCmd)
	
	// Add user subcommand to admin
	adminCmd.AddCommand(userCmd)
	
	// Add user management commands
	userCmd.AddCommand(userCreateCmd)
	userCmd.AddCommand(userListCmd)
	userCmd.AddCommand(userShowCmd)
	userCmd.AddCommand(userUpdateCmd)
	userCmd.AddCommand(userDeleteCmd)
	userCmd.AddCommand(userResetPasswordCmd)
	
	// Flags for user create
	userCreateCmd.Flags().StringVar(&userEmail, "email", "", "User email address")
	userCreateCmd.Flags().BoolVar(&userInactive, "inactive", false, "Create user as inactive")
	
	// Flags for user update
	userUpdateCmd.Flags().StringVar(&userEmail, "email", "", "New email address")
	userUpdateCmd.Flags().BoolVar(&userActive, "active", false, "Set user as active")
	userUpdateCmd.Flags().BoolVar(&userInactive, "inactive", false, "Set user as inactive")
}