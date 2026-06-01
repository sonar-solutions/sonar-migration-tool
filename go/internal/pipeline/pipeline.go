// Package pipeline provides the version-specific extraction interface and four
// concrete implementations covering SQ 9.9 LTS, SQ 10.0-10.3, SQ 10.4-10.8,
// and SQ 2025.1+. A version router selects the correct pipeline at startup.
package pipeline

import (
	"context"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// Issue is a normalized representation of a SonarQube issue across all
// supported server versions.
type Issue struct {
	Key          string   `json:"key"`
	Rule         string   `json:"rule"`
	Severity     string   `json:"severity"`
	Component    string   `json:"component"`
	Project      string   `json:"project"`
	Status       string   `json:"status"`
	Resolution   string   `json:"resolution,omitempty"`
	Type         string   `json:"type"`
	Line         int32    `json:"line,omitempty"`
	Message      string   `json:"message"`
	CreationDate string   `json:"creationDate"`
	UpdateDate   string   `json:"updateDate"`
	Tags         []string `json:"tags"`
	Assignee     string   `json:"assignee,omitempty"`
}

// Hotspot is a normalized security hotspot.
type Hotspot struct {
	Key                      string `json:"key"`
	Component                string `json:"component"`
	Project                  string `json:"project"`
	SecurityCategory         string `json:"securityCategory"`
	VulnerabilityProbability string `json:"vulnerabilityProbability"`
	Status                   string `json:"status"`
	Resolution               string `json:"resolution,omitempty"`
	Line                     int32  `json:"line,omitempty"`
	Message                  string `json:"message"`
	Author                   string `json:"author,omitempty"`
	CreationDate             string `json:"creationDate"`
	UpdateDate               string `json:"updateDate"`
	RuleKey                  string `json:"ruleKey"`
}

// Measure is a single metric measurement for a component.
type Measure struct {
	Metric    string `json:"metric"`
	Value     string `json:"value"`
	BestValue bool   `json:"bestValue,omitempty"`
}

// ComponentMetrics groups all measurements for a single component key.
type ComponentMetrics struct {
	Component string
	Measures  []Measure
}

// Group is a normalized SonarQube user group.
type Group struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	MembersCount int    `json:"membersCount"`
	Default      bool   `json:"default"`
}

// Pipeline defines the version-specific extraction and transformation
// operations. Each SQ version range has exactly one concrete implementation.
// The router selects the correct implementation once at startup; no runtime
// version branching occurs inside the extraction or build phases.
type Pipeline interface {
	// Version returns the human-readable pipeline identifier (e.g., "sq-9.9").
	Version() string

	// ExtractIssues extracts all issues for a project using version-appropriate
	// API parameters and status values.
	ExtractIssues(ctx context.Context, projectKey string) ([]Issue, error)

	// ExtractHotspots extracts all security hotspots for a project.
	ExtractHotspots(ctx context.Context, projectKey string) ([]Hotspot, error)

	// ExtractMetrics extracts all leaf-component metrics, respecting
	// version-specific batching requirements for metricKeys.
	ExtractMetrics(ctx context.Context, projectKey string, metricKeys []string) ([]ComponentMetrics, error)

	// ExtractGroups extracts user groups using the appropriate API endpoint
	// for the server version.
	ExtractGroups(ctx context.Context) ([]Group, error)

	// EnrichCleanCode applies Clean Code attributes to issues. For SQ 9.9 this
	// enriches from SonarQube Cloud (requires SPEC-012); for 10.0+ it is a no-op
	// because Clean Code attributes are natively present in the API response.
	EnrichCleanCode(ctx context.Context, issues []Issue, cloudClient *sqapi.Client) ([]Issue, error)

	// IssueSearchParam returns the query-parameter name used for issue status
	// filtering: "statuses" (SQ 9.9 and 10.0-10.3) or "issueStatuses" (SQ 10.4+).
	IssueSearchParam() string

	// IssueStatusValues returns the valid status values for the issue search.
	IssueStatusValues() []string

	// SupportsMetricBatching reports whether metricKeys must be split into
	// batches and the batch size to use. SQ 2025.1+ does not require batching
	// (returns false, 0); earlier versions use 15 keys per request.
	SupportsMetricBatching() (bool, int)
}
