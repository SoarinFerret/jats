package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

var currentConfig *Config

type Config struct {
	ServerURL string `toml:"server_url"`
	Token     string `toml:"token"`
	Username  string `toml:"username"`
}

// Load loads configuration from file or creates default config
func Load(configFile string) (*Config, error) {
	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configFile = filepath.Join(home, ".jats.toml")
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// Create default config
		cfg := &Config{
			ServerURL: "http://localhost:8081",
		}
		
		// Try to save default config
		if err := Save(cfg, configFile); err != nil {
			// If we can't save, just return the default
			return cfg, nil
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// Save saves configuration to file
func Save(cfg *Config, configFile string) error {
	if configFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configFile = filepath.Join(home, ".jats.toml")
	}

	// Ensure directory exists
	dir := filepath.Dir(configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// SetCurrent sets the current global config
func SetCurrent(cfg *Config) {
	currentConfig = cfg
}

// GetCurrent returns the current global config
func GetCurrent() *Config {
	return currentConfig
}