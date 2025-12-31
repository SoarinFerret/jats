package repository

import (
	"errors"
	"fmt"
	"time"

	"github.com/soarinferret/jats/internal/models"
	"gorm.io/gorm"
)

// AuthRepository handles authentication-related database operations
type AuthRepository struct {
	db *gorm.DB
}

// NewAuthRepository creates a new authentication repository
func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

// User operations

// CreateUser creates a new user
func (r *AuthRepository) CreateUser(user *models.User) error {
	if err := r.db.Create(user).Error; err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByUsername retrieves a user by username
func (r *AuthRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("username = ? AND is_active = ?", username, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *AuthRepository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ? AND is_active = ?", email, true).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *AuthRepository) GetUserByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.Where("is_active = ?", true).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}
	return &user, nil
}

// GetAllUsers retrieves all users (including inactive ones for admin purposes)
func (r *AuthRepository) GetAllUsers() ([]models.User, error) {
	var users []models.User
	if err := r.db.Order("username").Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get all users: %w", err)
	}
	return users, nil
}

// UpdateUser updates user information
func (r *AuthRepository) UpdateUser(user *models.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}

// UpdateUserLastLogin updates the user's last login time
func (r *AuthRepository) UpdateUserLastLogin(userID uint) error {
	now := time.Now()
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).Update("last_login_at", &now).Error; err != nil {
		return fmt.Errorf("failed to update user last login: %w", err)
	}
	return nil
}

// DeleteUser soft deletes a user
func (r *AuthRepository) DeleteUser(userID uint) error {
	if err := r.db.Delete(&models.User{}, userID).Error; err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// Session operations

// CreateSession creates a new session
func (r *AuthRepository) CreateSession(session *models.Session) error {
	if err := r.db.Create(session).Error; err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// GetSessionByToken retrieves a session by token
func (r *AuthRepository) GetSessionByToken(token string) (*models.Session, error) {
	var session models.Session
	if err := r.db.Preload("User").Where("token = ? AND expires_at > ?", token, time.Now()).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session by token: %w", err)
	}
	return &session, nil
}

// UpdateSessionLastUsed updates the session's last used time
func (r *AuthRepository) UpdateSessionLastUsed(sessionID uint) error {
	now := time.Now()
	if err := r.db.Model(&models.Session{}).Where("id = ?", sessionID).Update("last_used_at", now).Error; err != nil {
		return fmt.Errorf("failed to update session last used: %w", err)
	}
	return nil
}

// DeleteSession deletes a session
func (r *AuthRepository) DeleteSession(sessionID uint) error {
	if err := r.db.Delete(&models.Session{}, sessionID).Error; err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	return nil
}

// DeleteSessionByToken deletes a session by token
func (r *AuthRepository) DeleteSessionByToken(token string) error {
	if err := r.db.Where("token = ?", token).Delete(&models.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete session by token: %w", err)
	}
	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (r *AuthRepository) DeleteUserSessions(userID uint) error {
	if err := r.db.Where("user_id = ?", userID).Delete(&models.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

// DeleteExpiredSessions deletes all expired sessions
func (r *AuthRepository) DeleteExpiredSessions() error {
	if err := r.db.Where("expires_at <= ?", time.Now()).Delete(&models.Session{}).Error; err != nil {
		return fmt.Errorf("failed to delete expired sessions: %w", err)
	}
	return nil
}

// GetUserSessions retrieves all sessions for a user
func (r *AuthRepository) GetUserSessions(userID uint) ([]models.Session, error) {
	var sessions []models.Session
	if err := r.db.Where("user_id = ? AND expires_at > ?", userID, time.Now()).Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	return sessions, nil
}

// API Key operations

// CreateAPIKey creates a new API key
func (r *AuthRepository) CreateAPIKey(apiKey *models.APIKey) error {
	if err := r.db.Create(apiKey).Error; err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}
	return nil
}

// GetAPIKeyByHash retrieves an API key by its hash
func (r *AuthRepository) GetAPIKeyByHash(keyHash string) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.Preload("User").Where("key_hash = ? AND is_active = ?", keyHash, true).First(&apiKey).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get API key by hash: %w", err)
	}

	// Check if API key is expired
	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	return &apiKey, nil
}

// GetAPIKeyByID retrieves an API key by ID
func (r *AuthRepository) GetAPIKeyByID(id uint) (*models.APIKey, error) {
	var apiKey models.APIKey
	if err := r.db.Preload("User").First(&apiKey, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get API key by ID: %w", err)
	}
	return &apiKey, nil
}

// UpdateAPIKeyLastUsed updates the API key's last used time
func (r *AuthRepository) UpdateAPIKeyLastUsed(keyID uint) error {
	now := time.Now()
	if err := r.db.Model(&models.APIKey{}).Where("id = ?", keyID).Update("last_used_at", &now).Error; err != nil {
		return fmt.Errorf("failed to update API key last used: %w", err)
	}
	return nil
}

// UpdateAPIKey updates API key information
func (r *AuthRepository) UpdateAPIKey(apiKey *models.APIKey) error {
	if err := r.db.Save(apiKey).Error; err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}
	return nil
}

