package request

type CategoryItem struct {
	CategoryName    string  `json:"category_name" validate:"required"`
	SubCategoryName string  `json:"sub_category_name" validate:"required"`
	Budget          float64 `json:"budget" validate:"required"`
}

type SaveCategoryRequest struct {
	DailyBudget   float64        `json:"daily_budget"`
	MonthlyBudget float64        `json:"monthly_budget"`
	Categories    []CategoryItem `json:"categories"`
}
