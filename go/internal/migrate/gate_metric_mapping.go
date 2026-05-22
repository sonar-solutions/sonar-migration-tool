package migrate

// metricMapping maps a SonarQube Server quality-gate metric that does not
// exist on SonarQube Cloud to its closest SQC equivalent(s). The mapping
// follows the table maintained in issue #143.
//
// Semantics:
//
//   - A single-element slice replaces the source metric with that target
//     metric. The source condition's op + threshold are preserved.
//   - A multi-element slice (composite mapping) emits one SQC condition per
//     target metric, each preserving the source op + threshold. This is the
//     best-effort translation of the "metric_a + metric_b" entries in
//     the issue table.
//   - An empty slice means the source metric has no meaningful SQC
//     equivalent — the condition is dropped and the gate is migrated with
//     one fewer condition. The migration logs this at Warn level.
//
// Metrics that already exist verbatim on SQC are intentionally absent from
// this map: lookupMetricReplacement returns ok=false for them and the
// caller forwards the condition unchanged.
var metricMapping = map[string][]string{
	// *_with_aica / *_without_aica variants — drop the AICA suffix.
	"new_maintainability_rating_with_aica":     {"new_maintainability_rating"},
	"new_maintainability_rating_without_aica":  {"new_maintainability_rating"},
	"new_reliability_rating_with_aica":         {"new_reliability_rating"},
	"new_reliability_rating_without_aica":      {"new_reliability_rating"},
	"new_security_rating_with_aica":            {"new_security_rating"},
	"new_security_rating_without_aica":         {"new_security_rating"},
	"new_security_review_rating_with_aica":     {"new_security_review_rating"},
	"new_security_review_rating_without_aica":  {"new_security_review_rating"},
	"reliability_rating_with_aica":             {"reliability_rating"},
	"reliability_rating_without_aica":          {"reliability_rating"},
	"security_rating_with_aica":                {"security_rating"},
	"security_rating_without_aica":             {"security_rating"},
	"security_review_rating_with_aica":         {"security_review_rating"},
	"security_review_rating_without_aica":      {"security_review_rating"},
	"sqale_rating_with_aica":                   {"sqale_rating"},
	"sqale_rating_without_aica":                {"sqale_rating"},

	// software_quality_* (classic / SQS-only) → classic SQC equivalents.
	"software_quality_maintainability_debt_ratio":          {"sqale_debt_ratio"},
	"software_quality_maintainability_rating":              {"sqale_rating"},
	"software_quality_maintainability_rating_with_aica":    {"sqale_rating"},
	"software_quality_maintainability_rating_without_aica": {"sqale_rating"},
	"software_quality_reliability_rating_with_aica":        {"reliability_rating"},
	"software_quality_reliability_rating_without_aica":     {"reliability_rating"},
	"software_quality_security_rating":                     {"security_rating"},
	"software_quality_security_rating_with_aica":           {"security_rating"},
	"software_quality_security_rating_without_aica":        {"security_rating"},

	// new_software_quality_* → new_* equivalents.
	"new_software_quality_maintainability_debt_ratio":          {"new_sqale_debt_ratio"},
	"new_software_quality_maintainability_issues":              {"new_issues"},
	"new_software_quality_maintainability_rating":              {"new_sqale_debt_ratio"},
	"new_software_quality_maintainability_rating_with_aica":    {"new_maintainability_rating"},
	"new_software_quality_maintainability_rating_without_aica": {"new_maintainability_rating"},
	"new_software_quality_reliability_rating":                  {"new_reliability_rating"},
	"new_software_quality_reliability_rating_with_aica":        {"new_reliability_rating"},
	"new_software_quality_reliability_rating_without_aica":     {"new_reliability_rating"},
	"new_software_quality_security_rating":                     {"new_security_rating"},

	// Composite mappings — one source condition becomes multiple target
	// conditions, each preserving the source op and threshold. The set of
	// target metrics matches the right-hand column of issue #143.
	"software_quality_blocker_issues": {"security_review_rating", "reliability_rating"},
	"software_quality_high_issues":    {"security_review_rating", "reliability_rating"},
	"software_quality_medium_issues":  {"security_review_rating", "reliability_rating"},
	"software_quality_low_issues":     {"security_rating", "reliability_rating"},
	"software_quality_info_issues":    {"security_rating", "reliability_rating"},

	"new_software_quality_blocker_issues": {"new_security_review_rating", "new_reliability_rating"},
	"new_software_quality_high_issues":    {"new_security_review_rating", "new_reliability_rating"},
	"new_software_quality_medium_issues":  {"new_security_review_rating", "new_reliability_rating"},
	"new_software_quality_low_issues":     {"new_security_review_rating", "new_reliability_rating"},
	"new_software_quality_info_issues":    {"new_security_review_rating", "new_reliability_rating"},

	"new_software_quality_reliability_issues": {"new_reliability_rating"},
	"new_software_quality_security_issues":    {"new_security_rating"},

	// Source metrics with no meaningful SQC equivalent — drop the condition.
	"contains_ai_code":                                 {},
	"effort_to_reach_software_quality_maintainability_rating_a": {},
	"filename_size":                                    {},
	"filename_size_rating":                             {},
	"ncloc_with_aica":                                  {},
	"ncloc_without_aica":                               {},
	"new_software_quality_maintainability_remediation_effort": {},
	"new_software_quality_reliability_remediation_effort":     {},
	"new_software_quality_security_remediation_effort":        {},
	"prioritized_rule_issues":                          {},
	"releasability_rating":                             {},
	"releasability_rating_with_aica":                   {},
	"releasability_rating_without_aica":                {},
	"software_quality_maintainability_issues":          {},
	"software_quality_maintainability_remediation_effort": {},
	"software_quality_reliability_issues":              {},
	"software_quality_reliability_rating":              {},
	"software_quality_reliability_remediation_effort":  {},
	"software_quality_security_issues":                 {},
	"software_quality_security_remediation_effort":     {},
}

// lookupMetricReplacement returns the list of SonarQube Cloud metrics that
// should be used in place of the given source metric. The bool indicates
// whether the source metric is known to the mapping table:
//
//   - ok=true,  len(targets)>0  → replace with these target metric(s)
//   - ok=true,  len(targets)==0 → drop the condition (no SQC equivalent)
//   - ok=false                  → metric exists verbatim on SQC, pass through
func lookupMetricReplacement(sourceMetric string) (targets []string, ok bool) {
	t, ok := metricMapping[sourceMetric]
	return t, ok
}
