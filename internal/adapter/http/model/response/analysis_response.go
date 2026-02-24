package response

type AnalysisResponse struct {
	Status string       `json:"status"`
	Data   AnalysisData `json:"data"`
}

type AnalysisData struct {
	Expense AnalysisExpenseData `json:"expense"`
	Income  AnalysisIncomeData  `json:"income"`
}

type AnalysisExpenseData struct {
	Period               string                         `json:"period"`
	PeriodLabel          string                         `json:"period_label"`
	Summary              AnalysisExpenseSummary         `json:"summary"`
	Chart                AnalysisChart                  `json:"chart"`
	TopCategory          AnalysisTopCategory            `json:"top_category"`
	DailyAverage         AnalysisDailyAverage           `json:"daily_average"`
	PriorityDistribution []AnalysisPriorityDistribution `json:"priority_distribution"`
}

type AnalysisIncomeData struct {
	Period       string                `json:"period"`
	PeriodLabel  string                `json:"period_label"`
	Summary      AnalysisIncomeSummary `json:"summary"`
	Chart        AnalysisChart         `json:"chart"`
	TopCategory  AnalysisTopCategory   `json:"top_category"`
	DailyAverage AnalysisDailyAverage  `json:"daily_average"`
}

type AnalysisExpenseSummary struct {
	TotalSpent        float64 `json:"total_spent"`
	TotalSpentDisplay string  `json:"total_spent_display"`
}

type AnalysisIncomeSummary struct {
	TotalIncome        float64 `json:"total_income"`
	TotalIncomeDisplay string  `json:"total_income_display"`
}

type AnalysisChart struct {
	Categories []AnalysisCategory `json:"categories"`
}

type AnalysisCategory struct {
	Name    string  `json:"name"`
	Amount  float64 `json:"amount"`
	Percent int     `json:"percent"`
}

type AnalysisTopCategory struct {
	Name         string  `json:"name"`
	Total        float64 `json:"total"`
	TotalDisplay string  `json:"total_display"`
}

type AnalysisDailyAverage struct {
	Label         string  `json:"label"`
	Amount        float64 `json:"amount"`
	AmountDisplay string  `json:"amount_display"`
}

type AnalysisPriorityDistribution struct {
	Level         string  `json:"level"`
	Label         string  `json:"label"`
	Amount        float64 `json:"amount"`
	AmountDisplay string  `json:"amount_display"`
}
