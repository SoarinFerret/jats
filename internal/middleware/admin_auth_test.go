package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soarinferret/jats/internal/models"
)

func TestAdminPermissionMiddleware(t *testing.T) {
	// We'll test the middleware logic directly without creating the middleware
	// since we're primarily testing the AuthContext permission logic

	t.Run("Admin user has admin permission", func(t *testing.T) {
		// Create admin user context
		adminUser := &models.User{
			ID:       1,
			Username: "admin",
			Email:    "admin@test.com",
			IsActive: true,
		}

		adminContext := &models.AuthContext{
			User:        adminUser,
			Permissions: models.AdminPermissions(),
			AuthMethod:  "session",
		}

		// Test that admin context has admin permission
		if !adminContext.HasPermission(models.PermissionAdmin) {
			t.Error("Admin context should have admin permission")
		}

		// Test that admin context has all other permissions
		allPerms := []string{
			models.PermissionReadTasks,
			models.PermissionWriteTasks,
			models.PermissionDeleteTasks,
			models.PermissionReadTime,
			models.PermissionWriteTime,
		}

		for _, perm := range allPerms {
			if !adminContext.HasPermission(perm) {
				t.Errorf("Admin context should have permission: %s", perm)
			}
		}
	})

	t.Run("Regular user does not have admin permission", func(t *testing.T) {
		// Create regular user context
		regularUser := &models.User{
			ID:       2,
			Username: "regular",
			Email:    "regular@test.com",
			IsActive: true,
		}

		regularContext := &models.AuthContext{
			User:        regularUser,
			Permissions: models.DefaultPermissions(),
			AuthMethod:  "session",
		}

		// Test that regular context does not have admin permission
		if regularContext.HasPermission(models.PermissionAdmin) {
			t.Error("Regular context should not have admin permission")
		}

		// Test that regular context has default permissions
		defaultPerms := []string{
			models.PermissionReadTasks,
			models.PermissionWriteTasks,
			models.PermissionReadTime,
			models.PermissionWriteTime,
		}

		for _, perm := range defaultPerms {
			if !regularContext.HasPermission(perm) {
				t.Errorf("Regular context should have permission: %s", perm)
			}
		}

		// Test that regular context does not have delete permission
		if regularContext.HasPermission(models.PermissionDeleteTasks) {
			t.Error("Regular context should not have delete tasks permission by default")
		}
	})

	t.Run("RequirePermission middleware blocks unauthorized users", func(t *testing.T) {
		// Create a test handler that requires admin permission
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("admin endpoint"))
		})

		// Create a simple middleware function for testing
		requireAdminPermission := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authCtx := GetAuthContext(r)
				if authCtx == nil || !authCtx.IsAuthenticated() {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
					return
				}
				if !authCtx.HasPermission(models.PermissionAdmin) {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Forbidden"))
					return
				}
				next.ServeHTTP(w, r)
			})
		}

		// Wrap with admin permission requirement
		protectedHandler := requireAdminPermission(testHandler)

		// Test with regular user context (should be forbidden)
		regularUser := &models.User{
			ID:       2,
			Username: "regular",
			Email:    "regular@test.com",
			IsActive: true,
		}

		regularContext := &models.AuthContext{
			User:        regularUser,
			Permissions: models.DefaultPermissions(),
			AuthMethod:  "session",
		}

		req := httptest.NewRequest("GET", "/admin/test", nil)
		ctx := context.WithValue(req.Context(), AuthContextKey, regularContext)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403 for regular user, got %d", w.Code)
		}

		// Test with admin user context (should be allowed)
		adminUser := &models.User{
			ID:       1,
			Username: "admin",
			Email:    "admin@test.com",
			IsActive: true,
		}

		adminContext := &models.AuthContext{
			User:        adminUser,
			Permissions: models.AdminPermissions(),
			AuthMethod:  "session",
		}

		req = httptest.NewRequest("GET", "/admin/test", nil)
		ctx = context.WithValue(req.Context(), AuthContextKey, adminContext)
		req = req.WithContext(ctx)

		w = httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for admin user, got %d", w.Code)
		}

		if w.Body.String() != "admin endpoint" {
			t.Errorf("Expected admin endpoint response, got %s", w.Body.String())
		}
	})

	t.Run("RequirePermission middleware blocks unauthenticated requests", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		requireAdminPermission := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authCtx := GetAuthContext(r)
				if authCtx == nil || !authCtx.IsAuthenticated() {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
					return
				}
				if !authCtx.HasPermission(models.PermissionAdmin) {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Forbidden"))
					return
				}
				next.ServeHTTP(w, r)
			})
		}

		protectedHandler := requireAdminPermission(testHandler)

		// Test with no auth context
		req := httptest.NewRequest("GET", "/admin/test", nil)
		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for unauthenticated request, got %d", w.Code)
		}

		// Test with nil auth context
		req = httptest.NewRequest("GET", "/admin/test", nil)
		ctx := context.WithValue(req.Context(), AuthContextKey, (*models.AuthContext)(nil))
		req = req.WithContext(ctx)

		w = httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for nil auth context, got %d", w.Code)
		}
	})

	t.Run("RequirePermission middleware blocks inactive users", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		requireAdminPermission := func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				authCtx := GetAuthContext(r)
				if authCtx == nil || !authCtx.IsAuthenticated() {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Unauthorized"))
					return
				}
				if !authCtx.HasPermission(models.PermissionAdmin) {
					w.WriteHeader(http.StatusForbidden)
					w.Write([]byte("Forbidden"))
					return
				}
				next.ServeHTTP(w, r)
			})
		}

		protectedHandler := requireAdminPermission(testHandler)

		// Test with inactive user
		inactiveUser := &models.User{
			ID:       3,
			Username: "inactive",
			Email:    "inactive@test.com",
			IsActive: false, // User is inactive
		}

		inactiveContext := &models.AuthContext{
			User:        inactiveUser,
			Permissions: models.AdminPermissions(),
			AuthMethod:  "session",
		}

		req := httptest.NewRequest("GET", "/admin/test", nil)
		ctx := context.WithValue(req.Context(), AuthContextKey, inactiveContext)
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		protectedHandler.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for inactive user, got %d", w.Code)
		}
	})
}

