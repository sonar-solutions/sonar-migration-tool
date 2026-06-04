// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package types

// BranchStatus holds the quality gate result for a branch.
type BranchStatus struct {
	QualityGateStatus string `json:"qualityGateStatus"`
	Bugs              int    `json:"bugs"`
	Vulnerabilities   int    `json:"vulnerabilities"`
	CodeSmells        int    `json:"codeSmells"`
}

// Branch represents a single project branch returned by
// /api/project_branches/list. BranchID is the UUID consumed by other
// SonarQube Cloud endpoints (for example as projects[].branchId in the
// enterprise portfolios PATCH body).
type Branch struct {
	Name              string       `json:"name"`
	IsMain            bool         `json:"isMain"`
	Type              string       `json:"type"`
	Status            BranchStatus `json:"status"`
	AnalysisDate      string       `json:"analysisDate"`
	ExcludedFromPurge bool         `json:"excludedFromPurge"`
	BranchID          string       `json:"branchId"`
	BranchUUIDV1      string       `json:"branchUuidV1,omitempty"`
}

// BranchesResponse is the response envelope for /api/project_branches/list.
type BranchesResponse struct {
	Branches []Branch `json:"branches"`
}
