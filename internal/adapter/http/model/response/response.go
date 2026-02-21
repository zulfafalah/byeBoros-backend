package response

import "github.com/labstack/echo/v4"

// SuccessResponse is a standard success response
type SuccessResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse is a standard error response
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// Success returns a standard JSON success response
func Success(c echo.Context, code int, message string, data interface{}) error {
	return c.JSON(code, SuccessResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

// Error returns a standard JSON error response
func Error(c echo.Context, code int, message string, err string) error {
	return c.JSON(code, ErrorResponse{
		Status:  "error",
		Message: message,
		Error:   err,
	})
}

// PaginatedResponse is a standard paginated response
type PaginatedResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Meta    Meta        `json:"meta"`
}

// Meta contains pagination metadata
type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Paginated returns a standard JSON paginated response
func Paginated(c echo.Context, code int, message string, data interface{}, meta Meta) error {
	return c.JSON(code, PaginatedResponse{
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}
