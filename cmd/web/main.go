package main

import (
	"log"

	"byeboros-backend/config"
	apphttp "byeboros-backend/internal/adapter/http"
	"byeboros-backend/internal/adapter/http/controller"
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
	sheetClient, err := gsheet.NewClient(cfg.GoogleServiceAccFile)
	if err != nil {
		log.Printf("⚠️  Warning: Google Sheets client failed to initialize: %v", err)
		log.Println("   The server will start, but sheet operations will not work.")
	}

	// Initialize layers
	sheetRepo := repository.NewSheetRepository(sheetClient)
	authUsecase := usecase.NewAuthUsecase(cfg)
	transactionUsecase := usecase.NewTransactionUsecase(sheetRepo)
	categoryUsecase := usecase.NewCategoryUsecase(sheetRepo)

	// Controllers
	authController := controller.NewAuthController(authUsecase)
	transactionController := controller.NewTransactionController(transactionUsecase)
	categoryController := controller.NewCategoryController(categoryUsecase)

	// Setup Echo
	e := echo.New()
	e.HideBanner = true

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins: cfg.AllowedOrigins,
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.PATCH},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Spreadsheet-ID", "X-Sheet-Name"},
	}))

	// Register routes
	apphttp.SetupRoutes(e, authController, transactionController, categoryController, authUsecase)

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	e.Logger.Fatal(e.Start(":" + cfg.Port))
}
