package request

type CategoryItem struct {
	Name   string  `json:"name" validate:"required"`
	Budget float64 `json:"budget" validate:"required"`
}

type SaveCategoryRequest struct {
	DailyBudget   float64        `json:"daily_budget"`
	MonthlyBudget float64        `json:"monthly_budget"`
	Categories    []CategoryItem `json:"categories"`
}
