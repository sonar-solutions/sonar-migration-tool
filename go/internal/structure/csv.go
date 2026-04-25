package structure

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

// ExportCSV writes a slice of structs to a CSV file.
// Field names come from the `csv` struct tag. Complex values (slices, maps, bools)
// are JSON-serialized.
func ExportCSV(directory, name string, data any) error {
	rv := reflect.ValueOf(data)
	if rv.Kind() != reflect.Slice || rv.Len() == 0 {
		// Write empty file (header only not possible without data).
		return os.WriteFile(filepath.Join(directory, name+".csv"), nil, 0o644)
	}

	// Extract headers from struct tags.
	elemType := rv.Index(0).Type()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	headers := structCSVHeaders(elemType)

	path := filepath.Join(directory, name+".csv")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating CSV %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if err := w.Write(headers); err != nil {
		return err
	}

	for i := range rv.Len() {
		elem := rv.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		row := make([]string, len(headers))
		for j := range headers {
			row[j] = serializeCSVValue(elem.Field(j))
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// LoadCSV reads a CSV file into a slice of maps.
// Values that look like JSON (arrays, objects, booleans, numbers) are coerced.
func LoadCSV(directory, filename string) ([]map[string]any, error) {
	path := filepath.Join(directory, filename)
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 2 {
		return nil, nil
	}

	headers := records[0]
	var result []map[string]any
	for _, row := range records[1:] {
		m := make(map[string]any, len(headers))
		for i, header := range headers {
			if i < len(row) {
				m[header] = coerceCSVValue(row[i])
			}
		}
		result = append(result, m)
	}
	return result, nil
}

// structCSVHeaders extracts CSV column names from struct tags.
func structCSVHeaders(t reflect.Type) []string {
	headers := make([]string, t.NumField())
	for i := range t.NumField() {
		tag := t.Field(i).Tag.Get("csv")
		if tag == "" {
			tag = t.Field(i).Name
		}
		headers[i] = tag
	}
	return headers
}

// serializeCSVValue converts a reflect.Value to a CSV string.
// Serializes complex values (dicts, lists, bools) as JSON strings.
func serializeCSVValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	iface := v.Interface()

	switch v.Kind() {
	case reflect.Bool:
		b, _ := json.Marshal(iface)
		return string(b)
	case reflect.Slice, reflect.Map:
		if v.IsNil() {
			return ""
		}
		b, _ := json.Marshal(iface)
		return string(b)
	case reflect.Interface:
		if v.IsNil() {
			return ""
		}
		return serializeAny(iface)
	case reflect.Int, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", iface)
	}
}

// serializeAny handles the `any` typed fields.
func serializeAny(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		b, _ := json.Marshal(val)
		return string(b)
	case int:
		return strconv.Itoa(val)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	default:
		b, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(b)
	}
}

// coerceCSVValue attempts to parse a CSV string value back into a typed value,
// JSON arrays/objects/booleans/numbers are parsed back into typed values.
func coerceCSVValue(s string) any {
	if s == "" {
		return s
	}

	// Try JSON parse for complex types.
	s = strings.TrimSpace(s)
	if len(s) > 0 && (s[0] == '{' || s[0] == '[' || s == "true" || s == "false" || s == "null") {
		var v any
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v
		}
	}

	// Try numeric.
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		// Only coerce if it looks like a number (not a string that starts with digits).
		if strconv.FormatFloat(n, 'f', -1, 64) == s || strconv.FormatInt(int64(n), 10) == s {
			return n
		}
	}

	return s
}
