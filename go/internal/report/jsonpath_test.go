package report

import (
	"encoding/json"
	"testing"
)

const (
	testVersion  = "9.9"
	testPath     = "System.Version"
	errGotFloat  = "got %f"
	errGotInt    = "got %d"
)

func parseJSON(t *testing.T, s string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		t.Fatalf("parse JSON: %v", err)
	}
	return v
}

func TestExtractPathValueNestedDict(t *testing.T) {
	obj := parseJSON(t, `{"System": {"Version": "9.9", "Edition": "enterprise"}}`)
	got := ExtractPathValue(obj, testPath, nil)
	if got != testVersion {
		t.Errorf("got %v, want %s", got, testVersion)
	}
}

func TestExtractPathValueRootRef(t *testing.T) {
	obj := parseJSON(t, `{"key": "val"}`)
	got := ExtractPathValue(obj, "$", nil)
	if got == nil {
		t.Fatal("expected non-nil for $ path")
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", got)
	}
	if m["key"] != "val" {
		t.Errorf("got %v", m["key"])
	}
}

func TestExtractPathValueDirectKey(t *testing.T) {
	obj := parseJSON(t, `{"qualityProfiles": [{"key": "java-profile"}]}`)
	got := ExtractPathValue(obj, "qualityProfiles", nil)
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", got)
	}
	if len(arr) != 1 {
		t.Errorf("expected 1 profile, got %d", len(arr))
	}
}

func TestExtractPathValueArrayIndex(t *testing.T) {
	obj := parseJSON(t, `{"array": [{"name": "first"}, {"name": "second"}]}`)
	got := ExtractPathValue(obj, "array.0.name", nil)
	if got != "first" {
		t.Errorf("got %v, want first", got)
	}
	got2 := ExtractPathValue(obj, "array.1.name", nil)
	if got2 != "second" {
		t.Errorf("got %v, want second", got2)
	}
}

func TestExtractPathValueArrayIteration(t *testing.T) {
	obj := parseJSON(t, `{"array": [{"name": "a"}, {"name": "b"}]}`)
	got := ExtractPathValue(obj, "array.name", nil)
	arr, ok := got.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", got)
	}
	if len(arr) != 2 {
		t.Fatalf("expected 2, got %d", len(arr))
	}
	if arr[0] != "a" || arr[1] != "b" {
		t.Errorf("got %v", arr)
	}
}

func TestExtractPathValueMissingKey(t *testing.T) {
	obj := parseJSON(t, `{"a": 1}`)
	got := ExtractPathValue(obj, "missing", "default")
	if got != "default" {
		t.Errorf("got %v, want default", got)
	}
}

func TestExtractPathValueNilObject(t *testing.T) {
	got := ExtractPathValue(nil, "any.path", "fallback")
	if got != "fallback" {
		t.Errorf("got %v, want fallback", got)
	}
}

func TestExtractPathValueDeepNested(t *testing.T) {
	obj := parseJSON(t, `{"a": {"b": {"c": "deep"}}}`)
	got := ExtractPathValue(obj, "a.b.c", nil)
	if got != "deep" {
		t.Errorf("got %v, want deep", got)
	}
}

func TestExtractPathValueArrayOutOfBounds(t *testing.T) {
	obj := parseJSON(t, `{"arr": [1, 2]}`)
	got := ExtractPathValue(obj, "arr.5", "oob")
	if got != "oob" {
		t.Errorf("got %v, want oob", got)
	}
}

func TestExtractPathValueDollarNested(t *testing.T) {
	obj := parseJSON(t, `{"qualityGate": {"name": "Sonar way"}}`)
	got := ExtractPathValue(obj, "$.qualityGate.name", nil)
	if got != "Sonar way" {
		t.Errorf("got %v, want Sonar way", got)
	}
}

func TestExtractPathValueBoolField(t *testing.T) {
	obj := parseJSON(t, `{"isBuiltIn": true, "isDefault": false}`)
	if ExtractPathValue(obj, "isBuiltIn", false) != true {
		t.Error("expected true")
	}
	if ExtractPathValue(obj, "isDefault", true) != false {
		t.Error("expected false")
	}
}

// --- Convenience wrappers ---

func TestExtractString(t *testing.T) {
	obj := parseJSON(t, `{"System": {"Version": "9.9"}}`)
	if ExtractString(obj, testPath) != testVersion {
		t.Errorf("got %q", ExtractString(obj, testPath))
	}
	if ExtractString(obj, "missing") != "" {
		t.Error("expected empty for missing")
	}
	if ExtractString(nil, "any") != "" {
		t.Error("expected empty for nil")
	}
}

func TestExtractBoolWrapper(t *testing.T) {
	obj := parseJSON(t, `{"enabled": true, "disabled": false}`)
	if !ExtractBool(obj, "enabled") {
		t.Error("expected true")
	}
	if ExtractBool(obj, "disabled") {
		t.Error("expected false")
	}
	if ExtractBool(obj, "missing") {
		t.Error("expected false for missing")
	}
}

func TestExtractFloatWrapper(t *testing.T) {
	obj := parseJSON(t, `{"coverage": 85.5, "count": 42}`)
	if ExtractFloat(obj, "coverage", 0) != 85.5 {
		t.Errorf(errGotFloat, ExtractFloat(obj, "coverage", 0))
	}
	if ExtractFloat(obj, "count", 0) != 42 {
		t.Errorf(errGotFloat, ExtractFloat(obj, "count", 0))
	}
	if ExtractFloat(obj, "missing", -1) != -1 {
		t.Error("expected default")
	}
}

func TestExtractIntWrapper(t *testing.T) {
	obj := parseJSON(t, `{"count": 42, "float_count": 7.0}`)
	if ExtractInt(obj, "count", 0) != 42 {
		t.Errorf(errGotInt, ExtractInt(obj, "count", 0))
	}
	if ExtractInt(obj, "float_count", 0) != 7 {
		t.Errorf(errGotInt, ExtractInt(obj, "float_count", 0))
	}
	if ExtractInt(obj, "missing", -1) != -1 {
		t.Error("expected default")
	}
}

func TestExtractFloatFromString(t *testing.T) {
	obj := parseJSON(t, `{"value": "3.14"}`)
	if ExtractFloat(obj, "value", 0) != 3.14 {
		t.Errorf(errGotFloat, ExtractFloat(obj, "value", 0))
	}
}

func TestExtractIntFromString(t *testing.T) {
	obj := parseJSON(t, `{"value": "42"}`)
	if ExtractInt(obj, "value", 0) != 42 {
		t.Errorf(errGotInt, ExtractInt(obj, "value", 0))
	}
}
