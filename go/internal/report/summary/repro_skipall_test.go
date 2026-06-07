// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package summary

import (
	"strings"
	"testing"
)

// #359 follow-up: when --skip_project_data_migration is set, the
// per-project Details column for every successful project must
// stay clean. The v1 of this PR replaced the old "project data:
// Yes" wording with a per-project "Project data migration skipped:
// skipped by configuration" line on EVERY row, which was noise.
// The signal now lives once in the report-level Limitations.
func TestSkipProjectDataMigration_PerProjectStaysClean(t *testing.T) {
	dir := t.TempDir()
	writeTaskJSONL(t, dir, "createProjects", []map[string]any{
		{"name": "ProjA", "sonarcloud_org_key": "org1", "cloud_project_key": "org1_proja"},
		{"name": "ProjB", "sonarcloud_org_key": "org1", "cloud_project_key": "org1_projb"},
	})
	// run_meta.json signals importProjectData did NOT run.
	writeRunMeta(t, dir, map[string]any{
		"tasks": []map[string]any{
			{"name": "createProjects"},
			{"name": "setProjectGates"},
		},
	})
	// No importProjectData / syncIssueMetadata / syncHotspotMetadata records.

	summary, err := CollectSummary(dir, "")
	if err != nil {
		t.Fatalf("CollectSummary: %v", err)
	}
	projSection := findSection(summary, "Projects")
	if projSection == nil {
		t.Fatal("missing Projects section")
	}

	if len(projSection.Succeeded) != 2 {
		t.Fatalf("expected 2 succeeded projects, got %d", len(projSection.Succeeded))
	}
	for _, item := range projSection.Succeeded {
		rendered := partialDetails(item, false, false, true)
		for _, banned := range []string{"Project data migration skipped", "project data:", "skipped by configuration"} {
			if strings.Contains(rendered, banned) {
				t.Errorf("%s: unexpected substring %q in Details:\n%s", item.Name, banned, rendered)
			}
		}
	}

	// The report-level Limitations section MUST surface the global skip
	// once, so the signal is still discoverable.
	foundLimit := false
	for _, l := range summary.Limitations {
		if strings.Contains(l, "Project data migration was skipped by configuration") {
			foundLimit = true
			break
		}
	}
	if !foundLimit {
		t.Errorf("expected a Limitations entry about the global skip; got %v", summary.Limitations)
	}
}
