package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
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
	oidcProvider *oidc.Provider
	oauth2Config *oauth2.Config
	verifier     *oidc.IDTokenVerifier
	jwtSecret    []byte
	apiKeys      map[string]bool
	enabled      bool
}

// SessionClaims represents the claims stored in session JWTs.
type SessionClaims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Name  string `json:"name"`
}

// NewAuthManager creates a new AuthManager. If OIDC is not configured, auth is disabled (pass-through).
func NewAuthManager(ctx context.Context, cfg *config.Config) (*AuthManager, error) {
	am := &AuthManager{
		apiKeys: make(map[string]bool),
	}

	if cfg.Auth.OIDC.IssuerURL == "" {
		logrus.Info("Auth disabled (no OIDC issuer configured)")
		return am, nil
	}

	// Connect to OIDC provider
	provider, err := oidc.NewProvider(ctx, cfg.Auth.OIDC.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to OIDC provider %s: %w", cfg.Auth.OIDC.IssuerURL, err)
	}

	am.oidcProvider = provider
	am.oauth2Config = &oauth2.Config{
		ClientID:     cfg.Auth.OIDC.ClientID,
		ClientSecret: cfg.Auth.OIDC.ClientSecret,
		RedirectURL:  cfg.Auth.OIDC.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
	am.verifier = provider.Verifier(&oidc.Config{ClientID: cfg.Auth.OIDC.ClientID})
	am.jwtSecret = []byte(cfg.Auth.JWTSecret)
	am.enabled = true

	for _, key := range cfg.Auth.APIKeys {
		k := strings.TrimSpace(key)
		if k != "" {
			am.apiKeys[k] = true
		}
	}

	logrus.Infof("Auth enabled: OIDC issuer=%s, %d API keys configured", cfg.Auth.OIDC.IssuerURL, len(am.apiKeys))
	return am, nil
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

		// 2. Check session cookie
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

		// 3. Browser request → redirect to login
		accept := c.GetHeader("Accept")
		if strings.Contains(accept, "text/html") {
			redirectURL := c.Request.URL.RequestURI()
			c.Redirect(http.StatusFound, "/auth/login?redirect="+redirectURL)
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

	c.Redirect(http.StatusFound, am.oauth2Config.AuthCodeURL(state))
}

// HandleCallback processes the OIDC callback after user authentication.
func (am *AuthManager) HandleCallback(c *gin.Context) {
	if !am.enabled {
		c.Redirect(http.StatusFound, "/")
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

	// Exchange code for tokens
	token, err := am.oauth2Config.Exchange(c.Request.Context(), c.Query("code"))
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

	idToken, err := am.verifier.Verify(c.Request.Context(), rawIDToken)
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

// HandleLogout clears the session cookie and redirects to home.
func (am *AuthManager) HandleLogout(c *gin.Context) {
	c.SetCookie("checker_session", "", -1, "/", "", c.Request.TLS != nil, true)
	c.Redirect(http.StatusFound, "/")
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
