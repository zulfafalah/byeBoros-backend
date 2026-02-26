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
func (u *TransactionUsecase) AddIncomeTransaction(spreadsheetID string, sheetName string, req request.IncomeTransactionRequest, createdBy string) error {
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
		createdBy,         // Column M
	}

	if err := u.sheetRepo.AppendRow(spreadsheetID, sheetName+"!H:M", values); err != nil {
		return fmt.Errorf("failed to add income transaction: %w", err)
	}

	return nil
}

// AddExpenseTransaction inserts an expense transaction row into the sheet
func (u *TransactionUsecase) AddExpenseTransaction(spreadsheetID string, sheetName string, req request.ExpenseTransactionRequest, createdBy string) error {
	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	values := []interface{}{
		req.Description,   // Column A
		req.Category,      // Column B
		"",                // Column C (Priority - not applicable for expense)
		req.Amount,        // Column D
		notes,             // Column E
		req.TransactionAt, // Column F
		createdBy,         // Column G
	}

	if err := u.sheetRepo.AppendRow(spreadsheetID, sheetName+"!A:G", values); err != nil {
		return fmt.Errorf("failed to add expense transaction: %w", err)
	}

	return nil
}

// GetAnalysis fetches the financial analysis data
func (u *TransactionUsecase) GetAnalysis(spreadsheetID string, sheetName string) (*response.AnalysisResponse, error) {
	ranges := []string{
		sheetName + "!AA2",  // 0: total expense
		sheetName + "!R2:S", // 1: exp categories
		sheetName + "!A2:G", // 2: exp priorities
		sheetName + "!AD2",  // 3: total income
		sheetName + "!I2:O", // 4: inc categories/transactions
		"Master Data!H4:H",  // 5: master income categories
	}

	valueRanges, err := u.sheetRepo.BatchGetValues(spreadsheetID, ranges)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch analysis data: %w", err)
	}

	getVal := func(idx int) [][]interface{} {
		if len(valueRanges) > idx && valueRanges[idx] != nil && valueRanges[idx].Values != nil {
			return valueRanges[idx].Values
		}
		return [][]interface{}{}
	}

	expenseData := u.getExpenseAnalysis(getVal(0), getVal(1), getVal(2))
	incomeData := u.getIncomeAnalysis(getVal(3), getVal(4), getVal(5))

	resp := &response.AnalysisResponse{
		Status: "success",
		Data: response.AnalysisData{
			Expense: expenseData,
			Income:  incomeData,
		},
	}
	return resp, nil
}

