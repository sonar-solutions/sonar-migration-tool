// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package types

// Measure represents a single component metric measurement returned by
// /api/measures/search.
type Measure struct {
	Metric    string `json:"metric"`
	Value     string `json:"value"`
	Component string `json:"component"`
	BestValue bool   `json:"bestValue"`
}

// MeasuresSearchResponse is the response envelope for /api/measures/search.
type MeasuresSearchResponse struct {
	Measures []Measure `json:"measures"`
}
