package usecase

import (
	"fmt"
	"strings"

	"byeboros-backend/internal/adapter/http/model/request"
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
	catRange := sheetName + "!A4:C"
	budgetRange := sheetName + "!F4:F5"

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
		res.DailyBudget = parseAmount(budgetRows[0][0])
	}
	if len(budgetRows) > 1 && len(budgetRows[1]) > 0 {
		res.MonthlyBudget = parseAmount(budgetRows[1][0])
	}

	for _, row := range catRows {
		if len(row) < 2 {
			continue
		}
		category_name := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		if category_name == "" {
			continue
		}

		sub_category_name := strings.TrimSpace(fmt.Sprintf("%v", row[1]))
		if sub_category_name == "" {
			continue
		}

		budget := parseAmount(row[2])

		res.Categories = append(res.Categories, response.CategoryItem{
			CategoryName:    category_name,
			SubCategoryName: sub_category_name,
			Budget:          budget,
		})
	}

	return res, nil
}

func (u *CategoryUsecase) GetIncomeCategory(spreadsheetID string, sheetName string) ([]string, error) {
	incomeRange := sheetName + "!H4:H"

	rows, err := u.sheetRepo.GetRangeValues(spreadsheetID, incomeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get income categories from spreadsheet: %w", err)
	}

	categories := make([]string, 0)
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		if name == "" {
			continue
		}
		categories = append(categories, name)
	}

	return categories, nil
}

func (u *CategoryUsecase) SaveCategory(spreadsheetID string, sheetName string, req *request.SaveCategoryRequest) error {
	err := u.sheetRepo.UpdateCell(spreadsheetID, sheetName, 4, 5, req.DailyBudget)
	if err != nil {
		return err
	}

	err = u.sheetRepo.UpdateCell(spreadsheetID, sheetName, 5, 5, req.MonthlyBudget)
	if err != nil {
		return err
	}

	err = u.sheetRepo.ClearRange(spreadsheetID, sheetName+"!A4:C")
	if err != nil {
		return fmt.Errorf("failed to clear existing categories: %w", err)
	}

	if len(req.Categories) > 0 {
		var values [][]interface{}
		for _, cat := range req.Categories {
			values = append(values, []interface{}{cat.CategoryName, cat.SubCategoryName, cat.Budget})
		}
		err = u.sheetRepo.UpdateRange(spreadsheetID, sheetName+"!A4:C", values)
		if err != nil {
			return fmt.Errorf("failed to save new categories: %w", err)
		}
	}

	return nil
}
