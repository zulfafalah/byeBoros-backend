package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"byeboros-backend/config"
	"byeboros-backend/internal/domain/model"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// AuthUsecase handles Google OAuth and JWT operations
type AuthUsecase struct {
	oauthConfig *oauth2.Config
	jwtSecret   []byte
	frontendURL string
}

// JWTClaims represents the custom JWT claims
type JWTClaims struct {
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
	jwt.RegisteredClaims
}

// NewAuthUsecase creates a new AuthUsecase
func NewAuthUsecase(cfg *config.Config) *AuthUsecase {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &AuthUsecase{
		oauthConfig: oauthConfig,
		jwtSecret:   []byte(cfg.JWTSecret),
		frontendURL: cfg.FrontendURL,
	}
}

// GetGoogleLoginURL returns the Google OAuth consent URL
func (u *AuthUsecase) GetGoogleLoginURL(state string) string {
	return u.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// HandleGoogleCallback exchanges the auth code for a token and fetches user info
func (u *AuthUsecase) HandleGoogleCallback(ctx context.Context, code string) (*model.User, string, error) {
	// Exchange authorization code for token
	token, err := u.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, "", fmt.Errorf("failed to exchange token: %w", err)
	}

	// Fetch user info from Google
	user, err := u.fetchGoogleUser(token.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch user info: %w", err)
	}

	// Generate JWT
	jwtToken, err := u.GenerateJWT(user)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate JWT: %w", err)
	}

	return user, jwtToken, nil
}

// fetchGoogleUser fetches the user profile from Google's userinfo API
func (u *AuthUsecase) fetchGoogleUser(accessToken string) (*model.User, error) {
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var googleUser struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}

	if err := json.Unmarshal(body, &googleUser); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user info: %w", err)
	}

	return &model.User{
		Email:   googleUser.Email,
		Name:    googleUser.Name,
		Picture: googleUser.Picture,
	}, nil
}

// GenerateJWT generates a JWT token for the given user
func (u *AuthUsecase) GenerateJWT(user *model.User) (string, error) {
	claims := JWTClaims{
		Email:   user.Email,
		Name:    user.Name,
		Picture: user.Picture,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(u.jwtSecret)
}

// ValidateJWT validates a JWT token and returns the claims
func (u *AuthUsecase) ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return u.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GetFrontendURL returns the frontend URL
func (u *AuthUsecase) GetFrontendURL() string {
	return u.frontendURL
}
