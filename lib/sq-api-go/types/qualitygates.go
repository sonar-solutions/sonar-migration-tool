// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package types

import "encoding/json"

// QualityGateCondition is a single condition on a quality gate.
type QualityGateCondition struct {
	ID     int    `json:"id"`
	Metric string `json:"metric"`
	Op     string `json:"op"`
	Error  string `json:"error"`
}

// QualityGate represents a single quality gate returned by /api/qualitygates/list
// or /api/qualitygates/show.
type QualityGate struct {
	ID         int                    `json:"id"`
	Name       string                 `json:"name"`
	IsDefault  bool                   `json:"isDefault"`
	IsBuiltIn  bool                   `json:"isBuiltIn"`
	Conditions []QualityGateCondition `json:"conditions"`
}

// QualityGatesListResponse is the response envelope for /api/qualitygates/list.
//
// DefaultGate is intentionally json.RawMessage rather than a typed
// value: SonarQube Server returns the default gate as a NAME string,
// while SonarQube Cloud returns it as a numeric ID. Any concrete type
// would fail to unmarshal on one of the two backends — keeping the
// raw bytes lets callers decode either shape on demand (or ignore
// the field, which is what the migrate tool does — IsDefault on each
// QualityGate carries the same signal without ambiguity).
type QualityGatesListResponse struct {
	QualityGates []QualityGate   `json:"qualitygates"`
	DefaultGate  json.RawMessage `json:"default,omitempty"`
}

// QualityGateGroup is returned by /api/qualitygates/search_groups.
type QualityGateGroup struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Selected    bool   `json:"selected"`
}

// QualityGateGroupsResponse is the response envelope for /api/qualitygates/search_groups.
type QualityGateGroupsResponse struct {
	PagedResponse
	Groups []QualityGateGroup `json:"groups"`
}

// QualityGateUser is returned by /api/qualitygates/search_users.
type QualityGateUser struct {
	Login    string `json:"login"`
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

// QualityGateUsersResponse is the response envelope for /api/qualitygates/search_users.
type QualityGateUsersResponse struct {
	PagedResponse
	Users []QualityGateUser `json:"users"`
}
