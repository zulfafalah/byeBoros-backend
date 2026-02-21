package middleware

import (
	"net/http"
	"strings"

	"byeboros-backend/internal/usecase"

	"github.com/labstack/echo/v4"
)

// JWTMiddleware validates JWT tokens from the Authorization header
func JWTMiddleware(authUsecase *usecase.AuthUsecase) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Missing authorization header",
				})
			}

			// Expect "Bearer <token>"
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid authorization header format",
				})
			}

			tokenString := parts[1]

			claims, err := authUsecase.ValidateJWT(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid or expired token",
				})
			}

			// Set user info in context
			c.Set("email", claims.Email)
			c.Set("name", claims.Name)
			c.Set("picture", claims.Picture)

			return next(c)
		}
	}
}
