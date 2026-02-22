package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// SpreadsheetIDMiddleware reads X-Spreadsheet-ID from the request header
// and stores it in the Echo context. Returns 400 if the header is missing.
func SpreadsheetIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			spreadsheetID := c.Request().Header.Get("X-Spreadsheet-ID")
			if spreadsheetID == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Missing X-Spreadsheet-ID header",
				})
			}

			sheetName := c.Request().Header.Get("X-Sheet-Name")
			if sheetName == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Missing X-Sheet-Name header",
				})
			}

			c.Set("spreadsheet_id", spreadsheetID)
			c.Set("sheet_name", sheetName)
			return next(c)
		}
	}
}
