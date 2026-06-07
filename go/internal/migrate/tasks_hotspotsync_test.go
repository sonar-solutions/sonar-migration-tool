// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package migrate

import (
	"testing"
)

// #356: filter now runs source-side directly on matchableHotspot —
// no longer pair-based, since we no longer pre-match against the
// full Cloud hotspot list before filtering.
func TestHotspotHasManualChanges(t *testing.T) {
	comment := hotspotComment{Login: "alice", Markdown: "please review"}

	tests := []struct {
		name string
		h    matchableHotspot
		want bool
	}{
		{name: "TO_REVIEW without comments — skip", h: matchableHotspot{Status: "TO_REVIEW"}, want: false},
		{name: "TO_REVIEW with comments — sync", h: matchableHotspot{Status: "TO_REVIEW", Comments: []hotspotComment{comment}}, want: true},
		// #350: REVIEWED without a resolution carries no payload.
		{name: "REVIEWED no resolution — skip", h: matchableHotspot{Status: "REVIEWED"}, want: false},
		{name: "REVIEWED + SAFE — sync", h: matchableHotspot{Status: "REVIEWED", Resolution: "SAFE"}, want: true},
		{name: "REVIEWED + ACKNOWLEDGED — sync", h: matchableHotspot{Status: "REVIEWED", Resolution: "ACKNOWLEDGED"}, want: true},
		{name: "REVIEWED + FIXED — sync", h: matchableHotspot{Status: "REVIEWED", Resolution: "FIXED"}, want: true},
		{name: "REVIEWED + unknown resolution — skip", h: matchableHotspot{Status: "REVIEWED", Resolution: "WHATEVER"}, want: false},
		{name: "REVIEWED + unknown resolution + comment — sync via comment", h: matchableHotspot{Status: "REVIEWED", Resolution: "WHATEVER", Comments: []hotspotComment{comment}}, want: true},
		{name: "case-insensitive status / resolution", h: matchableHotspot{Status: "reviewed", Resolution: "safe"}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hotspotHasManualChanges(tc.h)
			if got != tc.want {
				t.Errorf("hotspotHasManualChanges(%+v) = %v, want %v", tc.h, got, tc.want)
			}
		})
	}
}

// #356: hotspot classifier uses (ruleKey, line) because /api/hotspots/search
// doesn't accept a rules filter — we fetch by file and resolve in
// memory. 1 → synced, 0 → not_found, n>1 → line_mismatch; mismatched
// rules / lines are skipped.
func TestClassifyHotspotCandidatesByLine(t *testing.T) {
	cand := func(key, rule string, line int) matchableHotspot {
		return matchableHotspot{Key: key, RuleKey: rule, Line: line}
	}
	tests := []struct {
		name        string
		candidates  []matchableHotspot
		sourceRule  string
		sourceLine  int
		wantKey     string
		wantOutcome syncOutcome
	}{
		{
			name:        "exactly one rule+line match — synced (a)",
			candidates:  []matchableHotspot{cand("h-1", "javasecurity:S1", 42)},
			sourceRule:  "javasecurity:S1",
			sourceLine:  42,
			wantKey:     "h-1",
			wantOutcome: syncOutcomeSynced,
		},
		{
			name:        "match among other rules on same file — synced (a)",
			candidates:  []matchableHotspot{cand("h-1", "javasecurity:S1", 10), cand("h-2", "javasecurity:S1", 42), cand("h-3", "javasecurity:S2", 42)},
			sourceRule:  "javasecurity:S1",
			sourceLine:  42,
			wantKey:     "h-2",
			wantOutcome: syncOutcomeSynced,
		},
		{
			name:        "two same-rule+line — line_mismatch (b)",
			candidates:  []matchableHotspot{cand("h-1", "javasecurity:S1", 42), cand("h-2", "javasecurity:S1", 42)},
			sourceRule:  "javasecurity:S1",
			sourceLine:  42,
			wantOutcome: syncOutcomeLineMismatch,
		},
		{
			name:        "different rule on the line — not_found (c)",
			candidates:  []matchableHotspot{cand("h-1", "javasecurity:S2", 42)},
			sourceRule:  "javasecurity:S1",
			sourceLine:  42,
			wantOutcome: syncOutcomeNotFound,
		},
		{
			name:        "rule matches but on different line — not_found (c)",
			candidates:  []matchableHotspot{cand("h-1", "javasecurity:S1", 40)},
			sourceRule:  "javasecurity:S1",
			sourceLine:  42,
			wantOutcome: syncOutcomeNotFound,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, outcome := classifyHotspotCandidatesByLine(tc.candidates, tc.sourceRule, tc.sourceLine)
			if outcome != tc.wantOutcome {
				t.Errorf("outcome = %v, want %v", outcome, tc.wantOutcome)
			}
			if tc.wantOutcome == syncOutcomeSynced && got.Key != tc.wantKey {
				t.Errorf("pick = %q, want %q", got.Key, tc.wantKey)
			}
		})
	}
}
