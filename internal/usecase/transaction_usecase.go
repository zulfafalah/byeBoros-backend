package usecase

import (
	"fmt"

	"byeboros-backend/internal/adapter/http/model/request"
	"byeboros-backend/internal/adapter/repository"
)

// TransactionUsecase handles transaction business logic
type TransactionUsecase struct {
	sheetRepo *repository.SheetRepository
}

// NewTransactionUsecase creates a new TransactionUsecase
func NewTransactionUsecase(sheetRepo *repository.SheetRepository) *TransactionUsecase {
	return &TransactionUsecase{sheetRepo: sheetRepo}
}

// GetListTransaction fetches transaction data from sheet "Januari" range A2:E
func (u *TransactionUsecase) GetListTransaction(spreadsheetID string) ([][]interface{}, error) {
	rangeStr := "Januari!A2:E"

	resp, err := u.sheetRepo.GetRangeValues(spreadsheetID, rangeStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return resp, nil
}

// AddIncomeTransaction inserts an income transaction row into the "Income" sheet
func (u *TransactionUsecase) AddIncomeTransaction(spreadsheetID string, sheetName string, req request.IncomeTransactionRequest) error {
	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	values := []interface{}{
		req.Description,   // Column A
		req.Category,      // Column B
		req.Priority,      // Column C
		req.Amount,        // Column D
		notes,             // Column E
		req.TransactionAt, // Column F
	}

	if err := u.sheetRepo.AppendRow(spreadsheetID, sheetName+"!A:F", values); err != nil {
		return fmt.Errorf("failed to add income transaction: %w", err)
	}

	return nil
}
