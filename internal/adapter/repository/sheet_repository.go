package repository

import (
	"fmt"
	"strconv"

	"byeboros-backend/internal/infrastructure/gsheet"

	"google.golang.org/api/sheets/v4"
)

// SheetRepository provides CRUD operations for Google Sheets
type SheetRepository struct {
	client *gsheet.Client
}

// NewSheetRepository creates a new SheetRepository
func NewSheetRepository(client *gsheet.Client) *SheetRepository {
	return &SheetRepository{client: client}
}

// GetAllRows returns all rows from a sheet (including header)
func (r *SheetRepository) GetAllRows(spreadsheetID, sheetName string) ([][]interface{}, error) {
	resp, err := r.client.Service.Spreadsheets.Values.Get(
		spreadsheetID, sheetName,
	).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get all rows: %w", err)
	}
	return resp.Values, nil
}

// GetRangeValues returns values from a specific range string (e.g. "Januari!A2:E")
func (r *SheetRepository) GetRangeValues(spreadsheetID, rangeStr string) ([][]interface{}, error) {
	resp, err := r.client.Service.Spreadsheets.Values.Get(
		spreadsheetID, rangeStr,
	).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get range values: %w", err)
	}
	return resp.Values, nil
}

// GetDataRows returns all rows excluding the header row
func (r *SheetRepository) GetDataRows(spreadsheetID, sheetName string) ([][]interface{}, error) {
	rows, err := r.GetAllRows(spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}
	if len(rows) <= 1 {
		return [][]interface{}{}, nil
	}
	return rows[1:], nil
}

// GetHeaders returns the first row (header) of a sheet
func (r *SheetRepository) GetHeaders(spreadsheetID, sheetName string) ([]interface{}, error) {
	resp, err := r.client.Service.Spreadsheets.Values.Get(
		spreadsheetID, sheetName+"!1:1",
	).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get headers: %w", err)
	}
	if len(resp.Values) == 0 {
		return []interface{}{}, nil
	}
	return resp.Values[0], nil
}

// GetRowByIndex returns a single row by its 1-based index
func (r *SheetRepository) GetRowByIndex(spreadsheetID, sheetName string, rowIndex int) ([]interface{}, error) {
	rangeStr := fmt.Sprintf("%s!%d:%d", sheetName, rowIndex, rowIndex)
	resp, err := r.client.Service.Spreadsheets.Values.Get(
		spreadsheetID, rangeStr,
	).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get row %d: %w", rowIndex, err)
	}
	if len(resp.Values) == 0 {
		return nil, fmt.Errorf("row %d not found", rowIndex)
	}
	return resp.Values[0], nil
}

// FindRows searches for rows where the specified column matches the given value
// colIndex is 0-based
func (r *SheetRepository) FindRows(spreadsheetID, sheetName string, colIndex int, value string) ([][]interface{}, error) {
	rows, err := r.GetAllRows(spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}

	var results [][]interface{}
	for _, row := range rows {
		if colIndex < len(row) {
			cellValue := fmt.Sprintf("%v", row[colIndex])
			if cellValue == value {
				results = append(results, row)
			}
		}
	}
	return results, nil
}

// FindRowIndex searches for the first row where the specified column matches the value
// Returns the 1-based row index, or -1 if not found
func (r *SheetRepository) FindRowIndex(spreadsheetID, sheetName string, colIndex int, value string) (int, error) {
	rows, err := r.GetAllRows(spreadsheetID, sheetName)
	if err != nil {
		return -1, err
	}

	for i, row := range rows {
		if colIndex < len(row) {
			cellValue := fmt.Sprintf("%v", row[colIndex])
			if cellValue == value {
				return i + 1, nil // 1-based index
			}
		}
	}
	return -1, nil
}

// AppendRow appends a new row at the end of the sheet
func (r *SheetRepository) AppendRow(spreadsheetID, sheetName string, values []interface{}) error {
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{values},
	}

	_, err := r.client.Service.Spreadsheets.Values.Append(
		spreadsheetID, sheetName, valueRange,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("failed to append row: %w", err)
	}
	return nil
}

// UpdateRow updates a specific row by its 1-based index
func (r *SheetRepository) UpdateRow(spreadsheetID, sheetName string, rowIndex int, values []interface{}) error {
	lastCol := columnIndexToLetter(len(values) - 1)
	rangeStr := fmt.Sprintf("%s!A%d:%s%d", sheetName, rowIndex, lastCol, rowIndex)

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{values},
	}

	_, err := r.client.Service.Spreadsheets.Values.Update(
		spreadsheetID, rangeStr, valueRange,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("failed to update row %d: %w", rowIndex, err)
	}
	return nil
}

