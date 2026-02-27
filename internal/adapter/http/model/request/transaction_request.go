package request

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
