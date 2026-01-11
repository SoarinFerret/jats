package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/soarinferret/jats/internal/auth"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidTOTP        = errors.New("invalid TOTP code")
	ErrTOTPRequired       = errors.New("TOTP verification required")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
	ErrInvalidAPIKey      = errors.New("invalid API key")
	ErrSessionExpired     = errors.New("session expired")
)

// AuthService handles authentication business logic
type AuthService struct {
	authRepo *repository.AuthRepository
	config   *AuthConfig
}

// AuthConfig holds authentication service configuration
type AuthConfig struct {
	SessionDuration    time.Duration
	MaxLoginAttempts   int
	RateLimitWindow    time.Duration
	CleanupInterval    time.Duration
	APIKeyLength       int
	RequireTOTP        bool
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		SessionDuration:   24 * time.Hour,
		MaxLoginAttempts:  5,
		RateLimitWindow:   15 * time.Minute,
		CleanupInterval:   time.Hour,
		APIKeyLength:      32,
		RequireTOTP:       false,
	}
}

// NewAuthService creates a new authentication service
func NewAuthService(authRepo *repository.AuthRepository, config *AuthConfig) *AuthService {
	if config == nil {
		config = DefaultAuthConfig()
	}
	
	service := &AuthService{
		authRepo: authRepo,
		config:   config,
	}
	
	// Start cleanup routine
	go service.startCleanupRoutine()
	
	return service
}

// RegisterUser registers a new user
func (s *AuthService) RegisterUser(username, email, password string) (*models.User, error) {
	// Validate input
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("username is required")
	}
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("email is required")
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("password is required")
	}
	
	// Check if user exists
	existingUser, err := s.authRepo.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}
	
	// Check if email exists
	existingUser, err = s.authRepo.GetUserByEmail(email)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing email: %w", err)
	}
	if existingUser != nil {
		return nil, ErrUserExists
	}
	
	// Hash password
	hashedPassword, err := auth.HashPassword(password, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	
	// Create user
	user := &models.User{
		Username:       username,
		Email:          email,
		HashedPassword: hashedPassword,
		IsActive:       true,
	}
	
	if err := s.authRepo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	
	return user, nil
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string
	Password string
	TOTPCode string
	UserAgent string
	IPAddress string
}

// LoginResult represents a login result
type LoginResult struct {
	User    *models.User
	Session *models.Session
	RequiresTOTP bool
}

// Login authenticates a user and creates a session
func (s *AuthService) Login(req *LoginRequest) (*LoginResult, error) {
	// Check rate limiting
	if err := s.checkRateLimit(req.Username, req.IPAddress); err != nil {
		return nil, err
	}
	
	// Record login attempt
	attempt := &models.LoginAttempt{
		Username:  req.Username,
		IPAddress: req.IPAddress,
		Success:   false,
	}
	defer func() {
		s.authRepo.CreateLoginAttempt(attempt)
	}()
	
	// Get user
	user, err := s.authRepo.GetUserByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	
	// Verify password
	valid, err := auth.VerifyPassword(req.Password, user.HashedPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}
	if !valid {
		return nil, ErrInvalidCredentials
	}
	
	// Check TOTP if enabled
	if user.TOTPEnabled {
		if req.TOTPCode == "" {
			return &LoginResult{
				RequiresTOTP: true,
			}, nil
		}
		
		if !auth.ValidateTOTPCode(user.TOTPSecret, req.TOTPCode) {
			return nil, ErrInvalidTOTP
		}
	} else if s.config.RequireTOTP && req.TOTPCode == "" {
		return &LoginResult{
			RequiresTOTP: true,
		}, nil
	}
	
	// Create session
	sessionToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}
	
	session := &models.Session{
		UserID:     user.ID,
		Token:      sessionToken,
		UserAgent:  req.UserAgent,
		IPAddress:  req.IPAddress,
		ExpiresAt:  time.Now().Add(s.config.SessionDuration),
		LastUsedAt: time.Now(),
	}
	
	if err := s.authRepo.CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	
	// Update last login
	if err := s.authRepo.UpdateUserLastLogin(user.ID); err != nil {
		// Log error but don't fail login
		fmt.Printf("Failed to update last login: %v\n", err)
	}
	
	// Mark attempt as successful
	attempt.Success = true
	
	return &LoginResult{
		User:    user,
		Session: session,
	}, nil
}

