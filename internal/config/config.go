package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Port       string      `toml:"port"`
	DBHost     string      `toml:"db_host"`
	DBPort     string      `toml:"db_port"`
	DBUser     string      `toml:"db_user"`
	DBPassword string      `toml:"db_password"`
	DBName     string      `toml:"db_name"`
	DBURL      string      `toml:"db_url"`
	Email      EmailConfig `toml:"email"`
}

type EmailConfig struct {
	// IMAP settings
	IMAPHost           string        `toml:"imap_host"`
	IMAPPort           string        `toml:"imap_port"`
	IMAPUsername       string        `toml:"imap_username"`
	IMAPPassword       string        `toml:"imap_password"`
	UseSSL             bool          `toml:"imap_use_ssl"`
	IMAPInsecure       bool          `toml:"imap_insecure_skip_verify"`
	InboxFolder        string        `toml:"imap_inbox_folder"`
	PollInterval       string `toml:"imap_poll_interval"`

	// SMTP settings
	SMTPHost           string `toml:"smtp_host"`
	SMTPPort           string `toml:"smtp_port"`
	SMTPAuth           bool   `toml:"smtp_auth"`
	SMTPUseTLS         bool   `toml:"smtp_use_tls"`
	SMTPInsecure       bool   `toml:"smtp_insecure_skip_verify"`
	SMTPUsername       string `toml:"smtp_username"`
	SMTPPassword       string `toml:"smtp_password"`
	FromName           string `toml:"smtp_from_name"`
	FromEmail          string `toml:"smtp_from_email"`
}

// LoadFromFile loads configuration from a TOML file, with environment variable fallbacks
func LoadFromFile(configPath string) (*Config, error) {
	// Start with default config
	config := getDefaultConfig()
	
	// If config file exists, load it
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		
		if err := toml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}
	}
	
	// Override with environment variables if they exist
	config.applyEnvOverrides()
	
	return config, nil
}

// Load loads configuration using environment variables only (backward compatibility)
func Load() *Config {
	config := getDefaultConfig()
	config.applyEnvOverrides()
	return config
}

// getDefaultConfig returns a config with default values
func getDefaultConfig() *Config {
	return &Config{
		Port:       "8080",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBUser:     "jats",
		DBPassword: "",
		DBName:     "jats",
		DBURL:      "",
		Email: EmailConfig{
			// IMAP settings
			IMAPHost:     "",
			IMAPPort:     "993",
			IMAPUsername: "",
			IMAPPassword: "",
			UseSSL:       true,
			IMAPInsecure: false,
			InboxFolder:  "INBOX",
			PollInterval: "5m",

			// SMTP settings
			SMTPHost:     "",
			SMTPPort:     "587",
			SMTPAuth:     true,
			SMTPUseTLS:   true,
			SMTPInsecure: false,
			SMTPUsername: "",
			SMTPPassword: "",
			FromName:     "JATS",
			FromEmail:    "",
		},
	}
}

// applyEnvOverrides applies environment variable overrides to the config
func (c *Config) applyEnvOverrides() {
	if val := os.Getenv("PORT"); val != "" {
		c.Port = val
	}
	if val := os.Getenv("DB_HOST"); val != "" {
		c.DBHost = val
	}
	if val := os.Getenv("DB_PORT"); val != "" {
		c.DBPort = val
	}
	if val := os.Getenv("DB_USER"); val != "" {
		c.DBUser = val
	}
	if val := os.Getenv("DB_PASSWORD"); val != "" {
		c.DBPassword = val
	}
	if val := os.Getenv("DB_NAME"); val != "" {
		c.DBName = val
	}
	if val := os.Getenv("DB_URL"); val != "" {
		c.DBURL = val
	}
	
	// Email IMAP settings
	if val := os.Getenv("IMAP_HOST"); val != "" {
		c.Email.IMAPHost = val
	}
	if val := os.Getenv("IMAP_PORT"); val != "" {
		c.Email.IMAPPort = val
	}
	if val := os.Getenv("IMAP_USERNAME"); val != "" {
		c.Email.IMAPUsername = val
	}
	if val := os.Getenv("IMAP_PASSWORD"); val != "" {
		c.Email.IMAPPassword = val
	}
	if val := os.Getenv("IMAP_USE_SSL"); val != "" {
		c.Email.UseSSL = getEnvBool("IMAP_USE_SSL", true)
	}
	if val := os.Getenv("IMAP_INSECURE_SKIP_VERIFY"); val != "" {
		c.Email.IMAPInsecure = getEnvBool("IMAP_INSECURE_SKIP_VERIFY", false)
	}
	if val := os.Getenv("IMAP_INBOX_FOLDER"); val != "" {
		c.Email.InboxFolder = val
	}
	if val := os.Getenv("IMAP_POLL_INTERVAL_MINUTES"); val != "" {
		if minutes := getEnvInt("IMAP_POLL_INTERVAL_MINUTES", 5); minutes > 0 {
			c.Email.PollInterval = fmt.Sprintf("%dm", minutes)
		}
	}
	if val := os.Getenv("IMAP_POLL_INTERVAL"); val != "" {
		c.Email.PollInterval = val
	}
	
	// Email SMTP settings
	if val := os.Getenv("SMTP_HOST"); val != "" {
		c.Email.SMTPHost = val
	}
	if val := os.Getenv("SMTP_PORT"); val != "" {
		c.Email.SMTPPort = val
	}
	if val := os.Getenv("SMTP_AUTH"); val != "" {
		c.Email.SMTPAuth = getEnvBool("SMTP_AUTH", true)
	}
	if val := os.Getenv("SMTP_USE_TLS"); val != "" {
		c.Email.SMTPUseTLS = getEnvBool("SMTP_USE_TLS", true)
	}
	if val := os.Getenv("SMTP_INSECURE_SKIP_VERIFY"); val != "" {
		c.Email.SMTPInsecure = getEnvBool("SMTP_INSECURE_SKIP_VERIFY", false)
	}
	if val := os.Getenv("SMTP_USERNAME"); val != "" {
		c.Email.SMTPUsername = val
	}
	if val := os.Getenv("SMTP_PASSWORD"); val != "" {
		c.Email.SMTPPassword = val
	}
	if val := os.Getenv("SMTP_FROM_NAME"); val != "" {
		c.Email.FromName = val
	}
	if val := os.Getenv("SMTP_FROM_EMAIL"); val != "" {
		c.Email.FromEmail = val
	}
}

func (c *Config) DatabaseURL() string {
	// If a custom database URL is provided, use it
	if c.DBURL != "" {
		return c.DBURL
	}
	
	// Otherwise, construct PostgreSQL URL from individual components
	return "postgres://" + c.DBUser + ":" + c.DBPassword + "@" + c.DBHost + ":" + c.DBPort + "/" + c.DBName + "?sslmode=disable"
}

// GetPollInterval parses the poll interval string and returns a time.Duration
func (c *Config) GetPollInterval() time.Duration {
	if c.Email.PollInterval == "" {
		return 5 * time.Minute // default
	}
	
	duration, err := time.ParseDuration(c.Email.PollInterval)
	if err != nil {
		// If parsing fails, try to parse as minutes (backward compatibility)
		if minutes, parseErr := strconv.Atoi(c.Email.PollInterval); parseErr == nil {
			return time.Duration(minutes) * time.Minute
		}
		return 5 * time.Minute // fallback to default
	}
	
	return duration
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}
