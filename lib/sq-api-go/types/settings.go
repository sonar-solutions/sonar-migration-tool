package types

// Setting represents a single configuration setting returned by
// /api/settings/values.
type Setting struct {
	Key         string                   `json:"key"`
	Value       string                   `json:"value"`
	Values      []string                 `json:"values"`
	FieldValues []map[string]interface{} `json:"fieldValues"`
	Inherited   bool                     `json:"inherited"`
}

// SettingsValuesResponse is the response envelope for /api/settings/values.
type SettingsValuesResponse struct {
	Settings []Setting `json:"settings"`
}
