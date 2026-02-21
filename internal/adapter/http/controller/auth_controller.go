package controller

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"byeboros-backend/internal/usecase"

	"github.com/labstack/echo/v4"
)

// AuthController handles authentication HTTP endpoints
type AuthController struct {
	authUsecase *usecase.AuthUsecase
}

// NewAuthController creates a new AuthController
func NewAuthController(authUsecase *usecase.AuthUsecase) *AuthController {
	return &AuthController{authUsecase: authUsecase}
}

// GoogleLogin redirects the user to Google's OAuth consent screen
func (h *AuthController) GoogleLogin(c echo.Context) error {
	// Generate a random state for CSRF protection
	state, err := generateRandomState()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate state",
		})
	}

	url := h.authUsecase.GetGoogleLoginURL(state)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

// GoogleCallback handles the OAuth callback from Google
func (h *AuthController) GoogleCallback(c echo.Context) error {
	code := c.QueryParam("code")
	if code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Authorization code not found",
		})
	}

	user, token, err := h.authUsecase.HandleGoogleCallback(c.Request().Context(), code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to authenticate: " + err.Error(),
		})
	}

	// Redirect to frontend with token
	frontendURL := h.authUsecase.GetFrontendURL()
	redirectURL := frontendURL + "?token=" + token
	_ = user // user info is encoded in the JWT

	return c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

// GetMe returns the currently authenticated user
func (h *AuthController) GetMe(c echo.Context) error {
	// User info is set by the JWT middleware
	email := c.Get("email")
	name := c.Get("name")
	picture := c.Get("picture")

	return c.JSON(http.StatusOK, map[string]interface{}{
		"email":   email,
		"name":    name,
		"picture": picture,
	})
}

// generateRandomState generates a random hex string for OAuth state
func generateRandomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
