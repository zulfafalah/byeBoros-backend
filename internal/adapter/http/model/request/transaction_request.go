package request

import (
	"fmt"
	"time"
)

// ValidateTransactionAt validates that transaction_at follows the format d/MM/yyyy H:mm:ss
// e.g. 27/02/2026 1:02:19 — day first, then month (not US M/d/yyyy).
func ValidateTransactionAt(transactionAt string) error {
	_, err := time.Parse("2/1/2006 15:4:5", transactionAt)
	if err != nil {
		return fmt.Errorf("transaction_at must use format d/MM/yyyy H:mm:ss (e.g. 27/02/2026 1:02:19)")
	}
	return nil
}

// IncomeTransactionRequest represents the payload for adding an income transaction
type IncomeTransactionRequest struct {
	Description   string  `json:"description" validate:"required"`
	Category      string  `json:"category" validate:"required"`
	Amount        float64 `json:"amount" validate:"required"`
	Notes         *string `json:"notes"`
	TransactionAt string  `json:"transaction_at" validate:"required"`
}

// ExpenseTransactionRequest represents the payload for adding an expense transaction
type ExpenseTransactionRequest struct {
	Description   string  `json:"description" validate:"required"`
	Category      string  `json:"category" validate:"required"`
	Priority      string  `json:"priority"`
	Amount        float64 `json:"amount" validate:"required"`
	Notes         *string `json:"notes"`
	TransactionAt string  `json:"transaction_at" validate:"required"`
}

// UpdateTransactionRequest represents the payload for updating a transaction
type UpdateTransactionRequest struct {
	ID            string  `json:"id" validate:"required"`
	Type          string  `json:"type" validate:"required,oneof=income expense"`
	Description   string  `json:"description" validate:"required"`
	Category      string  `json:"category" validate:"required"`
	Priority      string  `json:"priority"`
	Amount        float64 `json:"amount" validate:"required"`
	Notes         *string `json:"notes"`
	TransactionAt string  `json:"transaction_at" validate:"required"`
}
