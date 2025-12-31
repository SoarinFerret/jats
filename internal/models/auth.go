package models

import (
	"time"

	"gorm.io/gorm"
)

// User represents a user account
type User struct {
	ID              uint           `json:"id" gorm:"primaryKey"`
	Username        string         `json:"username" gorm:"uniqueIndex;not null"`
	Email           string         `json:"email" gorm:"uniqueIndex;not null"`
	HashedPassword  string         `json:"-" gorm:"not null"` // Never return in JSON
	TOTPSecret      string         `json:"-" gorm:"column:totp_secret"` // Never return in JSON
	TOTPEnabled     bool           `json:"totp_enabled" gorm:"default:false"`
	IsActive        bool           `json:"is_active" gorm:"default:true"`
	LastLoginAt     *time.Time     `json:"last_login_at,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	
	// Relationships
	Sessions        []Session      `json:"-" gorm:"foreignKey:UserID"`
	APIKeys         []APIKey       `json:"-" gorm:"foreignKey:UserID"`
}

// UserRole represents user permissions (for future use)
type UserRole string

const (
	UserRoleAdmin  UserRole = "admin"
	UserRoleUser   UserRole = "user"
	UserRoleViewer UserRole = "viewer"
)

// Session represents a user session
type Session struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id" gorm:"not null;index"`
	Token        string         `json:"-" gorm:"uniqueIndex;not null"` // Never return in JSON
	UserAgent    string         `json:"user_agent,omitempty"`
	IPAddress    string         `json:"ip_address,omitempty"`
	ExpiresAt    time.Time      `json:"expires_at"`
	LastUsedAt   time.Time      `json:"last_used_at"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	
	// Relationships
	User         User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// APIKey represents an API key for programmatic access
type APIKey struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	Name        string         `json:"name" gorm:"not null"` // User-friendly name for the key
	KeyHash     string         `json:"-" gorm:"uniqueIndex;not null"` // Never return in JSON
	KeyPrefix   string         `json:"key_prefix" gorm:"not null"` // First 8 chars for identification
	Permissions []string       `json:"permissions" gorm:"serializer:json"` // JSON array of permissions
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	LastUsedAt  *time.Time     `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"` // Optional expiration
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	
	// Relationships
	User        User           `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// Permission constants
const (
	PermissionReadTasks   = "tasks:read"
	PermissionWriteTasks  = "tasks:write"
	PermissionDeleteTasks = "tasks:delete"
	PermissionReadTime    = "time:read"
	PermissionWriteTime   = "time:write"
	PermissionAdmin       = "admin:all"
)

// DefaultPermissions returns the default permissions for a new user
func DefaultPermissions() []string {
	return []string{
		PermissionReadTasks,
		PermissionWriteTasks,
		PermissionReadTime,
		PermissionWriteTime,
	}
}

// AdminPermissions returns all permissions
func AdminPermissions() []string {
	return []string{
		PermissionReadTasks,
		PermissionWriteTasks,
		PermissionDeleteTasks,
		PermissionReadTime,
		PermissionWriteTime,
		PermissionAdmin,
	}
}

// LoginAttempt represents a login attempt for rate limiting
type LoginAttempt struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"index;not null"`
	IPAddress string    `json:"ip_address" gorm:"index;not null"`
	Success   bool      `json:"success"`
	CreatedAt time.Time `json:"created_at"`
}

// AuthContext represents the authentication context
type AuthContext struct {
	User        *User    `json:"user,omitempty"`
	Session     *Session `json:"session,omitempty"`
	APIKey      *APIKey  `json:"api_key,omitempty"`
	Permissions []string `json:"permissions"`
	AuthMethod  string   `json:"auth_method"` // "session" or "api_key"
}

// HasPermission checks if the auth context has a specific permission
func (ac *AuthContext) HasPermission(permission string) bool {
	if ac == nil {
		return false
	}
	
	// Admin has all permissions
	for _, p := range ac.Permissions {
		if p == PermissionAdmin || p == permission {
			return true
		}
	}
	
	return false
}

// IsAuthenticated checks if the context represents an authenticated user
func (ac *AuthContext) IsAuthenticated() bool {
	return ac != nil && ac.User != nil && ac.User.IsActive
}