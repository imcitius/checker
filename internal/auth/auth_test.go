package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"checker/pkg/config"
)

func TestNewAuthManager_Disabled(t *testing.T) {
	cfg := &config.Config{}
	am, err := NewAuthManager(context.Background(), cfg)
	require.NoError(t, err)
	assert.False(t, am.Enabled())
}

func TestSessionJWT_RoundTrip(t *testing.T) {
	am := &AuthManager{
		jwtSecret: []byte("test-secret-32-bytes-long-enough"),
		enabled:   true,
	}

	token, err := am.createSessionJWT("user@example.com", "Test User")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	email, name, err := am.parseSessionJWT(token)
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", email)
	assert.Equal(t, "Test User", name)
}

func TestSessionJWT_Expired(t *testing.T) {
	am := &AuthManager{
		jwtSecret: []byte("test-secret-32-bytes-long-enough"),
		enabled:   true,
	}

	// Create an expired token manually
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-25 * time.Hour)),
			Issuer:    "checker",
		},
		Email: "user@example.com",
		Name:  "Test User",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(am.jwtSecret)
	require.NoError(t, err)

	_, _, err = am.parseSessionJWT(tokenString)
	assert.Error(t, err)
}

func TestSessionJWT_WrongSecret(t *testing.T) {
	am1 := &AuthManager{jwtSecret: []byte("secret-one")}
	am2 := &AuthManager{jwtSecret: []byte("secret-two")}

	token, err := am1.createSessionJWT("user@example.com", "Test User")
	require.NoError(t, err)

	_, _, err = am2.parseSessionJWT(token)
	assert.Error(t, err)
}

func TestExtractAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{"X-API-Key header", map[string]string{"X-API-Key": "my-key"}, "my-key"},
		{"Bearer token", map[string]string{"Authorization": "Bearer my-key"}, "my-key"},
		{"No key", map[string]string{}, ""},
		{"X-API-Key takes precedence", map[string]string{"X-API-Key": "key1", "Authorization": "Bearer key2"}, "key1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/", nil)
			for k, v := range tt.headers {
				c.Request.Header.Set(k, v)
			}
			assert.Equal(t, tt.expected, extractAPIKey(c))
		})
	}
}

func TestMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	am := &AuthManager{enabled: false}
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(am.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	c.Request = httptest.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, c.Request)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddleware_ValidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	am := &AuthManager{
		enabled: true,
		apiKeys: map[string]bool{"valid-key": true},
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(am.Middleware())
	r.GET("/test", func(c *gin.Context) {
		assert.Equal(t, "apikey", c.GetString("auth_type"))
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "valid-key")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddleware_InvalidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	am := &AuthManager{
		enabled: true,
		apiKeys: map[string]bool{"valid-key": true},
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(am.Middleware())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Key", "wrong-key")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestMiddleware_ValidSessionCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	am := &AuthManager{
		enabled:   true,
		mode:      AuthModeOIDC,
		jwtSecret: []byte("test-secret-32-bytes-long-enough"),
		apiKeys:   make(map[string]bool),
	}

	token, err := am.createSessionJWT("user@example.com", "Test User")
	require.NoError(t, err)

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(am.Middleware())
	r.GET("/test", func(c *gin.Context) {
		assert.Equal(t, "oidc", c.GetString("auth_type"))
		assert.Equal(t, "user@example.com", c.GetString("user_email"))
		assert.Equal(t, "Test User", c.GetString("user_name"))
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.AddCookie(&http.Cookie{Name: "checker_session", Value: token})
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMiddleware_NoAuth_BrowserRedirect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	am := &AuthManager{
		enabled:   true,
		jwtSecret: []byte("test-secret"),
		apiKeys:   make(map[string]bool),
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(am.Middleware())
	r.GET("/dashboard", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/dashboard", nil)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/login")
}

func TestMiddleware_NoAuth_APIUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	am := &AuthManager{
		enabled:   true,
		jwtSecret: []byte("test-secret"),
		apiKeys:   make(map[string]bool),
	}

	w := httptest.NewRecorder()
	_, r := gin.CreateTestContext(w)

	r.Use(am.Middleware())
	r.GET("/api/checks", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/api/checks", nil)
	req.Header.Set("Accept", "application/json")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
