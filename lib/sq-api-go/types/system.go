// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package types

import "encoding/json"

// SystemInfo holds the raw response from /api/system/info.
//
// The system info payload is a deeply nested, version-varying object.
// Values are kept as raw JSON so callers can unmarshal only the fields
// they need without requiring this package to track every possible field.
//
// Example:
//
//	var edition struct{ Name string }
//	if raw, ok := info["System"]; ok {
//	    _ = json.Unmarshal(raw, &edition)
//	}
type SystemInfo map[string]json.RawMessage
