package report

import (
	"fmt"
	"strconv"
	"strings"
)

// ExtractPathValue navigates a parsed JSON structure using dot-notation paths.
// Supports: "$.key", "nested.key", "array.0.field", and list iteration.
func ExtractPathValue(obj any, path string, defaultVal any) any {
	if obj == nil {
		return defaultVal
	}
	segments := splitPath(obj, path)
	return extractSegments(obj, segments, defaultVal)
}

// splitPath converts a path string into segments, checking for direct key match first.
func splitPath(obj any, path string) []string {
	if m, ok := obj.(map[string]any); ok {
		if _, exists := m[path]; exists {
			return []string{path}
		}
	}
	return strings.Split(path, ".")
}

// extractSegments recursively traverses the object using path segments.
func extractSegments(obj any, segments []string, defaultVal any) any {
	if len(segments) == 0 || obj == nil {
		return coalesce(obj, defaultVal)
	}

	seg := segments[0]
	rest := segments[1:]
	val := resolveSegment(obj, seg, defaultVal)

	if val == nil {
		return defaultVal
	}
	if len(rest) > 0 {
		return extractSegments(val, rest, defaultVal)
	}
	return val
}

// resolveSegment handles a single path segment against the current object.
func resolveSegment(obj any, seg string, defaultVal any) any {
	if seg == "$" {
		return obj
	}
	if m, ok := obj.(map[string]any); ok {
		if v, exists := m[seg]; exists {
			return v
		}
		return defaultVal
	}
	if arr, ok := toSlice(obj); ok {
		return resolveArraySegment(arr, seg, defaultVal)
	}
	return defaultVal
}

// resolveArraySegment handles array indexing or iteration.
func resolveArraySegment(arr []any, seg string, defaultVal any) any {
	if idx, err := strconv.Atoi(seg); err == nil {
		if idx >= 0 && idx < len(arr) {
			return arr[idx]
		}
		return defaultVal
	}
	// Non-numeric segment on array: iterate and extract from each element.
	result := make([]any, 0, len(arr))
	for _, item := range arr {
		v := ExtractPathValue(item, seg, nil)
		if v != nil {
			result = append(result, v)
		}
	}
	if len(result) == 0 {
		return defaultVal
	}
	return result
}

func toSlice(v any) ([]any, bool) {
	if arr, ok := v.([]any); ok {
		return arr, true
	}
	return nil, false
}

func coalesce(v, defaultVal any) any {
	if v == nil {
		return defaultVal
	}
	return v
}

// --- Convenience wrappers ---

// ExtractString extracts a string value at the given path.
func ExtractString(obj any, path string) string {
	v := ExtractPathValue(obj, path, nil)
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// ExtractBool extracts a boolean value at the given path.
func ExtractBool(obj any, path string) bool {
	v := ExtractPathValue(obj, path, false)
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// ExtractFloat extracts a float64 value at the given path.
func ExtractFloat(obj any, path string, defaultVal float64) float64 {
	v := ExtractPathValue(obj, path, defaultVal)
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case string:
		if f, err := strconv.ParseFloat(n, 64); err == nil {
			return f
		}
	}
	return defaultVal
}

// ExtractInt extracts an int value at the given path.
func ExtractInt(obj any, path string, defaultVal int) int {
	v := ExtractPathValue(obj, path, defaultVal)
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case string:
		if i, err := strconv.Atoi(n); err == nil {
			return i
		}
	}
	return defaultVal
}

// SetPathValue sets a value at the given dot-notation path in a parsed JSON structure.
func SetPathValue(obj any, path string, val any) any {
	segments := strings.Split(path, ".")
	if len(segments) > 0 && segments[0] == "$" {
		segments = segments[1:]
	}
	return setSegments(obj, segments, val)
}

func setSegments(obj any, segments []string, val any) any {
	if len(segments) == 0 {
		return val
	}
	seg := segments[0]
	rest := segments[1:]

	if m, ok := obj.(map[string]any); ok {
		return setMapSegment(m, seg, rest, val)
	}
	if arr, ok := obj.([]any); ok {
		return setArraySegment(arr, seg, rest, val)
	}
	return obj
}

func setMapSegment(m map[string]any, seg string, rest []string, val any) map[string]any {
	if len(rest) == 0 {
		m[seg] = val
	} else {
		if m[seg] == nil {
			m[seg] = make(map[string]any)
		}
		m[seg] = setSegments(m[seg], rest, val)
	}
	return m
}

func setArraySegment(arr []any, seg string, rest []string, val any) []any {
	idx, err := strconv.Atoi(seg)
	if err != nil || idx < 0 || idx >= len(arr) {
		return arr
	}
	if len(rest) == 0 {
		arr[idx] = val
	} else {
		arr[idx] = setSegments(arr[idx], rest, val)
	}
	return arr
}