// DeleteAPIKey deletes an API key
func (r *AuthRepository) DeleteAPIKey(keyID uint) error {
	if err := r.db.Delete(&models.APIKey{}, keyID).Error; err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	return nil
}

// GetUserAPIKeys retrieves all API keys for a user
func (r *AuthRepository) GetUserAPIKeys(userID uint) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.db.Where("user_id = ?", userID).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get user API keys: %w", err)
	}
	return apiKeys, nil
}

// GetAPIKeysByPrefix retrieves all active API keys with a given prefix
func (r *AuthRepository) GetAPIKeysByPrefix(prefix string) ([]models.APIKey, error) {
	var apiKeys []models.APIKey
	if err := r.db.Preload("User").Where("key_prefix = ? AND is_active = ?", prefix, true).Find(&apiKeys).Error; err != nil {
		return nil, fmt.Errorf("failed to get API keys by prefix: %w", err)
	}
	return apiKeys, nil
}

// DeactivateAPIKey deactivates an API key
func (r *AuthRepository) DeactivateAPIKey(keyID uint) error {
	if err := r.db.Model(&models.APIKey{}).Where("id = ?", keyID).Update("is_active", false).Error; err != nil {
		return fmt.Errorf("failed to deactivate API key: %w", err)
	}
	return nil
}

// Login attempt operations

// CreateLoginAttempt creates a new login attempt record
func (r *AuthRepository) CreateLoginAttempt(attempt *models.LoginAttempt) error {
	if err := r.db.Create(attempt).Error; err != nil {
		return fmt.Errorf("failed to create login attempt: %w", err)
	}
	return nil
}

// GetRecentLoginAttempts gets recent login attempts for rate limiting
func (r *AuthRepository) GetRecentLoginAttempts(username, ipAddress string, since time.Time) (int64, error) {
	var count int64
	query := r.db.Model(&models.LoginAttempt{}).Where("created_at > ?", since)

	if username != "" {
		query = query.Where("username = ?", username)
	}
	if ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count recent login attempts: %w", err)
	}
	return count, nil
}

// GetFailedLoginAttempts gets failed login attempts for rate limiting
func (r *AuthRepository) GetFailedLoginAttempts(username, ipAddress string, since time.Time) (int64, error) {
	var count int64
	query := r.db.Model(&models.LoginAttempt{}).Where("success = ? AND created_at > ?", false, since)

	if username != "" {
		query = query.Where("username = ?", username)
	}
	if ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}

	if err := query.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("failed to count failed login attempts: %w", err)
	}
	return count, nil
}

// CleanupOldLoginAttempts removes old login attempt records
func (r *AuthRepository) CleanupOldLoginAttempts(before time.Time) error {
	if err := r.db.Where("created_at < ?", before).Delete(&models.LoginAttempt{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old login attempts: %w", err)
	}
	return nil
}
