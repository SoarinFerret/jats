package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/soarinferret/jats/internal/common"
	"github.com/soarinferret/jats/internal/middleware"
	"github.com/soarinferret/jats/internal/models"
	"github.com/soarinferret/jats/internal/services"
)

// AuthHandlers handles authentication-related HTTP endpoints
type AuthHandlers struct {
	authService *services.AuthService
}

// NewAuthHandlers creates a new auth handlers instance
func NewAuthHandlers(authService *services.AuthService) *AuthHandlers {
	return &AuthHandlers{
		authService: authService,
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

// TOTPSetupResponse represents the TOTP setup response
type TOTPSetupResponse struct {
	Secret          string `json:"secret"`
	ProvisioningURI string `json:"provisioning_uri"`
	QRCodeURL       string `json:"qr_code_url"`
}

// APIKeyRequest represents an API key creation request
type APIKeyRequest struct {
	Name        string     `json:"name"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// APIKeyResponse represents an API key creation response
type APIKeyResponse struct {
	APIKey *models.APIKey `json:"api_key"`
	Key    string         `json:"key"` // The actual key (only returned once)
}

// Register handles user registration
func (h *AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}
	
	user, err := h.authService.RegisterUser(req.Username, req.Email, req.Password)
	if err != nil {
		if err == services.ErrUserExists {
			common.SendErrorResponse(w, http.StatusConflict, "USER_EXISTS", "User already exists", nil)
			return
		}
		common.SendErrorResponse(w, http.StatusInternalServerError, "REGISTRATION_FAILED", err.Error(), nil)
		return
	}
	
	// Remove sensitive fields before returning
	user.HashedPassword = ""
	user.TOTPSecret = ""
	
	common.SendSuccessResponse(w, http.StatusCreated, user, "User registered successfully")
}

// Login handles user authentication
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}
	
	// Get client info
	userAgent := r.Header.Get("User-Agent")
	ipAddress := r.RemoteAddr
	
	loginReq := &services.LoginRequest{
		Username:  req.Username,
		Password:  req.Password,
		TOTPCode:  req.TOTPCode,
		UserAgent: userAgent,
		IPAddress: ipAddress,
	}
	
	result, err := h.authService.Login(loginReq)
	if err != nil {
		switch err {
		case services.ErrInvalidCredentials:
			common.SendErrorResponse(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid credentials", nil)
		case services.ErrInvalidTOTP:
			common.SendErrorResponse(w, http.StatusUnauthorized, "INVALID_TOTP", "Invalid TOTP code", nil)
		case services.ErrRateLimitExceeded:
			common.SendErrorResponse(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", "Too many login attempts", nil)
		default:
			common.SendErrorResponse(w, http.StatusInternalServerError, "LOGIN_FAILED", err.Error(), nil)
		}
		return
	}
	
	// If TOTP is required but not provided
	if result.RequiresTOTP {
		common.SendSuccessResponse(w, http.StatusOK, map[string]interface{}{
			"requires_totp": true,
		}, "TOTP verification required")
		return
	}
	
	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    result.Session.Token,
		Expires:  result.Session.ExpiresAt,
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	
	// Remove sensitive fields before returning
	result.User.HashedPassword = ""
	result.User.TOTPSecret = ""
	result.Session.Token = ""
	
	common.SendSuccessResponse(w, http.StatusOK, map[string]interface{}{
		"user":    result.User,
		"session": result.Session,
	}, "Login successful")
}

// Logout handles user logout
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Get session token from cookie or header
	var sessionToken string
	if cookie, err := r.Cookie("session_token"); err == nil {
		sessionToken = cookie.Value
	}
	
	if sessionToken != "" {
		h.authService.Logout(sessionToken)
	}
	
	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	
	common.SendSuccessResponse(w, http.StatusOK, nil, "Logout successful")
}

// GetProfile returns the current user's profile
func (h *AuthHandlers) GetProfile(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	// Remove sensitive fields
	user.HashedPassword = ""
	user.TOTPSecret = ""
	
	common.SendSuccessResponse(w, http.StatusOK, user, "Profile retrieved successfully")
}

// SetupTOTP initiates TOTP setup for a user
func (h *AuthHandlers) SetupTOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	secret, provisioningURI, err := h.authService.SetupTOTP(user.ID)
	if err != nil {
		common.SendErrorResponse(w, http.StatusInternalServerError, "TOTP_SETUP_FAILED", err.Error(), nil)
		return
	}
	
	// Generate QR code URL (you can use a service like qr-server.com or implement your own)
	qrCodeURL := "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=" + provisioningURI
	
	response := TOTPSetupResponse{
		Secret:          secret,
		ProvisioningURI: provisioningURI,
		QRCodeURL:       qrCodeURL,
	}
	
	common.SendSuccessResponse(w, http.StatusOK, response, "TOTP setup initiated")
}

// EnableTOTP enables TOTP for a user after verification
func (h *AuthHandlers) EnableTOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	var req struct {
		TOTPCode string `json:"totp_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}
	
	if err := h.authService.EnableTOTP(user.ID, req.TOTPCode); err != nil {
		if err == services.ErrInvalidTOTP {
			common.SendErrorResponse(w, http.StatusBadRequest, "INVALID_TOTP", "Invalid TOTP code", nil)
			return
		}
		common.SendErrorResponse(w, http.StatusInternalServerError, "TOTP_ENABLE_FAILED", err.Error(), nil)
		return
	}
	
	common.SendSuccessResponse(w, http.StatusOK, nil, "TOTP enabled successfully")
}

// DisableTOTP disables TOTP for a user
func (h *AuthHandlers) DisableTOTP(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	if err := h.authService.DisableTOTP(user.ID); err != nil {
		common.SendErrorResponse(w, http.StatusInternalServerError, "TOTP_DISABLE_FAILED", err.Error(), nil)
		return
	}
	
	common.SendSuccessResponse(w, http.StatusOK, nil, "TOTP disabled successfully")
}

// CreateAPIKey creates a new API key
func (h *AuthHandlers) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	var req APIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.SendErrorResponse(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body", nil)
		return
	}
	
	// Default permissions if none provided
	if len(req.Permissions) == 0 {
		req.Permissions = models.DefaultPermissions()
	}
	
	apiKeyRecord, apiKey, err := h.authService.CreateAPIKey(user.ID, req.Name, req.Permissions, req.ExpiresAt)
	if err != nil {
		common.SendErrorResponse(w, http.StatusInternalServerError, "API_KEY_CREATION_FAILED", err.Error(), nil)
		return
	}
	
	response := APIKeyResponse{
		APIKey: apiKeyRecord,
		Key:    apiKey,
	}
	
	common.SendSuccessResponse(w, http.StatusCreated, response, "API key created successfully")
}

// GetAPIKeys returns all API keys for the current user
func (h *AuthHandlers) GetAPIKeys(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	apiKeys, err := h.authService.GetUserAPIKeys(user.ID)
	if err != nil {
		common.SendErrorResponse(w, http.StatusInternalServerError, "FAILED_TO_GET_API_KEYS", err.Error(), nil)
		return
	}
	
	common.SendSuccessResponse(w, http.StatusOK, apiKeys, "API keys retrieved successfully")
}

// DeleteAPIKey deletes an API key
func (h *AuthHandlers) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	// Get key ID from URL parameter
	keyIDStr := r.URL.Query().Get("id")
	if keyIDStr == "" {
		common.SendErrorResponse(w, http.StatusBadRequest, "MISSING_PARAMETER", "API key ID is required", nil)
		return
	}
	
	keyID, err := strconv.ParseUint(keyIDStr, 10, 32)
	if err != nil {
		common.SendErrorResponse(w, http.StatusBadRequest, "INVALID_PARAMETER", "Invalid API key ID", nil)
		return
	}
	
	if err := h.authService.DeleteAPIKey(uint(keyID)); err != nil {
		common.SendErrorResponse(w, http.StatusInternalServerError, "API_KEY_DELETION_FAILED", err.Error(), nil)
		return
	}
	
	common.SendSuccessResponse(w, http.StatusOK, nil, "API key deleted successfully")
}

// GetSessions returns all active sessions for the current user
func (h *AuthHandlers) GetSessions(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	sessions, err := h.authService.GetUserSessions(user.ID)
	if err != nil {
		common.SendErrorResponse(w, http.StatusInternalServerError, "FAILED_TO_GET_SESSIONS", err.Error(), nil)
		return
	}
	
	// Remove tokens from response
	for i := range sessions {
		sessions[i].Token = ""
	}
	
	common.SendSuccessResponse(w, http.StatusOK, sessions, "Sessions retrieved successfully")
}

// LogoutAll logs out all sessions for the current user
func (h *AuthHandlers) LogoutAll(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetCurrentUser(r)
	if user == nil {
		common.SendErrorResponse(w, http.StatusUnauthorized, "NOT_AUTHENTICATED", "Not authenticated", nil)
		return
	}
	
	if err := h.authService.LogoutAllSessions(user.ID); err != nil {
		common.SendErrorResponse(w, http.StatusInternalServerError, "LOGOUT_ALL_FAILED", err.Error(), nil)
		return
	}
	
	// Clear current session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
	
	common.SendSuccessResponse(w, http.StatusOK, nil, "All sessions logged out successfully")
}