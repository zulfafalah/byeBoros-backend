package usecase

import (
	"fmt"
	"strings"

	"byeboros-backend/internal/adapter/http/model/response"
	"byeboros-backend/internal/adapter/repository"
)

type CategoryUsecase struct {
	sheetRepo *repository.SheetRepository
}

func NewCategoryUsecase(sheetRepo *repository.SheetRepository) *CategoryUsecase {
	return &CategoryUsecase{sheetRepo: sheetRepo}
}

func (u *CategoryUsecase) GetCategory(spreadsheetID string, sheetName string) (*response.CategoryResponse, error) {
	catRange := sheetName + "!A2:B"
	budgetRange := sheetName + "!E2:E3"

	catRows, err := u.sheetRepo.GetRangeValues(spreadsheetID, catRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories from spreadsheet: %w", err)
	}

	budgetRows, err := u.sheetRepo.GetRangeValues(spreadsheetID, budgetRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get budgets from spreadsheet: %w", err)
	}

	res := &response.CategoryResponse{
		Categories: make([]response.CategoryItem, 0),
	}

	if len(budgetRows) > 0 && len(budgetRows[0]) > 0 {
		res.MonthlyBudget = parseAmount(budgetRows[0][0])
	}
	if len(budgetRows) > 1 && len(budgetRows[1]) > 0 {
		res.DailyBudget = parseAmount(budgetRows[1][0])
	}

	for _, row := range catRows {
		if len(row) < 2 {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		if name == "" {
			continue
		}
		budget := parseAmount(row[1])

		res.Categories = append(res.Categories, response.CategoryItem{
			Name:   name,
			Budget: budget,
		})
	}

	return res, nil
}
