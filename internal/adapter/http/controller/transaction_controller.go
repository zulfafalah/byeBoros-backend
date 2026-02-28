package controller

import (
	"net/http"

	"byeboros-backend/internal/adapter/http/model/request"
	"byeboros-backend/internal/usecase"

	"github.com/labstack/echo/v4"
)

// TransactionController handles transaction HTTP endpoints
type TransactionController struct {
	transactionUsecase *usecase.TransactionUsecase
}

// NewTransactionController creates a new TransactionController
func NewTransactionController(transactionUsecase *usecase.TransactionUsecase) *TransactionController {
	return &TransactionController{transactionUsecase: transactionUsecase}
}

// AddIncomeTransaction handles POST /api/income-transaction
func (h *TransactionController) AddIncomeTransaction(c echo.Context) error {
	spreadsheetID, ok := c.Get("spreadsheet_id").(string)
	if !ok || spreadsheetID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Spreadsheet ID not found in context",
		})
	}

	sheetName, ok := c.Get("sheet_name").(string)
	if !ok || sheetName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Sheet name not found in context",
		})
	}

	var req request.IncomeTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload: " + err.Error(),
		})
	}

	// Basic validation
	if req.Description == "" || req.Category == "" || req.TransactionAt == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description, category, and transaction_at are required",
		})
	}

	if err := request.ValidateTransactionAt(req.TransactionAt); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "amount must be greater than 0",
		})
	}

	// Extract email from JWT token (set by JWTMiddleware)
	createdBy, _ := c.Get("email").(string)

	if err := h.transactionUsecase.AddIncomeTransaction(spreadsheetID, sheetName, req, createdBy); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add income transaction: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Income transaction added successfully",
	})
}

// AddExpenseTransaction handles POST /api/transaction/expense
func (h *TransactionController) AddExpenseTransaction(c echo.Context) error {
	spreadsheetID, ok := c.Get("spreadsheet_id").(string)
	if !ok || spreadsheetID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Spreadsheet ID not found in context",
		})
	}

	sheetName, ok := c.Get("sheet_name").(string)
	if !ok || sheetName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Sheet name not found in context",
		})
	}

	var req request.ExpenseTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload: " + err.Error(),
		})
	}

	// Basic validation
	if req.Description == "" || req.Category == "" || req.TransactionAt == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description, category, and transaction_at are required",
		})
	}

	if err := request.ValidateTransactionAt(req.TransactionAt); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "amount must be greater than 0",
		})
	}

	// Extract email from JWT token (set by JWTMiddleware)
	createdBy, _ := c.Get("email").(string)

	if err := h.transactionUsecase.AddExpenseTransaction(spreadsheetID, sheetName, req, createdBy); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add expense transaction: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Expense transaction added successfully",
	})
}

// UpdateTransaction handles PUT /api/transaction
func (h *TransactionController) UpdateTransaction(c echo.Context) error {
	spreadsheetID, ok := c.Get("spreadsheet_id").(string)
	if !ok || spreadsheetID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Spreadsheet ID not found in context",
		})
	}

	sheetName, ok := c.Get("sheet_name").(string)
	if !ok || sheetName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Sheet name not found in context",
		})
	}

	var req request.UpdateTransactionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload: " + err.Error(),
		})
	}

	// Basic validation
	if req.ID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "id is required",
		})
	}

	if req.Type != "income" && req.Type != "expense" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "type must be either 'income' or 'expense'",
		})
	}

	if req.Description == "" || req.Category == "" || req.TransactionAt == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description, category, and transaction_at are required",
		})
	}

	if err := request.ValidateTransactionAt(req.TransactionAt); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	if req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "amount must be greater than 0",
		})
	}

	// Extract email from JWT token (set by JWTMiddleware)
	updatedBy, _ := c.Get("email").(string)

	if err := h.transactionUsecase.UpdateTransaction(spreadsheetID, sheetName, req, updatedBy); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to update transaction: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Transaction updated successfully",
	})
}

// ListTransaction returns the list of transactions with optional date and category filters
func (h *TransactionController) ListTransaction(c echo.Context) error {
	spreadsheetID, ok := c.Get("spreadsheet_id").(string)
	if !ok || spreadsheetID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Spreadsheet ID not found in context",
		})
	}
	sheetName, ok := c.Get("sheet_name").(string)
	if !ok || sheetName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Sheet name not found in context",
		})
	}

	dateFilter := c.QueryParam("date")
	categoryFilter := c.QueryParam("category")
	typeFilter := c.QueryParam("type")

	data, err := h.transactionUsecase.GetListTransaction(spreadsheetID, sheetName, dateFilter, categoryFilter, typeFilter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch transactions: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   data,
	})
}

// GetAnalysis fetches the financial analysis data with optional period filter
func (h *TransactionController) GetAnalysis(c echo.Context) error {
	spreadsheetID, ok := c.Get("spreadsheet_id").(string)
	if !ok || spreadsheetID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Spreadsheet ID not found in context",
		})
	}

	sheetName, ok := c.Get("sheet_name").(string)
	if !ok || sheetName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Sheet name not found in context",
		})
	}

	// Get period query parameter (default: "Month")
	period := c.QueryParam("period")
	if period == "" {
		period = "Month"
	}

	// Validate period
	validPeriods := []string{"Day", "Month", "3 Months", "6 Months", "Year"}
	isValid := false
	for _, vp := range validPeriods {
		if period == vp {
			isValid = true
			break
		}
	}
	if !isValid {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid period. Must be one of: Day, Month, 3 Months, 6 Months, Year",
		})
	}

	data, err := h.transactionUsecase.GetAnalysis(spreadsheetID, sheetName, period)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch analysis: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, data)
}
