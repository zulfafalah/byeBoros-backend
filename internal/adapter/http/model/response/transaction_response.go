package response

type TransactionResponse struct {
	Transactions []TransactionGroupResponse `json:"transactions"`
}

type TransactionGroupResponse struct {
	GroupLabel string                    `json:"group_label"`
	GroupDate  string                    `json:"group_date"`
	Items      []TransactionItemResponse `json:"items"`
}

type TransactionItemResponse struct {
	ID              string  `json:"id"`
	TransactionName string  `json:"transaction_name"`
	Category        string  `json:"category"`
	Time            string  `json:"time"`
	Amount          float64 `json:"amount"`
	AmountDisplay   string  `json:"amount_display"`
	Type            string  `json:"type"`            // "expense" or "income"
	Label           string  `json:"label,omitempty"` // "PEMASUKAN" for income
}
