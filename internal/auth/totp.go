package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// TOTPConfig holds TOTP configuration
type TOTPConfig struct {
	Issuer      string
	AccountName string
	SecretSize  int
}

// DefaultTOTPConfig returns default TOTP configuration for JATS
func DefaultTOTPConfig() *TOTPConfig {
	return &TOTPConfig{
		Issuer:     "JATS Task Management",
		SecretSize: 32,
	}
}

// GenerateTOTPSecret generates a new TOTP secret for a user
func GenerateTOTPSecret(username string, config *TOTPConfig) (string, string, error) {
	if config == nil {
		config = DefaultTOTPConfig()
	}

	// Generate random secret
	secret := make([]byte, config.SecretSize)
	_, err := rand.Read(secret)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate secret: %w", err)
	}

	// Encode secret as base32
	secretBase32 := base32.StdEncoding.EncodeToString(secret)

	// Generate provisioning URI for QR code
	key, err := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s",
		config.Issuer, username, secretBase32, config.Issuer))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	provisioningURI := key.URL()

	return secretBase32, provisioningURI, nil
}

// ValidateTOTPCode validates a TOTP code against a secret
func ValidateTOTPCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateTOTPCode generates a TOTP code for a given secret (useful for testing)
func GenerateTOTPCode(secret string) (string, error) {
	return totp.GenerateCode(secret, time.Now())
}

// BackupCodes represents backup codes for TOTP
type BackupCodes struct {
	Codes []string `json:"codes"`
}

// GenerateBackupCodes generates backup codes for TOTP recovery
func GenerateBackupCodes() (*BackupCodes, error) {
	const numCodes = 10
	const codeLength = 8
	
	codes := make([]string, numCodes)
	for i := 0; i < numCodes; i++ {
		code, err := generateRandomCode(codeLength)
		if err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = code
	}
	
	return &BackupCodes{Codes: codes}, nil
}

// generateRandomCode generates a random alphanumeric code
func generateRandomCode(length int) (string, error) {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, length)
	
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	
	return string(bytes), nil
}