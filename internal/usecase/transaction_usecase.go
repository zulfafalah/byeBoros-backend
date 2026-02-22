package usecase

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"byeboros-backend/internal/adapter/http/model/request"
	"byeboros-backend/internal/adapter/http/model/response"
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

// GetListTransaction fetches transaction data from sheet A2:F (Expense) & H2:L (Income) and formats it
func (u *TransactionUsecase) GetListTransaction(spreadsheetID string, sheetName string, dateFilter string, categoryFilter string) (*response.TransactionResponse, error) {
	expenseRange := sheetName + "!A2:F"
	incomeRange := sheetName + "!H2:L"

	expenseRows, err := u.sheetRepo.GetRangeValues(spreadsheetID, expenseRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get expense transactions: %w", err)
	}

	incomeRows, err := u.sheetRepo.GetRangeValues(spreadsheetID, incomeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get income transactions: %w", err)
	}

	type rawItem struct {
		Item    response.TransactionItemResponse
		DateStr string
		Time    time.Time
	}
	var allItems []rawItem

	for i, row := range expenseRows {
		if len(row) < 6 {
			continue
		}
		desc := fmt.Sprintf("%v", row[0])
		cat := fmt.Sprintf("%v", row[1])
		amtF := parseAmount(row[3])
		t := parseDate(row[5])
		if t.IsZero() {
			continue
		}

		dateStr := t.Format("2006-01-02")
		if dateFilter != "" && dateStr != dateFilter {
			continue
		}
		if categoryFilter != "" && !strings.EqualFold(cat, categoryFilter) {
			continue
		}

		timeStr := t.Format("15:04")

		item := response.TransactionItemResponse{
			ID:              fmt.Sprintf("txn_exp_%d", i+1),
			TransactionName: desc,
			Category:        cat,
			Time:            timeStr,
			Amount:          -amtF,
			AmountDisplay:   formatAmount(amtF, false),
			Type:            "expense",
		}
		allItems = append(allItems, rawItem{item, dateStr, t})
	}

	for i, row := range incomeRows {
		if len(row) < 5 {
			continue
		}
		desc := fmt.Sprintf("%v", row[0])
		cat := fmt.Sprintf("%v", row[1])
		amtF := parseAmount(row[2])
		t := parseDate(row[4])
		if t.IsZero() {
			continue
		}

		dateStr := t.Format("2006-01-02")
		if dateFilter != "" && dateStr != dateFilter {
			continue
		}
		if categoryFilter != "" && !strings.EqualFold(cat, categoryFilter) {
			continue
		}

		timeStr := t.Format("15:04")

		item := response.TransactionItemResponse{
			ID:              fmt.Sprintf("txn_inc_%d", i+1),
			TransactionName: desc,
			Category:        cat,
			Time:            timeStr,
			Amount:          amtF,
			AmountDisplay:   formatAmount(amtF, true),
			Type:            "income",
			Label:           "PEMASUKAN",
		}
		allItems = append(allItems, rawItem{item, dateStr, t})
	}

	sort.Slice(allItems, func(i, j int) bool {
		return allItems[i].Time.After(allItems[j].Time)
	})

	groupsMap := make(map[string][]response.TransactionItemResponse)
	var order []string

	for _, raw := range allItems {
		if len(groupsMap[raw.DateStr]) == 0 {
			order = append(order, raw.DateStr)
		}
		groupsMap[raw.DateStr] = append(groupsMap[raw.DateStr], raw.Item)
	}

	finalGroups := []response.TransactionGroupResponse{}
	for _, dateStr := range order {
		finalGroups = append(finalGroups, response.TransactionGroupResponse{
			GroupLabel: getGroupLabel(dateStr),
			GroupDate:  dateStr,
			Items:      groupsMap[dateStr],
		})
	}

	return &response.TransactionResponse{Transactions: finalGroups}, nil
}

func parseAmount(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		s := strings.ReplaceAll(v, "Rp", "")
		s = strings.ReplaceAll(s, " ", "")
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
		f, _ := strconv.ParseFloat(s, 64)
		return f
	}
	return 0
}

func formatAmount(amount float64, isIncome bool) string {
	absAmt := int64(amount)
	if absAmt < 0 {
		absAmt = -absAmt
	}
	strAmt := strconv.FormatInt(absAmt, 10)
	var result []byte
	for i := range strAmt {
		if i > 0 && (len(strAmt)-i)%3 == 0 {
			result = append(result, '.')
		}
		result = append(result, strAmt[i])
	}
	if isIncome {
		return "+Rp " + string(result)
	}
	return "-Rp " + string(result)
}

func parseDate(val interface{}) time.Time {
	s, ok := val.(string)
	if !ok {
		return time.Time{}
	}
	formats := []string{
		"02/01/2006 15:04:05", // DD/MM/YYYY
		"2/1/2006 15:04:05",   // D/M/YYYY
		"1/2/2006 15:04:05",   // M/D/YYYY
		"01/02/2006 15:04:05", // MM/DD/YYYY
		"2006-01-02 15:04:05", // YYYY-MM-DD
		"02/01/2006 15:04",    // DD/MM/YYYY no secs
		"2/1/2006 15:04",      // D/M/YYYY no secs
		"1/2/2006 15:04",      // M/D/YYYY no secs
		"2006-01-02 15:04",    // YYYY-MM-DD no secs
		time.RFC3339,
	}

	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func getGroupLabel(dateStr string) string {
	now := time.Now()
	// Using Jakarta time as default assumption for ID based app
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err == nil {
		now = now.In(loc)
	}
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")

	if dateStr == today {
		return "Hari Ini"
	} else if dateStr == yesterday {
		return "Kemarin"
	}

	t, err := time.Parse("2006-01-02", dateStr)
	if err == nil {
		return t.Format("02 Jan 2006")
	}
	return dateStr
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

// AddExpenseTransaction inserts an expense transaction row into the sheet
func (u *TransactionUsecase) AddExpenseTransaction(spreadsheetID string, sheetName string, req request.ExpenseTransactionRequest) error {
	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	values := []interface{}{
		req.Description,   // Column H
		req.Category,      // Column I
		req.Amount,        // Column J
		notes,             // Column K
		req.TransactionAt, // Column L
	}

	if err := u.sheetRepo.AppendRow(spreadsheetID, sheetName+"!H:L", values); err != nil {
		return fmt.Errorf("failed to add expense transaction: %w", err)
	}

	return nil
}
