package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"syscall"

	"golang.org/x/term"
	"github.com/soarinferret/jats/internal/config"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
	"github.com/soarinferret/jats/internal/routes"
	"github.com/soarinferret/jats/internal/services"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openDatabase opens a database connection with the appropriate driver based on the URL scheme
func openDatabase(dbURL string) (*gorm.DB, error) {
	// Detect database type from URL scheme
	if strings.HasPrefix(dbURL, "sqlite:") || strings.HasSuffix(dbURL, ".db") || strings.Contains(dbURL, "file:") {
		// SQLite database
		// Remove sqlite: prefix if present
		dbPath := strings.TrimPrefix(dbURL, "sqlite:")
		return gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	} else if strings.HasPrefix(dbURL, "postgres://") || strings.HasPrefix(dbURL, "postgresql://") {
		// PostgreSQL database
		return gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	} else {
		// Default to PostgreSQL for backward compatibility
		return gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	}
}

// generateRandomPassword generates a secure random password
func generateRandomPassword(length int) string {
	bytes := make([]byte, length/2) // hex encoding doubles the length
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a basic random string if crypto rand fails
		return "jats-admin-" + hex.EncodeToString([]byte("fallback"))[:8]
	}
	return hex.EncodeToString(bytes)
}

// createDefaultAdminUser creates the default admin user if no users exist
func createDefaultAdminUser(authService *services.AuthService, port string) error {
	// Check if any users already exist
	// We'll use a simple approach - try to get the admin user
	// If it fails, we assume no users exist and create the admin
	adminUser, err := authService.GetUserByUsername("jats-admin")
	if err != nil {
		// Admin user already exists
		return nil
	}
	if adminUser != nil {
		// Admin user already exists
		return nil
	}

	// Generate a random password
	password := generateRandomPassword(16)

	// Create the admin user
	adminUser, err = authService.RegisterUser("jats-admin", "admin@localhost", password)
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	log.Println("=====================================")
	log.Println("🎉 JATS Initial Setup Complete!")
	log.Println("=====================================")
	log.Printf("Default admin user created:")
	log.Printf("  Username: jats-admin")
	log.Printf("  Password: %s", password)
	log.Printf("  Email:    admin@localhost")
	log.Println("=====================================")
	log.Printf("Login at: http://localhost:%s/login", port)
	log.Println("⚠️  Please save these credentials and change the password after first login!")
	log.Println("=====================================")

	_ = adminUser // Use the variable to avoid unused warning
	return nil
}

// handlePasswordReset handles the password reset command
func handlePasswordReset(authService *services.AuthService, username string) error {
	fmt.Printf("Resetting password for user: %s\n", username)
	
	// Check if user exists
	user, err := authService.GetUserByUsername(username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user '%s' not found", username)
	}
	
	// Prompt for new password
	fmt.Print("Enter new password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Add newline after password input
	
	newPassword := strings.TrimSpace(string(passwordBytes))
	if newPassword == "" {
		return fmt.Errorf("password cannot be empty")
	}
	
	// Confirm password
	fmt.Print("Confirm new password: ")
	confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read confirmation password: %w", err)
	}
	fmt.Println() // Add newline after password input
	
	confirmPassword := strings.TrimSpace(string(confirmBytes))
	if newPassword != confirmPassword {
		return fmt.Errorf("passwords do not match")
	}
	
	// Reset the password
	if err := authService.ResetPassword(username, newPassword); err != nil {
		return fmt.Errorf("failed to reset password: %w", err)
	}
	
	fmt.Printf("✓ Password reset successfully for user '%s'\n", username)
	fmt.Println("✓ All existing sessions have been invalidated")
	return nil
}

// handleListUsers handles the list users command
func handleListUsers(authRepo *repository.AuthRepository) error {
	fmt.Println("Users in the system:")
	fmt.Println("====================")
	
	// Get all users - we need to implement this in the repository
	users, err := getAllUsers(authRepo)
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}
	
	if len(users) == 0 {
		fmt.Println("No users found")
		return nil
	}
	
	fmt.Printf("%-20s %-30s %-10s %-20s\n", "Username", "Email", "Active", "Last Login")
	fmt.Printf("%-20s %-30s %-10s %-20s\n", "--------", "-----", "------", "----------")
	
	for _, user := range users {
		activeStatus := "Yes"
		if !user.IsActive {
			activeStatus = "No"
		}
		
		lastLogin := "Never"
		if user.LastLoginAt != nil {
			lastLogin = user.LastLoginAt.Format("2006-01-02 15:04")
		}
		
		fmt.Printf("%-20s %-30s %-10s %-20s\n", user.Username, user.Email, activeStatus, lastLogin)
	}
	
	fmt.Printf("\nTotal users: %d\n", len(users))
	return nil
}

// getAllUsers gets all users from the repository
func getAllUsers(authRepo *repository.AuthRepository) ([]models.User, error) {
	return authRepo.GetAllUsers()
}

