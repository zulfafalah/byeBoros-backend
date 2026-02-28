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

// GetListTransaction fetches transaction data from sheet A2:G (Expense) & I2:N (Income) and formats it
func (u *TransactionUsecase) GetListTransaction(spreadsheetID string, sheetName string, dateFilter string, categoryFilter string, typeFilter string) (*response.TransactionResponse, error) {
	expenseRange := sheetName + "!A2:G"
	incomeRange := sheetName + "!I2:N"

	expenseRows, err := u.sheetRepo.GetRangeValues(spreadsheetID, expenseRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get expense transactions: %w", err)
	}

	incomeRows, err := u.sheetRepo.GetRangeValues(spreadsheetID, incomeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get income transactions: %w", err)
	}

	// DEBUG: Log raw income data
	fmt.Printf("DEBUG incomeRows count: %d\n", len(incomeRows))
	for i, row := range incomeRows {
		fmt.Printf("DEBUG incomeRow[%d] len=%d data=%v\n", i, len(row), row)
	}

	type rawItem struct {
		Item    response.TransactionItemResponse
		DateStr string
		Time    time.Time
	}
	var allItems []rawItem

	// Skip processing expense rows if type filter is set to "income"
	if typeFilter == "" || typeFilter == "expense" {
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
	}

	// Skip processing income rows if type filter is set to "expense"
	if typeFilter == "" || typeFilter == "income" {
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
		var totalExpense, totalIncome float64
		for _, item := range groupsMap[dateStr] {
			if item.Type == "expense" {
				// Expenses are stored as negative amounts, so we use their absolute value for the total
				totalExpense += -item.Amount
			} else if item.Type == "income" {
				totalIncome += item.Amount
			}
		}

		finalGroups = append(finalGroups, response.TransactionGroupResponse{
			GroupLabel:   getGroupLabel(dateStr),
			GroupDate:    dateStr,
			TotalExpense: totalExpense,
			TotalIncome:  totalIncome,
			Items:        groupsMap[dateStr],
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
	// Excel stores dates in M/D/YYYY H:MM:SS (American format, month first)
	// Prioritize M/D formats before D/M to avoid misinterpreting dates like 2/4/2026
	formats := []string{
		"1/2/2006 15:04:05",   // M/D/YYYY H:MM:SS (non-padded, e.g. 2/27/2026 1:02:19)
		"01/02/2006 15:04:05", // MM/DD/YYYY HH:MM:SS (padded, e.g. 02/27/2026 01:02:19)
		"1/2/2006 15:04",      // M/D/YYYY H:MM no secs
		"01/02/2006 15:04",    // MM/DD/YYYY HH:MM no secs
		"2006-01-02 15:04:05", // YYYY-MM-DD HH:MM:SS
		"2006-01-02 15:04",    // YYYY-MM-DD HH:MM no secs
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
		req.Description,   // Column I
		req.Category,      // Column J
		req.Amount,        // Column K
		notes,             // Column L
		req.TransactionAt, // Column M
		createdBy,         // Column N
	}

	if err := u.sheetRepo.AppendRow(spreadsheetID, sheetName+"!I:N", values); err != nil {
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
		req.Priority,      // Column C (Priority)
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

// UpdateTransaction updates an existing transaction (income or expense) based on ID and type
func (u *TransactionUsecase) UpdateTransaction(spreadsheetID string, sheetName string, req request.UpdateTransactionRequest, updatedBy string) error {
	// Parse ID to extract row number
	// ID format: txn_exp_1, txn_inc_2, etc.
	var rowNumber int
	var idPrefix string

	if strings.HasPrefix(req.ID, "txn_exp_") {
		idPrefix = "txn_exp_"
	} else if strings.HasPrefix(req.ID, "txn_inc_") {
		idPrefix = "txn_inc_"
	} else {
		return fmt.Errorf("invalid transaction ID format: %s", req.ID)
	}

	idNumStr := strings.TrimPrefix(req.ID, idPrefix)
	idNum, err := strconv.Atoi(idNumStr)
	if err != nil {
		return fmt.Errorf("invalid transaction ID number: %s", req.ID)
	}

	// Calculate actual row number (ID starts from 1, row 2 is the first data row)
	rowNumber = idNum + 1

	notes := ""
	if req.Notes != nil {
		notes = *req.Notes
	}

	// Update based on transaction type
	if req.Type == "expense" {
		// Expense columns: A-G (Description, Category, Priority, Amount, Notes, TransactionAt, CreatedBy)
		values := []interface{}{
			req.Description,
			req.Category,
			req.Priority,
			req.Amount,
			notes,
			req.TransactionAt,
			updatedBy,
		}

		rangeStr := fmt.Sprintf("%s!A%d:G%d", sheetName, rowNumber, rowNumber)
		if err := u.sheetRepo.UpdateRange(spreadsheetID, rangeStr, [][]interface{}{values}); err != nil {
			return fmt.Errorf("failed to update expense transaction: %w", err)
		}
	} else if req.Type == "income" {
		// Income columns: I-N (Description, Category, Amount, Notes, TransactionAt, CreatedBy)
		values := []interface{}{
			req.Description,
			req.Category,
			req.Amount,
			notes,
			req.TransactionAt,
			updatedBy,
		}

		rangeStr := fmt.Sprintf("%s!I%d:N%d", sheetName, rowNumber, rowNumber)
		if err := u.sheetRepo.UpdateRange(spreadsheetID, rangeStr, [][]interface{}{values}); err != nil {
			return fmt.Errorf("failed to update income transaction: %w", err)
		}
	} else {
		return fmt.Errorf("invalid transaction type: %s (must be 'income' or 'expense')", req.Type)
	}

	return nil
}

// getIndonesianMonthName returns Indonesian month name for a given month number (1-12)
func getIndonesianMonthName(month int) string {
	months := []string{
		"Januari", "Februari", "Maret", "April", "Mei", "Juni",
		"Juli", "Agustus", "September", "Oktober", "November", "Desember",
	}
	if month < 1 || month > 12 {
		return ""
	}
	return months[month-1]
}

// getSheetNamesForPeriod returns a list of sheet names based on the period filter
func getSheetNamesForPeriod(currentSheetName string, period string) []string {
	now := time.Now()

	switch period {
	case "Day":
		// Only return current sheet
		return []string{currentSheetName}

	case "Month":
		// Only return current sheet
		return []string{currentSheetName}

	case "3 Months":
		// Return last 3 months including current month
		sheets := []string{}
		for i := 2; i >= 0; i-- {
			monthDate := now.AddDate(0, -i, 0)
			sheetName := getIndonesianMonthName(int(monthDate.Month()))
			if sheetName != "" {
				sheets = append(sheets, sheetName)
			}
		}
		return sheets

	case "6 Months":
		// Return last 6 months including current month
		sheets := []string{}
		for i := 5; i >= 0; i-- {
			monthDate := now.AddDate(0, -i, 0)
			sheetName := getIndonesianMonthName(int(monthDate.Month()))
			if sheetName != "" {
				sheets = append(sheets, sheetName)
			}
		}
		return sheets

	case "Year":
		// Return all 12 months of the current year
		sheets := []string{}
		for month := 1; month <= 12; month++ {
			sheetName := getIndonesianMonthName(month)
			if sheetName != "" {
				sheets = append(sheets, sheetName)
			}
		}
		return sheets

	default:
		return []string{currentSheetName}
	}
}

// getPeriodLabel returns a display label for the period
func getPeriodLabel(period string) string {
	switch period {
	case "Day":
		return "Today"
	case "Month":
		return "This Month"
	case "3 Months":
		return "Last 3 Months"
	case "6 Months":
		return "Last 6 Months"
	case "Year":
		return "This Year"
	default:
		return "This Month"
	}
}

// getDaysInPeriod returns the number of days to use for daily average calculation
func getDaysInPeriod(period string) float64 {
	now := time.Now()

	switch period {
	case "Day":
		return 1

	case "Month":
		// Current day of the month
		return float64(now.Day())

	case "3 Months":
		// Days in the last 3 months up to today
		threeMonthsAgo := now.AddDate(0, -3, 0)
		return float64(now.Sub(threeMonthsAgo).Hours() / 24)

	case "6 Months":
		// Days in the last 6 months up to today
		sixMonthsAgo := now.AddDate(0, -6, 0)
		return float64(now.Sub(sixMonthsAgo).Hours() / 24)

	case "Year":
		// Days from January 1st to today
		yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		return float64(now.Sub(yearStart).Hours()/24) + 1

	default:
		return float64(now.Day())
	}
}

// GetAnalysis fetches the financial analysis data
func (u *TransactionUsecase) GetAnalysis(spreadsheetID string, sheetName string, period string) (*response.AnalysisResponse, error) {
	// Get sheet names based on period
	sheetNames := getSheetNamesForPeriod(sheetName, period)

	// For Day and Month periods, or if only one sheet
	if period == "Day" || period == "Month" || len(sheetNames) == 1 {
		ranges := []string{
			sheetName + "!AA2",  // 0: total expense
			sheetName + "!P2:T", // 1: exp categories (Nama Kategori, Sub kategori, Budget, Alokasi, Sisa Budget)
			sheetName + "!A2:G", // 2: exp priorities
			sheetName + "!AD2",  // 3: total income
			sheetName + "!I2:N", // 4: inc categories/transactions
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

		expenseData := u.getExpenseAnalysis(getVal(0), getVal(1), getVal(2), period)
		incomeData := u.getIncomeAnalysis(getVal(3), getVal(4), getVal(5), period)

		resp := &response.AnalysisResponse{
			Status: "success",
			Data: response.AnalysisData{
				Expense: expenseData,
				Income:  incomeData,
			},
		}
		return resp, nil
	}

	// For multi-month periods, aggregate data from multiple sheets
	var allExpenseCategories [][]interface{}
	var allExpensePriorities [][]interface{}
	var allIncomeTransactions [][]interface{}

	for _, sheet := range sheetNames {
		ranges := []string{
			sheet + "!P2:T", // exp categories
			sheet + "!A2:G", // exp priorities
			sheet + "!I2:N", // inc transactions
		}

		valueRanges, err := u.sheetRepo.BatchGetValues(spreadsheetID, ranges)
		if err != nil {
			// Skip sheets that don't exist
			continue
		}

		if len(valueRanges) > 0 && valueRanges[0] != nil && valueRanges[0].Values != nil {
			allExpenseCategories = append(allExpenseCategories, valueRanges[0].Values...)
		}
		if len(valueRanges) > 1 && valueRanges[1] != nil && valueRanges[1].Values != nil {
			allExpensePriorities = append(allExpensePriorities, valueRanges[1].Values...)
		}
		if len(valueRanges) > 2 && valueRanges[2] != nil && valueRanges[2].Values != nil {
			allIncomeTransactions = append(allIncomeTransactions, valueRanges[2].Values...)
		}
	}

	// Get master income categories
	masterIncRanges := []string{"Master Data!H4:H"}
	masterIncValueRanges, err := u.sheetRepo.BatchGetValues(spreadsheetID, masterIncRanges)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch master income categories: %w", err)
	}

	var masterIncData [][]interface{}
	if len(masterIncValueRanges) > 0 && masterIncValueRanges[0] != nil && masterIncValueRanges[0].Values != nil {
		masterIncData = masterIncValueRanges[0].Values
	}

	// Calculate totals from aggregated data (no AA2/AD2 for multi-month)
	expenseData := u.getExpenseAnalysis(nil, allExpenseCategories, allExpensePriorities, period)
	incomeData := u.getIncomeAnalysis(nil, allIncomeTransactions, masterIncData, period)

	resp := &response.AnalysisResponse{
		Status: "success",
		Data: response.AnalysisData{
			Expense: expenseData,
			Income:  incomeData,
		},
	}
	return resp, nil
}

func (u *TransactionUsecase) getExpenseAnalysis(aa2, p2t, a2g [][]interface{}, period string) response.AnalysisExpenseData {
	var totalSpent float64
	if len(aa2) > 0 && len(aa2[0]) > 0 {
		totalSpent = parseAmount(aa2[0][0])
	}

	// List each subcategory row with category_name and sub_category_name from P2:T
	// P=Nama Kategori(0), Q=Sub kategori(1), R=Budget(2), S=Alokasi(3), T=Sisa Budget(4)
	// Use map to consolidate duplicate categories (for multi-month aggregation)
	catMap := make(map[string]*response.AnalysisCategory)
	var expCatTotal float64

	for _, row := range p2t {
		if len(row) < 4 {
			continue
		}
		catName := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
		subCatName := strings.TrimSpace(fmt.Sprintf("%v", row[1]))
		if catName == "" || strings.EqualFold(catName, "Nama Kategori") || strings.EqualFold(catName, "Category") {
			continue
		}
		alokasi := parseAmount(row[3]) // Alokasi column

		// Create unique key for category + subcategory
		key := catName + "|" + subCatName

		if existing, exists := catMap[key]; exists {
			// Aggregate amounts for the same category
			existing.Amount += alokasi
		} else {
			catMap[key] = &response.AnalysisCategory{
				CategoryName:    catName,
				SubCategoryName: subCatName,
				Amount:          alokasi,
			}
		}
		expCatTotal += alokasi
	}

	// Convert map to slice
	expCats := []response.AnalysisCategory{}
	for _, cat := range catMap {
		expCats = append(expCats, *cat)
	}

	// Sort by category name, then subcategory name for consistent output
	sort.SliceStable(expCats, func(i, j int) bool {
		if expCats[i].CategoryName == expCats[j].CategoryName {
			return expCats[i].SubCategoryName < expCats[j].SubCategoryName
		}
		return expCats[i].CategoryName < expCats[j].CategoryName
	})

	// If totalSpent is 0 (multi-month aggregation), use the sum of categories
	if totalSpent == 0 && expCatTotal > 0 {
		totalSpent = expCatTotal
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
				Name:         c.SubCategoryName,
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

	daysDivider := getDaysInPeriod(period)
	if daysDivider == 0 {
		daysDivider = 1
	}
	dailyAvgExp := totalSpent / daysDivider

	return response.AnalysisExpenseData{
		Period:      strings.ToLower(strings.ReplaceAll(period, " ", "_")),
		PeriodLabel: getPeriodLabel(period),
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

func (u *TransactionUsecase) getIncomeAnalysis(ad2, incCatsData, masterIncData [][]interface{}, period string) response.AnalysisIncomeData {
	var totalIncome float64
	if len(ad2) > 0 && len(ad2[0]) > 0 {
		totalIncome = parseAmount(ad2[0][0])
	}

	incRowMap := make(map[string]float64)
	for i, row := range incCatsData {
		if i == 0 {
			headerVal := strings.TrimSpace(fmt.Sprintf("%v", row[0]))
			if strings.EqualFold(headerVal, "category") || strings.EqualFold(headerVal, "nama pemasukan") {
				continue
			}
		}
		if len(row) < 3 {
			continue
		}
		cat := strings.TrimSpace(fmt.Sprintf("%v", row[1])) // J is 1 (Jenis Pemasukan)
		amtF := parseAmount(row[2])                         // K is 2 (Jumlah)
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

	// Sort income categories by name for consistent output
	sort.SliceStable(incCats, func(i, j int) bool {
		return incCats[i].Name < incCats[j].Name
	})

	var incCatTotal float64
	for _, c := range incCats {
		incCatTotal += c.Amount
	}

	// Fallback: if AD2 returned 0, use calculated total from income data
	if totalIncome == 0 && incCatTotal > 0 {
		totalIncome = incCatTotal
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

	daysDivider := getDaysInPeriod(period)
	if daysDivider == 0 {
		daysDivider = 1
	}
	dailyAvgInc := totalIncome / daysDivider

	return response.AnalysisIncomeData{
		Period:      strings.ToLower(strings.ReplaceAll(period, " ", "_")),
		PeriodLabel: getPeriodLabel(period),
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