func (u *TransactionUsecase) getExpenseAnalysis(aa2, r2s, a2g [][]interface{}) response.AnalysisExpenseData {
	var totalSpent float64
	if len(aa2) > 0 && len(aa2[0]) > 0 {
		totalSpent = parseAmount(aa2[0][0])
	}

	expCats := []response.AnalysisCategory{}
	var expCatTotal float64
	for _, row := range r2s {
		if len(row) < 2 {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		if name == "" || strings.EqualFold(name, "Category") || strings.EqualFold(name, "Total") {
			// ignore headers if grabbed
			continue
		}
		amtF := parseAmount(row[1])
		expCats = append(expCats, response.AnalysisCategory{
			Name:   name,
			Amount: amtF,
		})
		expCatTotal += amtF
	}

	var topExpCat response.AnalysisTopCategory
	var maxExp float64
	for i, c := range expCats {
		if expCatTotal > 0 {
			expCats[i].Percent = int((c.Amount / expCatTotal) * 100)
		}
		if c.Amount >= maxExp && c.Amount > 0 {
			maxExp = c.Amount
			topExpCat = response.AnalysisTopCategory{
				Name:         c.Name,
				Total:        c.Amount,
				TotalDisplay: strings.Replace(formatAmount(c.Amount, false), "-", "", 1),
			}
		}
	}
	if topExpCat.Name == "" {
		topExpCat = response.AnalysisTopCategory{Name: "-", TotalDisplay: "Rp 0"}
	}

	priorityMap := make(map[string]float64)
	for i, row := range a2g {
		if i == 0 { // Skip header
			headerVal := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
			if strings.EqualFold(headerVal, "transaction_name") || strings.EqualFold(headerVal, "description") {
				continue
			}
		}
		if len(row) < 4 {
			continue
		}
		priStr := strings.TrimSpace(fmt.Sprintf("%v", row[2])) // C is index 2
		amtF := parseAmount(row[3])                            // D is index 3
		if priStr != "" {
			priorityMap[priStr] += amtF
		}
	}

	var priDist []response.AnalysisPriorityDistribution
	for pri, amt := range priorityMap {
		lvl := "other"
		lbl := pri
		if strings.EqualFold(pri, "tinggi") {
			lvl = "high"
			lbl = "High Priority"
		} else if strings.EqualFold(pri, "sedang") {
			lvl = "medium"
			lbl = "Medium Priority"
		} else if strings.EqualFold(pri, "rendah") {
			lvl = "low"
			lbl = "Low Priority"
		}
		priDist = append(priDist, response.AnalysisPriorityDistribution{
			Level:         lvl,
			Label:         lbl,
			Amount:        amt,
			AmountDisplay: strings.Replace(formatAmount(amt, false), "-", "", 1),
		})
	}
	sort.SliceStable(priDist, func(i, j int) bool {
		order := map[string]int{"high": 1, "medium": 2, "low": 3, "other": 4}
		return order[priDist[i].Level] < order[priDist[j].Level]
	})

	daysDivider := float64(time.Now().Day())
	if daysDivider == 0 {
		daysDivider = 1
	}
	dailyAvgExp := totalSpent / daysDivider

	return response.AnalysisExpenseData{
		Period:      "this_month",
		PeriodLabel: "This Month",
		Summary: response.AnalysisExpenseSummary{
			TotalSpent:        totalSpent,
			TotalSpentDisplay: strings.Replace(formatAmount(totalSpent, false), "-", "", 1),
		},
		Chart: response.AnalysisChart{
			Categories: expCats,
		},
		TopCategory: topExpCat,
		DailyAverage: response.AnalysisDailyAverage{
			Label:         "Average",
			Amount:        dailyAvgExp,
			AmountDisplay: strings.Replace(formatAmount(dailyAvgExp, false), "-", "", 1),
		},
		PriorityDistribution: priDist,
	}
}

func (u *TransactionUsecase) getIncomeAnalysis(ad2, incCatsData, masterIncData [][]interface{}) response.AnalysisIncomeData {
	var totalIncome float64
	if len(ad2) > 0 && len(ad2[0]) > 0 {
		totalIncome = parseAmount(ad2[0][0])
	}

	incRowMap := make(map[string]float64)
	for i, row := range incCatsData {
		if i == 0 {
			headerVal := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
			if strings.EqualFold(headerVal, "category") {
				continue
			}
		}
		if len(row) < 2 {
			continue
		}
		cat := strings.TrimSpace(fmt.Sprintf("%v", row[0])) // I is 0
		amtF := parseAmount(row[1])                         // J is 1
		if cat != "" {
			incRowMap[cat] += amtF
		}
	}

	masterIncSet := make(map[string]bool)
	incCats := []response.AnalysisCategory{}

	for _, row := range masterIncData {
		if len(row) == 0 {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		if name == "" || strings.EqualFold(name, "Kategori Pemasukan") {
			continue
		}
		amtF := incRowMap[name]
		incCats = append(incCats, response.AnalysisCategory{
			Name:   name,
			Amount: amtF,
		})
		masterIncSet[name] = true
	}

	for cat, amt := range incRowMap {
		if !masterIncSet[cat] {
			incCats = append(incCats, response.AnalysisCategory{
				Name:   cat,
				Amount: amt,
			})
		}
	}

	var incCatTotal float64
	for _, c := range incCats {
		incCatTotal += c.Amount
	}
	var topIncCat response.AnalysisTopCategory
	var maxInc float64
	for i, c := range incCats {
		if incCatTotal > 0 {
			incCats[i].Percent = int((c.Amount / incCatTotal) * 100)
		}
		if c.Amount >= maxInc && c.Amount > 0 {
			maxInc = c.Amount
			topIncCat = response.AnalysisTopCategory{
				Name:         c.Name,
				Total:        c.Amount,
				TotalDisplay: strings.Replace(formatAmount(c.Amount, true), "+", "", 1),
			}
		}
	}
	if topIncCat.Name == "" {
		topIncCat = response.AnalysisTopCategory{Name: "-", TotalDisplay: "Rp 0"}
	}

	daysDivider := float64(time.Now().Day())
	if daysDivider == 0 {
		daysDivider = 1
	}
	dailyAvgInc := totalIncome / daysDivider

	return response.AnalysisIncomeData{
		Period:      "this_month",
		PeriodLabel: "This Month",
		Summary: response.AnalysisIncomeSummary{
			TotalIncome:        totalIncome,
			TotalIncomeDisplay: strings.Replace(formatAmount(totalIncome, true), "+", "", 1),
		},
		Chart: response.AnalysisChart{
			Categories: incCats,
		},
		TopCategory: topIncCat,
		DailyAverage: response.AnalysisDailyAverage{
			Label:         "Average",
			Amount:        dailyAvgInc,
			AmountDisplay: strings.Replace(formatAmount(dailyAvgInc, true), "+", "", 1),
		},
	}
}