func main() {
	// Parse command-line flags
	var configFile string
	var resetPasswordUser string
	var listUsers bool
	
	flag.StringVar(&configFile, "c", "", "Path to TOML configuration file")
	flag.StringVar(&configFile, "config", "", "Path to TOML configuration file")
	flag.StringVar(&resetPasswordUser, "reset-password", "", "Reset password for specified username")
	flag.BoolVar(&listUsers, "list-users", false, "List all users")
	
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "JATS - Just Another To-do System\n\n")
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", "jatsd")
		flag.PrintDefaults()
		fmt.Fprintf(flag.CommandLine.Output(), "\nExamples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  jatsd                              # Start server\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  jatsd -c config.toml               # Start server with config file\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  jatsd -reset-password username     # Reset user password\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  jatsd -list-users                  # List all users\n")
		fmt.Fprintf(flag.CommandLine.Output(), "\nConfig files:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  See config.example.toml for full configuration options\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Environment variables override config file values\n")
	}
	flag.Parse()

	// Initialize configuration
	var cfg *config.Config
	var err error

	if configFile != "" {
		log.Printf("Loading configuration from: %s", configFile)
		cfg, err = config.LoadFromFile(configFile)
		if err != nil {
			log.Fatal("Failed to load configuration file:", err)
		}
	} else {
		log.Println("Loading configuration from environment variables")
		cfg = config.Load()
	}

	dbURL := cfg.DatabaseURL()
	log.Printf("Server configuration - Port: %s, Database: %s", cfg.Port, dbURL)

	// Log email configuration status
	if cfg.Email.IMAPHost != "" {
		log.Printf("Email configuration found:")
		log.Printf("  IMAP: %s:%s (SSL: %t)", cfg.Email.IMAPHost, cfg.Email.IMAPPort, cfg.Email.UseSSL)
		log.Printf("  SMTP: %s:%s (TLS: %t)", cfg.Email.SMTPHost, cfg.Email.SMTPPort, cfg.Email.SMTPUseTLS)
		log.Printf("  Poll interval: %s", cfg.Email.PollInterval)
	}

	// Setup database connection with appropriate driver
	db, err := openDatabase(dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate database schema
	err = db.AutoMigrate(
		&models.Task{},
		&models.Subtask{},
		&models.TimeEntry{},
		&models.Comment{},
		&models.EmailMessage{},
		&models.TaskSubscriber{},
		&models.Attachment{},
		&models.SavedQuery{},
		&models.User{},
		&models.Session{},
		&models.APIKey{},
		&models.LoginAttempt{},
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// Initialize repositories
	taskRepo := repository.NewTaskRepository(db)
	authRepo := repository.NewAuthRepository(db)

	// Initialize storage service for email attachments
	storageService := services.NewStorageService("./attachments")
	log.Println("Initialized storage service at ./attachments")

	// Initialize SMTP service for sending notifications
	smtpService := services.NewSMTPService(&cfg.Email)

	// Initialize notification service
	notificationService := services.NewNotificationService(taskRepo, authRepo, smtpService)

	// Initialize services with notification support
	taskService := services.NewTaskService(taskRepo, notificationService)
	authService := services.NewAuthService(authRepo, nil)

	// Handle admin commands if provided
	if resetPasswordUser != "" {
		if err := handlePasswordReset(authService, resetPasswordUser); err != nil {
			log.Fatal("Password reset failed:", err)
		}
		return
	}
	
	if listUsers {
		if err := handleListUsers(authRepo); err != nil {
			log.Fatal("List users failed:", err)
		}
		return
	}

	log.Println("Starting JATS server...")

	// Initialize email service if email configuration is provided
	var emailService *services.EmailService
	if cfg.Email.IMAPHost != "" && cfg.Email.IMAPUsername != "" && cfg.Email.IMAPPassword != "" {
		emailService = services.NewEmailService(taskService, taskRepo, authRepo, storageService, cfg)
		log.Printf("Initialized email service for %s@%s:%s", cfg.Email.IMAPUsername, cfg.Email.IMAPHost, cfg.Email.IMAPPort)

		// Start email polling in a separate goroutine
		go func() {
			log.Printf("Starting email polling with interval: %s", cfg.Email.PollInterval)
			if err := emailService.StartPolling(); err != nil {
				log.Printf("Email polling stopped with error: %v", err)
			}
		}()
	} else {
		log.Println("Email service disabled - IMAP configuration not provided")
	}

	// Create default admin user on first startup
	if err := createDefaultAdminUser(authService, cfg.Port); err != nil {
		log.Printf("Warning: Failed to create default admin user: %v", err)
	}

	// Setup routes and handlers with dependencies
	mux := routes.SetupRoutes(taskService, authService, authRepo)

	// Start HTTP server
	log.Println("==============================================")
	log.Printf("🚀 JATS Server starting on port %s", cfg.Port)
	log.Printf("📱 Web interface: http://localhost:%s/", cfg.Port)
	log.Printf("🔌 API endpoints: http://localhost:%s/api/v1/", cfg.Port)
	if emailService != nil {
		log.Printf("📧 Email integration: ACTIVE (polling %s inbox)", cfg.Email.IMAPUsername)
	} else {
		log.Printf("📧 Email integration: DISABLED")
	}
	log.Println("==============================================")
	log.Fatal(http.ListenAndServe(":"+cfg.Port, mux))
}
