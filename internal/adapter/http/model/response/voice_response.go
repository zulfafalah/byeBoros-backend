package response

// VoiceTransactionResponse is returned for both /voice/transcribe and
// /voice/transaction. Category/TransactionAt are only populated once resolved.
type VoiceTransactionResponse struct {
	Message     string `json:"message,omitempty"`
	Text        string `json:"text"`
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
	Type        string `json:"type,omitempty"`
	Priority    string `json:"priority,omitempty"`
	Notes       string `json:"notes,omitempty"`
	Date        string `json:"date,omitempty"`
	Category    string `json:"category,omitempty"`
	Created     bool   `json:"created"`
}