// Logout invalidates a session
func (s *AuthService) Logout(sessionToken string) error {
	if sessionToken == "" {
		return errors.New("session token is required")
	}
	
	return s.authRepo.DeleteSessionByToken(sessionToken)
}

// ValidateSession validates a session token and returns the auth context
func (s *AuthService) ValidateSession(sessionToken string) (*models.AuthContext, error) {
	if sessionToken == "" {
		return nil, errors.New("session token is required")
	}
	
	session, err := s.authRepo.GetSessionByToken(sessionToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return nil, ErrSessionExpired
	}
	
	// Update last used time
	if err := s.authRepo.UpdateSessionLastUsed(session.ID); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to update session last used: %v\n", err)
	}
	
	// Determine permissions for session-based authentication
	var permissions []string
	if session.User.Username == "jats-admin" {
		// Admin user gets admin permissions
		permissions = models.AdminPermissions()
	} else {
		// Regular users get default permissions
		permissions = models.DefaultPermissions()
	}

	return &models.AuthContext{
		User:        &session.User,
		Session:     session,
		Permissions: permissions,
		AuthMethod:  "session",
	}, nil
}

// ValidateAPIKey validates an API key and returns the auth context
func (s *AuthService) ValidateAPIKey(apiKey string) (*models.AuthContext, error) {
	if apiKey == "" {
		return nil, errors.New("API key is required")
	}
	
	// Validate the API key by checking all keys with matching prefix
	storedKey, err := s.validateAPIKeyByVerification(apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to validate API key: %w", err)
	}
	if storedKey == nil {
		return nil, ErrInvalidAPIKey
	}
	
	// Update last used time
	if err := s.authRepo.UpdateAPIKeyLastUsed(storedKey.ID); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to update API key last used: %v\n", err)
	}
	
	return &models.AuthContext{
		User:        &storedKey.User,
		APIKey:      storedKey,
		Permissions: storedKey.Permissions,
		AuthMethod:  "api_key",
	}, nil
}

// CreateAPIKey creates a new API key for a user
func (s *AuthService) CreateAPIKey(userID uint, name string, permissions []string, expiresAt *time.Time) (*models.APIKey, string, error) {
	if name == "" {
		return nil, "", errors.New("API key name is required")
	}
	
	// Generate API key
	apiKey, err := auth.GenerateSecureToken(s.config.APIKeyLength)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}
	
	// Hash the API key for storage
	keyHash, err := auth.HashAPIKey(apiKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}
	
	// Create API key record
	keyRecord := &models.APIKey{
		UserID:      userID,
		Name:        name,
		KeyHash:     keyHash,
		KeyPrefix:   apiKey[:8], // Store first 8 chars for identification
		Permissions: permissions,
		IsActive:    true,
		ExpiresAt:   expiresAt,
	}
	
	if err := s.authRepo.CreateAPIKey(keyRecord); err != nil {
		return nil, "", fmt.Errorf("failed to create API key: %w", err)
	}
	
	return keyRecord, apiKey, nil
}

