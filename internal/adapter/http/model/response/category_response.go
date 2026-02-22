package response

type CategoryItem struct {
	CategoryName    string  `json:"category_name"`
	SubCategoryName string  `json:"sub_category_name"`
	Budget          float64 `json:"budget"`
}

type CategoryResponse struct {
	DailyBudget   float64        `json:"daily_budget"`
	MonthlyBudget float64        `json:"monthly_budget"`
	Categories    []CategoryItem `json:"categories"`
}