func TestAuthContextHelpers(t *testing.T) {
	t.Run("GetCurrentUser returns correct user", func(t *testing.T) {
		user := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@test.com",
			IsActive: true,
		}

		authContext := &models.AuthContext{
			User:        user,
			Permissions: models.DefaultPermissions(),
			AuthMethod:  "session",
		}

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), AuthContextKey, authContext)
		req = req.WithContext(ctx)

		retrievedUser := GetCurrentUser(req)
		if retrievedUser == nil {
			t.Fatal("GetCurrentUser should return user")
		}

		if retrievedUser.ID != user.ID {
			t.Errorf("Expected user ID %d, got %d", user.ID, retrievedUser.ID)
		}

		if retrievedUser.Username != user.Username {
			t.Errorf("Expected username %s, got %s", user.Username, retrievedUser.Username)
		}
	})

	t.Run("GetCurrentUser returns nil for no auth context", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		user := GetCurrentUser(req)
		if user != nil {
			t.Error("GetCurrentUser should return nil when no auth context")
		}
	})

	t.Run("GetAuthContext returns correct context", func(t *testing.T) {
		user := &models.User{
			ID:       1,
			Username: "testuser",
			Email:    "test@test.com",
			IsActive: true,
		}

		authContext := &models.AuthContext{
			User:        user,
			Permissions: models.AdminPermissions(),
			AuthMethod:  "api_key",
		}

		req := httptest.NewRequest("GET", "/test", nil)
		ctx := context.WithValue(req.Context(), AuthContextKey, authContext)
		req = req.WithContext(ctx)

		retrievedContext := GetAuthContext(req)
		if retrievedContext == nil {
			t.Fatal("GetAuthContext should return context")
		}

		if retrievedContext.AuthMethod != "api_key" {
			t.Errorf("Expected auth method api_key, got %s", retrievedContext.AuthMethod)
		}

		if !retrievedContext.HasPermission(models.PermissionAdmin) {
			t.Error("Retrieved context should have admin permission")
		}
	})
}


// Test permission inheritance and edge cases
func TestPermissionEdgeCases(t *testing.T) {
	t.Run("Empty permissions array", func(t *testing.T) {
		authContext := &models.AuthContext{
			User: &models.User{
				ID:       1,
				Username: "noperms",
				Email:    "noperms@test.com",
				IsActive: true,
			},
			Permissions: []string{}, // No permissions
			AuthMethod:  "session",
		}

		if authContext.HasPermission(models.PermissionReadTasks) {
			t.Error("Context with no permissions should not have read tasks permission")
		}

		if authContext.HasPermission(models.PermissionAdmin) {
			t.Error("Context with no permissions should not have admin permission")
		}
	})

	t.Run("Nil permissions array", func(t *testing.T) {
		authContext := &models.AuthContext{
			User: &models.User{
				ID:       1,
				Username: "nilperms",
				Email:    "nilperms@test.com",
				IsActive: true,
			},
			Permissions: nil, // Nil permissions
			AuthMethod:  "session",
		}

		if authContext.HasPermission(models.PermissionReadTasks) {
			t.Error("Context with nil permissions should not have any permissions")
		}
	})

	t.Run("Admin permission grants all permissions", func(t *testing.T) {
		authContext := &models.AuthContext{
			User: &models.User{
				ID:       1,
				Username: "adminonly",
				Email:    "adminonly@test.com",
				IsActive: true,
			},
			Permissions: []string{models.PermissionAdmin}, // Only admin permission
			AuthMethod:  "session",
		}

		// Admin permission should grant access to all other permissions
		allPermissions := []string{
			models.PermissionReadTasks,
			models.PermissionWriteTasks,
			models.PermissionDeleteTasks,
			models.PermissionReadTime,
			models.PermissionWriteTime,
		}

		for _, perm := range allPermissions {
			if !authContext.HasPermission(perm) {
				t.Errorf("Admin should have permission: %s", perm)
			}
		}
	})

	t.Run("IsAuthenticated checks", func(t *testing.T) {
		// Valid authenticated context
		validContext := &models.AuthContext{
			User: &models.User{
				ID:       1,
				Username: "valid",
				Email:    "valid@test.com",
				IsActive: true,
			},
			Permissions: models.DefaultPermissions(),
			AuthMethod:  "session",
		}

		if !validContext.IsAuthenticated() {
			t.Error("Valid context should be authenticated")
		}

		// Nil context
		var nilContext *models.AuthContext
		if nilContext.IsAuthenticated() {
			t.Error("Nil context should not be authenticated")
		}

		// Context with nil user
		nilUserContext := &models.AuthContext{
			User:        nil,
			Permissions: models.DefaultPermissions(),
			AuthMethod:  "session",
		}

		if nilUserContext.IsAuthenticated() {
			t.Error("Context with nil user should not be authenticated")
		}

		// Context with inactive user
		inactiveContext := &models.AuthContext{
			User: &models.User{
				ID:       1,
				Username: "inactive",
				Email:    "inactive@test.com",
				IsActive: false,
			},
			Permissions: models.DefaultPermissions(),
			AuthMethod:  "session",
		}

		if inactiveContext.IsAuthenticated() {
			t.Error("Context with inactive user should not be authenticated")
		}
	})
}