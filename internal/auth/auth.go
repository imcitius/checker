package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
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

// AuthMode represents the authentication method in use.
type AuthMode string

const (
	AuthModeNone     AuthMode = "none"
	AuthModePassword AuthMode = "password"
	AuthModeOIDC     AuthMode = "oidc"
)

// AuthManager handles authentication via static password, OIDC, or API keys.
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

	jwtSecret      []byte
	apiKeys        map[string]bool
	staticPassword string
	mode           AuthMode
	enabled        bool
	oidcReady      bool
}

// SessionClaims represents the claims stored in session JWTs.
type SessionClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
}

// NewAuthManager creates a new AuthManager.
// Priority: AUTH_PASSWORD (simplest) > OIDC > disabled.
func NewAuthManager(ctx context.Context, cfg *config.Config) (*AuthManager, error) {
	am := &AuthManager{
		apiKeys: make(map[string]bool),
		mode:    AuthModeNone,
	}

	// Load API keys (work in all modes)
	for _, key := range cfg.Auth.APIKeys {
		k := strings.TrimSpace(key)
		if k != "" {
			am.apiKeys[k] = true
		}
	}

	// Static password mode (takes priority over OIDC)
	if cfg.Auth.Password != "" {
		am.staticPassword = cfg.Auth.Password
		am.jwtSecret = []byte(cfg.Auth.JWTSecret)
		am.mode = AuthModePassword
		am.enabled = true
		logrus.Infof("Auth enabled: static password mode, %d API keys configured", len(am.apiKeys))
		return am, nil
	}

	// OIDC mode
	if cfg.Auth.OIDC.IssuerURL != "" {
		am.oidcIssuerURL = cfg.Auth.OIDC.IssuerURL
		am.oidcClientID = cfg.Auth.OIDC.ClientID
		am.oidcClientSecret = cfg.Auth.OIDC.ClientSecret
		am.oidcRedirectURL = cfg.Auth.OIDC.RedirectURL
		am.jwtSecret = []byte(cfg.Auth.JWTSecret)
		am.mode = AuthModeOIDC
		am.enabled = true

		logrus.Infof("Auth enabled: OIDC issuer=%s, %d API keys configured", cfg.Auth.OIDC.IssuerURL, len(am.apiKeys))

		initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		if err := am.initOIDC(initCtx); err != nil {
			logrus.Warnf("OIDC provider not available at startup (will retry in background): %v", err)
			go am.retryOIDCInit()
		}

		return am, nil
	}

	logrus.Info("Auth disabled (no password or OIDC issuer configured)")
	return am, nil
}

// Mode returns the current authentication mode.
func (am *AuthManager) Mode() AuthMode {
	return am.mode
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

		// 2. Check session cookie (works for both password and OIDC modes)
		if cookie, err := c.Cookie("checker_session"); err == nil && cookie != "" {
			email, name, err := am.parseSessionJWT(cookie)
			if err == nil {
				c.Set("auth_type", string(am.mode))
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

// HandleAuthMode returns the current auth mode so the frontend can render the right login form.
func (am *AuthManager) HandleAuthMode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"mode": string(am.mode)})
}

// HandleLogin handles the login flow depending on auth mode.
// Password mode: GET shows info, POST validates password and sets session cookie.
// OIDC mode: GET redirects to OIDC provider.
func (am *AuthManager) HandleLogin(c *gin.Context) {
	if !am.enabled {
		c.Redirect(http.StatusFound, "/")
		return
	}

	switch am.mode {
	case AuthModePassword:
		am.handlePasswordLogin(c)
	case AuthModeOIDC:
		am.handleOIDCLogin(c)
	default:
		c.Redirect(http.StatusFound, "/")
	}
}

func (am *AuthManager) handlePasswordLogin(c *gin.Context) {
	if c.Request.Method != http.MethodPost {
		// GET — frontend handles rendering, just confirm mode
		c.JSON(http.StatusOK, gin.H{"mode": "password"})
		return
	}

	// POST — validate password
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if subtle.ConstantTimeCompare([]byte(body.Password), []byte(am.staticPassword)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid password"})
		return
	}

	// Create session JWT
	sessionToken, err := am.createSessionJWT("admin@local", "Admin")
	if err != nil {
		logrus.Errorf("Failed to create session JWT: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create session"})
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("checker_session", sessionToken, 86400, "/", "", c.Request.TLS != nil, true)

	logrus.Info("User authenticated via static password")
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (am *AuthManager) handleOIDCLogin(c *gin.Context) {
	if !am.isOIDCReady() {
		c.String(http.StatusServiceUnavailable, "Authentication service is starting up, please try again in a moment")
		return
	}

	state, err := generateRandomState()
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to generate state")
		return
	}

	c.SetCookie("checker_auth_state", state, 300, "/", "", c.Request.TLS != nil, true)

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
	if !am.enabled || am.mode != AuthModeOIDC {
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

	c.SetCookie("checker_auth_state", "", -1, "/", "", c.Request.TLS != nil, true)

	am.mu.RLock()
	oauthCfg := am.oauth2Config
	verifier := am.verifier
	am.mu.RUnlock()

	token, err := oauthCfg.Exchange(c.Request.Context(), c.Query("code"))
	if err != nil {
		logrus.Errorf("OIDC token exchange failed: %v", err)
		c.String(http.StatusInternalServerError, "Token exchange failed")
		return
	}

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

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		logrus.Errorf("Failed to extract claims: %v", err)
		c.String(http.StatusInternalServerError, "Failed to extract claims")
		return
	}

	sessionToken, err := am.createSessionJWT(claims.Email, claims.Name)
	if err != nil {
		logrus.Errorf("Failed to create session JWT: %v", err)
		c.String(http.StatusInternalServerError, "Failed to create session")
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("checker_session", sessionToken, 86400, "/", "", c.Request.TLS != nil, true)

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
	if key := c.GetHeader("X-API-Key"); key != "" {
		return key
	}

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
