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
	if req.Description == "" || req.Category == "" || req.Priority == "" || req.TransactionAt == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "description, category, priority, and transaction_at are required",
		})
	}

	if req.Amount <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "amount must be greater than 0",
		})
	}

	if err := h.transactionUsecase.AddIncomeTransaction(spreadsheetID, sheetName, req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to add income transaction: " + err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Income transaction added successfully",
	})
}

// ListTransaction returns the list of transactions from sheet "Januari" A2:E
func (h *TransactionController) ListTransaction(c echo.Context) error {
	spreadsheetID, ok := c.Get("spreadsheet_id").(string)
	if !ok || spreadsheetID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Spreadsheet ID not found in context",
		})
	}

	data, err := h.transactionUsecase.GetListTransaction(spreadsheetID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch transactions: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "success",
		"data":    data,
	})
}
