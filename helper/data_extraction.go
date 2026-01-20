package helper

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/xuri/excelize/v2"
)

// GetSortedUniqueKeys extracts all unique keys (field names) from a slice of Records
// and returns them as an alphabetically sorted slice of strings.
func GetSortedUniqueKeys(records []datatype.DataMap) []string {
	keyMap := make(map[string]struct{})
	for _, record := range records {
		for key := range record {
			keyMap[key] = struct{}{}
		}
	}

	var keys []string
	for key := range keyMap {
		keys = append(keys, key)
	}

	// Sort the keys alphabetically for a deterministic column order
	sort.Strings(keys)
	return keys
}

// ConvertJSONArrayToCSV takes a byte slice of JSON data (an array of objects)
// and a list of desired column names. It converts this data into a CSV string.
func ConvertJSONArrayToDataArray(jsonData interface{}, headingColumns []string) ([][]string, error) {
	var records []datatype.DataMap
	jsonInput := []byte(ToJson(jsonData))

	if err := json.Unmarshal(jsonInput, &records); err != nil {
		log.Fatalf("Error unmarshaling JSON data for key extraction: %v", err)

		return nil, fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}

	dataSize := len(records)
	headingColumnsNames := []string{}
	result := make([][]string, 0, dataSize)

	if len(headingColumns) == 0 {
		// Automatically determine the column names and sort them alphabetically
		headingColumns = GetSortedUniqueKeys(records)
	}

	for _, col := range headingColumns {
		headingColumnsNames = append(headingColumnsNames, ToTitle(col))
	}

	result = append(result, headingColumnsNames)

	// --- 4. Write Data Rows ---
	for _, record := range records {
		var csvRow []string
		// Iterate through the specified columns to maintain order
		for _, key := range headingColumns {
			// Get the value from the record map
			value, exists := record[key]

			if !exists {
				// If the key is not present in the record, add an empty string
				csvRow = append(csvRow, "")
				continue
			}

			// Convert the value to a string based on its underlying type
			var valueStr string
			switch v := value.(type) {
			case string:
				valueStr = v
			case float64:
				// JSON numbers are typically unmarshalled as float64
				// Format as a regular string representation
				valueStr = fmt.Sprintf("%v", v)
			case bool:
				valueStr = fmt.Sprintf("%v", v)
			case []interface{}:
				// This handles the JavaScript logic of joining arrays with " / "
				strElements := make([]string, len(v))
				for i, elem := range v {
					// Recursively convert array elements to string
					strElements[i] = fmt.Sprintf("%v", elem)
				}
				valueStr = strings.Join(strElements, " / ")
			default:
				// Catch-all for other types (e.g., nested objects, null)
				if value != nil {
					valueStr = fmt.Sprintf("%v", value)
				} else {
					valueStr = ""
				}
			}

			// Note: The csv.Writer handles complex escaping/quoting (like for strings
			// containing commas or quotes) automatically.
			csvRow = append(csvRow, valueStr)
		}

		result = append(result, csvRow)
	}

	return result, nil
}

// ConvertJSONArrayToCSV takes a byte slice of JSON data (an array of objects)
// and a list of desired column names. It converts this data into a CSV string.
func ConvertJSONArrayToCSV(jsonData interface{}, headingColumns []string, filename string) (string, error) {
	records, _ := ConvertJSONArrayToDataArray(jsonData, headingColumns)

	// 2. Prepare the CSV writer
	// A bytes.Buffer implements io.Writer, which the csv.Writer needs.
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// --- 4. Write Data Rows ---
	for _, csvRow := range records {
		if err := writer.Write(csvRow); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	// Flush the buffer to ensure all data is written
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("failed to flush CSV writer: %w", err)
	}

	if IsNotEmpty(filename) {
		// A bytes.Buffer implements io.Writer, which the csv.Writer needs.
		publicDir, _ := GetPublicPath()
		fileSaved := path.Join(publicDir, filename)

		SaveToFile(buf.String(), fileSaved)
		return fileSaved, nil
	}

	return buf.String(), nil
}

// ConvertJSONArrayToCSV takes a byte slice of JSON data (an array of objects)
// and a list of desired column names. It converts this data into a CSV string.
func ConvertJSONArrayToExcel(jsonData interface{}, headingColumns []string, filename string) (string, error) {
	records, _ := ConvertJSONArrayToDataArray(jsonData, headingColumns)

	if IsEmpty(filename) {
		filename = "tmp/" + GetHexString(24) + ".xlsx"
	}

	// A bytes.Buffer implements io.Writer, which the csv.Writer needs.
	publicDir, _ := GetPublicPath()
	fileSaved := path.Join(publicDir, filename)

	CreateDirectory(filepath.Dir(fileSaved))
	// console.Info("Saving Excel file to:", fileSaved)
	// console.Info("Saving Excel file to:", records)

	f := excelize.NewFile()

	// Create a new sheet
	index, err := f.NewSheet("Report")
	if err != nil {
		panic(err)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#FFFFFF", Family: "Calibri"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#797979ff"}, Pattern: 1},
	})

	if len(records) > 0 {
		columnsCount := uint(len(records[0]))

		// Apply style to header row
		f.SetCellStyle("Report", cellLocation(1, 1), cellLocation(1, columnsCount), style)

		// --- 4. Write Data Rows ---
		for iRow, dataRow := range records {

			for iCol, cellValue := range dataRow {
				// Convert column index to letter (A, B, C, ...)
				cellKey := cellLocation(uint(iRow+1), uint(iCol+1))

				f.SetCellValue("Report", cellKey, cellValue)
			}

		}
	}

	// Set active sheet
	f.SetActiveSheet(index)

	// Save to file
	if err := f.SaveAs(fileSaved); err != nil {
		console.Error("Error saving Excel file:", err)
		return "", err
	}

	return fileSaved, nil
}

func cellLocation(row uint, col uint) string {
	colLetter, _ := excelize.ColumnNumberToName(int(col))
	return fmt.Sprintf("%s%d", colLetter, row)
}