// SetupTOTP sets up TOTP for a user
func (s *AuthService) SetupTOTP(userID uint) (string, string, error) {
	user, err := s.authRepo.GetUserByID(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return "", "", ErrUserNotFound
	}
	
	// Generate TOTP secret and provisioning URI
	secret, provisioningURI, err := auth.GenerateTOTPSecret(user.Username, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	
	// Update user with TOTP secret (but don't enable yet)
	user.TOTPSecret = secret
	if err := s.authRepo.UpdateUser(user); err != nil {
		return "", "", fmt.Errorf("failed to update user: %w", err)
	}
	
	return secret, provisioningURI, nil
}

// EnableTOTP enables TOTP for a user after verifying the code
func (s *AuthService) EnableTOTP(userID uint, totpCode string) error {
	user, err := s.authRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	
	if user.TOTPSecret == "" {
		return errors.New("TOTP not set up for user")
	}
	
	// Verify TOTP code
	if !auth.ValidateTOTPCode(user.TOTPSecret, totpCode) {
		return ErrInvalidTOTP
	}
	
	// Enable TOTP
	user.TOTPEnabled = true
	if err := s.authRepo.UpdateUser(user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	return nil
}

// DisableTOTP disables TOTP for a user
func (s *AuthService) DisableTOTP(userID uint) error {
	user, err := s.authRepo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}
	
	// Disable TOTP and clear secret
	user.TOTPEnabled = false
	user.TOTPSecret = ""
	if err := s.authRepo.UpdateUser(user); err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	return nil
}

// checkRateLimit checks if the user/IP has exceeded rate limits
func (s *AuthService) checkRateLimit(username, ipAddress string) error {
	since := time.Now().Add(-s.config.RateLimitWindow)
	
	// Check failed attempts by username
	failedAttempts, err := s.authRepo.GetFailedLoginAttempts(username, "", since)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	
	if failedAttempts >= int64(s.config.MaxLoginAttempts) {
		return ErrRateLimitExceeded
	}
	
	// Check failed attempts by IP
	failedAttempts, err = s.authRepo.GetFailedLoginAttempts("", ipAddress, since)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	
	if failedAttempts >= int64(s.config.MaxLoginAttempts) {
		return ErrRateLimitExceeded
	}
	
	return nil
}

// startCleanupRoutine starts background cleanup tasks
func (s *AuthService) startCleanupRoutine() {
	ticker := time.NewTicker(s.config.CleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		// Clean up expired sessions
		if err := s.authRepo.DeleteExpiredSessions(); err != nil {
			fmt.Printf("Failed to cleanup expired sessions: %v\n", err)
		}
		
		// Clean up old login attempts (older than 24 hours)
		if err := s.authRepo.CleanupOldLoginAttempts(time.Now().Add(-24 * time.Hour)); err != nil {
			fmt.Printf("Failed to cleanup old login attempts: %v\n", err)
		}
	}
}

// GetUserSessions returns all active sessions for a user
func (s *AuthService) GetUserSessions(userID uint) ([]models.Session, error) {
	return s.authRepo.GetUserSessions(userID)
}

// GetUserAPIKeys returns all API keys for a user
func (s *AuthService) GetUserAPIKeys(userID uint) ([]models.APIKey, error) {
	return s.authRepo.GetUserAPIKeys(userID)
}

// DeleteAPIKey deletes an API key
func (s *AuthService) DeleteAPIKey(keyID uint) error {
	return s.authRepo.DeleteAPIKey(keyID)
}

// LogoutAllSessions logs out all sessions for a user
func (s *AuthService) LogoutAllSessions(userID uint) error {
	return s.authRepo.DeleteUserSessions(userID)
}

// GetUserByUsername gets a user by username
func (s *AuthService) GetUserByUsername(username string) (*models.User, error) {
	return s.authRepo.GetUserByUsername(username)
}

// ResetPassword resets a user's password
func (s *AuthService) ResetPassword(username, newPassword string) error {
	if strings.TrimSpace(username) == "" {
		return errors.New("username is required")
	}
	if strings.TrimSpace(newPassword) == "" {
		return errors.New("password is required")
	}

	// Get user
	user, err := s.authRepo.GetUserByUsername(username)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return ErrUserNotFound
	}

	// Hash new password
	hashedPassword, err := auth.HashPassword(newPassword, nil)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update user password
	user.HashedPassword = hashedPassword
	if err := s.authRepo.UpdateUser(user); err != nil {
		return fmt.Errorf("failed to update user password: %w", err)
	}

	// Invalidate all existing sessions for security
	if err := s.authRepo.DeleteUserSessions(user.ID); err != nil {
		// Log error but don't fail the password reset
		fmt.Printf("Warning: Failed to invalidate user sessions: %v\n", err)
	}

	return nil
}

// validateAPIKeyByVerification validates an API key by checking all keys with matching prefix
func (s *AuthService) validateAPIKeyByVerification(apiKey string) (*models.APIKey, error) {
	if len(apiKey) < 8 {
		return nil, nil
	}
	
	prefix := apiKey[:8]
	apiKeys, err := s.authRepo.GetAPIKeysByPrefix(prefix)
	if err != nil {
		return nil, err
	}
	
	// Check each API key to see if it matches
	for _, storedKey := range apiKeys {
		// Check if key is expired
		if storedKey.ExpiresAt != nil && storedKey.ExpiresAt.Before(time.Now()) {
			continue
		}
		
		// Verify the API key against the stored hash
		valid, err := auth.VerifyAPIKey(apiKey, storedKey.KeyHash)
		if err != nil {
			continue // Skip this key if verification fails
		}
		
		if valid {
			return &storedKey, nil
		}
	}
	
	return nil, nil
}