// DeleteRow deletes a row by its 1-based index
func (r *SheetRepository) DeleteRow(spreadsheetID, sheetName string, sheetID int64, rowIndex int) error {
	req := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: []*sheets.Request{
			{
				DeleteDimension: &sheets.DeleteDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    sheetID,
						Dimension:  "ROWS",
						StartIndex: int64(rowIndex - 1), // 0-based
						EndIndex:   int64(rowIndex),
					},
				},
			},
		},
	}

	_, err := r.client.Service.Spreadsheets.BatchUpdate(spreadsheetID, req).Do()
	if err != nil {
		return fmt.Errorf("failed to delete row %d: %w", rowIndex, err)
	}
	return nil
}

// UpdateCell updates a specific cell
// row is 1-based, col is 0-based
func (r *SheetRepository) UpdateCell(spreadsheetID, sheetName string, row int, col int, value interface{}) error {
	colLetter := columnIndexToLetter(col)
	rangeStr := fmt.Sprintf("%s!%s%d", sheetName, colLetter, row)

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{value}},
	}

	_, err := r.client.Service.Spreadsheets.Values.Update(
		spreadsheetID, rangeStr, valueRange,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("failed to update cell: %w", err)
	}
	return nil
}

// GetColumnValues returns all values in a specific column (0-based index)
func (r *SheetRepository) GetColumnValues(spreadsheetID, sheetName string, colIndex int) ([]interface{}, error) {
	colLetter := columnIndexToLetter(colIndex)
	rangeStr := fmt.Sprintf("%s!%s:%s", sheetName, colLetter, colLetter)

	resp, err := r.client.Service.Spreadsheets.Values.Get(
		spreadsheetID, rangeStr,
	).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get column values: %w", err)
	}

	var values []interface{}
	for _, row := range resp.Values {
		if len(row) > 0 {
			values = append(values, row[0])
		}
	}
	return values, nil
}

// GetRowsAsMap returns all data rows as a slice of maps,
// where the keys are the header column names
func (r *SheetRepository) GetRowsAsMap(spreadsheetID, sheetName string) ([]map[string]interface{}, error) {
	rows, err := r.GetAllRows(spreadsheetID, sheetName)
	if err != nil {
		return nil, err
	}
	if len(rows) <= 1 {
		return []map[string]interface{}{}, nil
	}

	headers := rows[0]
	var results []map[string]interface{}

	for _, row := range rows[1:] {
		record := make(map[string]interface{})
		for j, header := range headers {
			key := fmt.Sprintf("%v", header)
			if j < len(row) {
				record[key] = row[j]
			} else {
				record[key] = ""
			}
		}
		results = append(results, record)
	}
	return results, nil
}

// GetSheetID returns the sheet ID (gid) for a given sheet name
func (r *SheetRepository) GetSheetID(spreadsheetID, sheetName string) (int64, error) {
	spreadsheet, err := r.client.Service.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	for _, s := range spreadsheet.Sheets {
		if s.Properties.Title == sheetName {
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet '%s' not found", sheetName)
}

// columnIndexToLetter converts a 0-based column index to a column letter (A, B, ..., Z, AA, AB, ...)
func columnIndexToLetter(index int) string {
	result := ""
	for index >= 0 {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
	}
	return result
}

// CountRows returns the total number of rows (including header) in a sheet
func (r *SheetRepository) CountRows(spreadsheetID, sheetName string) (int, error) {
	rows, err := r.GetAllRows(spreadsheetID, sheetName)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// ClearSheet clears all data in a sheet
func (r *SheetRepository) ClearSheet(spreadsheetID, sheetName string) error {
	_, err := r.client.Service.Spreadsheets.Values.Clear(
		spreadsheetID, sheetName,
		&sheets.ClearValuesRequest{},
	).Do()
	if err != nil {
		return fmt.Errorf("failed to clear sheet: %w", err)
	}
	return nil
}

// BatchAppendRows appends multiple rows at once
func (r *SheetRepository) BatchAppendRows(spreadsheetID, sheetName string, rows [][]interface{}) error {
	valueRange := &sheets.ValueRange{
		Values: rows,
	}

	_, err := r.client.Service.Spreadsheets.Values.Append(
		spreadsheetID, sheetName, valueRange,
	).ValueInputOption("USER_ENTERED").Do()
	if err != nil {
		return fmt.Errorf("failed to batch append rows: %w", err)
	}
	return nil
}

// RowToMap converts a row ([]interface{}) to a map using the provided headers
func RowToMap(headers []interface{}, row []interface{}) map[string]interface{} {
	record := make(map[string]interface{})
	for j, header := range headers {
		key := fmt.Sprintf("%v", header)
		if j < len(row) {
			record[key] = row[j]
		} else {
			record[key] = ""
		}
	}
	return record
}

// ToInt converts an interface{} value to int (useful for sheet cell values)
func ToInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	case int:
		return v, nil
	default:
		return 0, fmt.Errorf("cannot convert %v to int", val)
	}
}
