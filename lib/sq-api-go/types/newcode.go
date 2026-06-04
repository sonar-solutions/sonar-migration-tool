// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package types

// NewCodePeriod represents a single new code period definition returned by
// /api/new_code_periods/list.
type NewCodePeriod struct {
	Type           string `json:"type"`
	Value          string `json:"value"`
	Inherited      bool   `json:"inherited"`
	BranchKey      string `json:"branchKey"`
	EffectiveValue string `json:"effectiveValue"`
}

// NewCodePeriodsResponse is the response envelope for /api/new_code_periods/list.
type NewCodePeriodsResponse struct {
	NewCodePeriods []NewCodePeriod `json:"newCodePeriods"`
}
