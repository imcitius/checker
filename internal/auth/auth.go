package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"checker/internal/config"
)

// AuthManager handles OIDC authentication, API key validation, and session management.
type AuthManager struct {
	// OIDC fields — guarded by mu for lazy initialization
	mu           sync.RWMutex
	oidcProvider *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier

	// OIDC config (saved for lazy init retries)
	oidcIssuerURL    string
	oidcClientID     string
	oidcClientSecret string
	oidcRedirectURL  string

	jwtSecret []byte
	apiKeys   map[string]bool
	enabled   bool
	oidcReady bool // true once OIDC provider is connected
}

// SessionClaims represents the claims stored in session JWTs.
type SessionClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
}

// NewAuthManager creates a new AuthManager. If OIDC is not configured, auth is disabled (pass-through).
// OIDC provider connection is attempted with a short timeout; if it fails, the app
// continues to start (healthcheck works, API keys work) and OIDC is retried in the background.
func NewAuthManager(ctx context.Context, cfg *config.Config) (*AuthManager, error) {
	am := &AuthManager{
		apiKeys: make(map[string]bool),
	}

	if cfg.Auth.OIDC.IssuerURL == "" {
		logrus.Info("Auth disabled (no OIDC issuer configured)")
		return am, nil
	}

	am.oidcIssuerURL = cfg.Auth.OIDC.IssuerURL
	am.oidcClientID = cfg.Auth.OIDC.ClientID
	am.oidcClientSecret = cfg.Auth.OIDC.ClientSecret
	am.oidcRedirectURL = cfg.Auth.OIDC.RedirectURL
	am.jwtSecret = []byte(cfg.Auth.JWTSecret)
	am.enabled = true

	for _, key := range cfg.Auth.APIKeys {
		k := strings.TrimSpace(key)
		if k != "" {
			am.apiKeys[k] = true
		}
	}

	logrus.Infof("Auth enabled: OIDC issuer=%s, %d API keys configured", cfg.Auth.OIDC.IssuerURL, len(am.apiKeys))

	// Try to connect to OIDC provider with a short timeout (don't block startup)
	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := am.initOIDC(initCtx); err != nil {
		logrus.Warnf("OIDC provider not available at startup (will retry in background): %v", err)
		go am.retryOIDCInit()
	}

	return am, nil
}

