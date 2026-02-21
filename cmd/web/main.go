package main

import (
	"log"

	"byeboros-backend/config"
	"byeboros-backend/internal/adapter/http/controller"
	"byeboros-backend/internal/adapter/http/middleware"
	"byeboros-backend/internal/adapter/repository"
	"byeboros-backend/internal/infrastructure/gsheet"
	"byeboros-backend/internal/usecase"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Google Sheets client
	sheetClient, err := gsheet.NewClient(cfg.GoogleServiceAccFile, cfg.SpreadsheetID)
	if err != nil {
		log.Printf("⚠️  Warning: Google Sheets client failed to initialize: %v", err)
		log.Println("   The server will start, but sheet operations will not work.")
	}

	// Initialize layers
	sheetRepo := repository.NewSheetRepository(sheetClient)
	authUsecase := usecase.NewAuthUsecase(cfg)

	// Controllers
	authController := controller.NewAuthController(authUsecase)

	// Setup Echo
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: []string{cfg.FrontendURL, "http://localhost:3000"},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Register routes
	setupRoutes(e, authController, authUsecase, middleware.JWTMiddleware(authUsecase))

	// Log initialized components
	_ = sheetRepo // will be injected into usecases as features are added

	// Start server
	log.Printf("🚀 Server starting on port %s", cfg.Port)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}

func setupRoutes(e *echo.Echo, authCtrl *controller.AuthController, authUsecase *usecase.AuthUsecase, jwtMiddleware echo.MiddlewareFunc) {
	// Health check
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status":  "ok",
			"message": "Byeboros Backend is running 🚀",
		})
	})

	// Auth routes (public)
	auth := e.Group("/auth")
	auth.GET("/google/login", authCtrl.GoogleLogin)
	auth.GET("/google/callback", authCtrl.GoogleCallback)

	// Protected routes
	api := e.Group("/api")
	api.Use(jwtMiddleware)
	api.GET("/me", authCtrl.GetMe)
}
