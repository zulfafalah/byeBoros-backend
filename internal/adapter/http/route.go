package http

import (
	"byeboros-backend/internal/adapter/http/controller"
	"byeboros-backend/internal/adapter/http/middleware"
	"byeboros-backend/internal/usecase"

	"github.com/labstack/echo/v4"
)

// SetupRoutes registers all application routes
func SetupRoutes(e *echo.Echo, authCtrl *controller.AuthController, transactionCtrl *controller.TransactionController, categoryCtrl *controller.CategoryController, authUsecase *usecase.AuthUsecase) {
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
	api.Use(middleware.JWTMiddleware(authUsecase))
	api.Use(middleware.SpreadsheetIDMiddleware())
	api.GET("/me", authCtrl.GetMe)

	// Transaction routes
	api.POST("/transaction/income", transactionCtrl.AddIncomeTransaction)
	api.POST("/transaction/expense", transactionCtrl.AddExpenseTransaction)
	api.GET("/transaction", transactionCtrl.ListTransaction)

	// Category routes
	api.GET("/category", categoryCtrl.ListCategory)
	api.POST("/category", categoryCtrl.SaveCategory)
	api.PUT("/category", categoryCtrl.SaveCategory)

	// Analysis routes
	api.GET("/analysis", transactionCtrl.GetAnalysis)

}
