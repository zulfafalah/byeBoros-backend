package controller

import (
	"net/http"

	"byeboros-backend/internal/adapter/http/model/request"
	"byeboros-backend/internal/usecase"

	"github.com/labstack/echo/v4"
)

type CategoryController struct {
	categoryUsecase *usecase.CategoryUsecase
}

func NewCategoryController(categoryUsecase *usecase.CategoryUsecase) *CategoryController {
	return &CategoryController{categoryUsecase: categoryUsecase}
}

func (h *CategoryController) ListCategory(c echo.Context) error {
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

	data, err := h.categoryUsecase.GetCategory(spreadsheetID, sheetName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch categories: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, data)
}

func (h *CategoryController) SaveCategory(c echo.Context) error {
	spreadsheetID, ok := c.Get("spreadsheet_id").(string)
	if !ok || spreadsheetID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Spreadsheet ID not found in context"})
	}

	sheetName, ok := c.Get("sheet_name").(string)
	if !ok || sheetName == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Sheet name not found in context"})
	}

	var req request.SaveCategoryRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	if err := h.categoryUsecase.SaveCategory(spreadsheetID, sheetName, &req); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Category saved successfully",
		"data":    req,
	})
}

func (h *CategoryController) ListIncomeCategory(c echo.Context) error {
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

	categories, err := h.categoryUsecase.GetIncomeCategory(spreadsheetID, sheetName)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to fetch income categories: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"categories": categories,
	})
}