// initOIDC connects to the OIDC provider. Must be called with am.mu NOT held.
func (am *AuthManager) initOIDC(ctx context.Context) error {
	provider, err := oidc.NewProvider(ctx, am.oidcIssuerURL)
	if err != nil {
		return fmt.Errorf("failed to connect to OIDC provider %s: %w", am.oidcIssuerURL, err)
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	am.oidcProvider = provider
	am.oauth2Config = &oauth2.Config{
		ClientID:     am.oidcClientID,
		ClientSecret: am.oidcClientSecret,
		RedirectURL:  am.oidcRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
	am.verifier = provider.Verifier(&oidc.Config{ClientID: am.oidcClientID})
	am.oidcReady = true

	logrus.Info("OIDC provider connected successfully")
	return nil
}

// retryOIDCInit retries OIDC initialization in the background with exponential backoff.
func (am *AuthManager) retryOIDCInit() {
	backoff := 5 * time.Second
	maxBackoff := 2 * time.Minute

	for {
		time.Sleep(backoff)

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		err := am.initOIDC(ctx)
		cancel()

		if err == nil {
			return
		}

		logrus.Warnf("OIDC retry failed (next attempt in %s): %v", backoff, err)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// isOIDCReady returns whether OIDC is initialized.
func (am *AuthManager) isOIDCReady() bool {
	am.mu.RLock()
	defer am.mu.RUnlock()
	return am.oidcReady
}

// Enabled returns whether authentication is active.
func (am *AuthManager) Enabled() bool {
	return am.enabled
}

// Middleware returns a Gin middleware that enforces authentication.
func (am *AuthManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !am.enabled {
			c.Next()
			return
		}

		// 1. Check API key (Authorization: Bearer <key> or X-API-Key header)
		if key := extractAPIKey(c); key != "" {
			if am.apiKeys[key] {
				c.Set("auth_type", "apikey")
				c.Next()
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		// 2. Check session cookie (works even if OIDC is not ready — JWTs are self-contained)
		if cookie, err := c.Cookie("checker_session"); err == nil && cookie != "" {
			email, name, err := am.parseSessionJWT(cookie)
			if err == nil {
				c.Set("auth_type", "oidc")
				c.Set("user_email", email)
				c.Set("user_name", name)
				c.Next()
				return
			}
			logrus.Debugf("Invalid session cookie: %v", err)
		}

		// 3. Browser request → redirect to login page
		accept := c.GetHeader("Accept")
		if strings.Contains(accept, "text/html") {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 4. API request → 401
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

// HandleLogin initiates the OIDC authorization code flow.
func (am *AuthManager) HandleLogin(c *gin.Context) {
	if !am.enabled {
		c.Redirect(http.StatusFound, "/")
		return
	}

	if !am.isOIDCReady() {
		c.String(http.StatusServiceUnavailable, "Authentication service is starting up, please try again in a moment")
		return
	}

	state, err := generateRandomState()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate state")
		return
	}

	// Store state in cookie for CSRF verification
	c.SetCookie("checker_auth_state", state, 300, "/", "", c.Request.TLS != nil, true)

	// Store redirect target
	if redirect := c.Query("redirect"); redirect != "" {
		c.SetCookie("checker_auth_redirect", redirect, 300, "/", "", c.Request.TLS != nil, true)
	}

	am.mu.RLock()
	authURL := am.oauth2Config.AuthCodeURL(state)
	am.mu.RUnlock()

	c.Redirect(http.StatusFound, authURL)
}

// HandleCallback processes the OIDC callback after user authentication.
func (am *AuthManager) HandleCallback(c *gin.Context) {
	if !am.enabled {
		c.Redirect(http.StatusFound, "/")
		return
	}

	if !am.isOIDCReady() {
		c.String(http.StatusServiceUnavailable, "Authentication service is not ready")
		return
	}

	// Verify state
	stateCookie, err := c.Cookie("checker_auth_state")
	if err != nil || stateCookie == "" {
		c.String(http.StatusBadRequest, "Missing state cookie")
		return
	}
	if c.Query("state") != stateCookie {
		c.String(http.StatusBadRequest, "State mismatch")
		return
	}

	// Clear state cookie
	c.SetCookie("checker_auth_state", "", -1, "/", "", c.Request.TLS != nil, true)

	am.mu.RLock()
	oauthCfg := am.oauth2Config
	verifier := am.verifier
	am.mu.RUnlock()

	// Exchange code for tokens
	token, err := oauthCfg.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		logrus.Errorf("OIDC token exchange failed: %v", err)
		c.String(http.StatusInternalServerError, "Token exchange failed")
		return
	}

	// Extract and verify ID token
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		c.String(http.StatusInternalServerError, "No id_token in response")
		return
	}

	idToken, err := verifier.Verify(c.Request.Context(), rawIDToken)
	if err != nil {
		logrus.Errorf("ID token verification failed: %v", err)
		c.String(http.StatusInternalServerError, "Token verification failed")
		return
	}

	// Extract claims
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		logrus.Errorf("Failed to extract claims: %v", err)
		c.String(http.StatusInternalServerError, "Failed to extract claims")
		return
	}

	// Create session JWT
	sessionToken, err := am.createSessionJWT(claims.Email, claims.Name)
	if err != nil {
		logrus.Errorf("Failed to create session JWT: %v", err)
		c.String(http.StatusInternalServerError, "Failed to create session")
		return
	}

	// Set session cookie (24h)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("checker_session", sessionToken, 86400, "/", "", c.Request.TLS != nil, true)

	// Redirect to saved URL or home
	redirect := "/"
	if saved, err := c.Cookie("checker_auth_redirect"); err == nil && saved != "" {
		redirect = saved
		c.SetCookie("checker_auth_redirect", "", -1, "/", "", c.Request.TLS != nil, true)
	}

	logrus.Infof("User authenticated: %s (%s)", claims.Email, claims.Name)
	c.Redirect(http.StatusFound, redirect)
}

// HandleLogout clears the session cookie and redirects to login page.
func (am *AuthManager) HandleLogout(c *gin.Context) {
	c.SetCookie("checker_session", "", -1, "/", "", c.Request.TLS != nil, true)
	c.Redirect(http.StatusFound, "/login")
}

func (am *AuthManager) createSessionJWT(email, name string) (string, error) {
	claims := SessionClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "checker",
		},
		Email: email,
		Name:  name,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(am.jwtSecret)
}

func (am *AuthManager) parseSessionJWT(tokenString string) (email, name string, err error) {
	token, err := jwt.ParseWithClaims(tokenString, &SessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return am.jwtSecret, nil
	})
	if err != nil {
		return "", "", err
	}

	claims, ok := token.Claims.(*SessionClaims)
	if !ok || !token.Valid {
		return "", "", fmt.Errorf("invalid token claims")
	}

	return claims.Email, claims.Name, nil
}

func extractAPIKey(c *gin.Context) string {
	// Check X-API-Key header
	if key := c.GetHeader("X-API-Key"); key != "" {
		return key
	}

	// Check Authorization: Bearer <key>
	auth := c.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}

	return ""
}

func generateRandomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
