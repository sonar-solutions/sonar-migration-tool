// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package migrate

import (
	"testing"
)

func TestIsAlreadyMigratedIssueComment(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		cloudComments []issueComment
		want         bool
	}{
		{
			name: "no cloud comments",
			body: "some comment text",
			cloudComments: nil,
			want: false,
		},
		{
			name: "cloud comment contains prefix and body",
			body: "some comment text",
			cloudComments: []issueComment{
				{Markdown: "[Migrated from admin on 2024-01-01]\n\nsome comment text"},
			},
			want: true,
		},
		{
			name: "cloud comment has prefix but different body",
			body: "some comment text",
			cloudComments: []issueComment{
				{Markdown: "[Migrated from admin on 2024-01-01]\n\ndifferent comment"},
			},
			want: false,
		},
		{
			name: "cloud comment contains body but no prefix",
			body: "some comment text",
			cloudComments: []issueComment{
				{Markdown: "some comment text"},
			},
			want: false,
		},
		{
			name: "matches on HTMLText when Markdown is empty",
			body: "html comment",
			cloudComments: []issueComment{
				{HTMLText: "[Migrated from bob]\n\nhtml comment"},
			},
			want: true,
		},
		{
			name: "prefers Markdown over HTMLText",
			body: "real body",
			cloudComments: []issueComment{
				{Markdown: "[Migrated from bob]\n\nreal body", HTMLText: "ignored"},
			},
			want: true,
		},
		{
			name: "no match among multiple cloud comments",
			body: "target body",
			cloudComments: []issueComment{
				{Markdown: "[Migrated from alice]\n\nother body"},
				{Markdown: "plain comment without prefix"},
			},
			want: false,
		},
		{
			name: "match found in second of multiple cloud comments",
			body: "target body",
			cloudComments: []issueComment{
				{Markdown: "[Migrated from alice]\n\nother body"},
				{Markdown: "[Migrated from bob]\n\ntarget body"},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isAlreadyMigratedIssueComment(tc.body, tc.cloudComments)
			if got != tc.want {
				t.Errorf("isAlreadyMigratedIssueComment(%q, ...) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

// #350: narrowed source-side candidate set. Sync only issues whose
// triage state, tags or comments mark them as touched by a human;
// auto-assigned issues and CONFIRMED-no-extras issues are skipped.
func TestHasManualChanges(t *testing.T) {
	comment := issueComment{Login: "alice", Markdown: "looks fine"}

	tests := []struct {
		name string
		iss  matchableIssue
		want bool
	}{
		// Triage state — primary triggers.
		{name: "status ACCEPTED", iss: matchableIssue{Status: "ACCEPTED"}, want: true},
		{name: "status accepted lowercase", iss: matchableIssue{Status: "accepted"}, want: true},
		// Modern unified status enum — issueStatus folded into the
		// `status` field on post-10.4 servers.
		{name: "status FALSE_POSITIVE (modern)", iss: matchableIssue{Status: "FALSE_POSITIVE"}, want: true},
		{name: "status false_positive lowercase", iss: matchableIssue{Status: "false_positive"}, want: true},
		// Legacy resolution surface (pre-10.4 servers).
		{name: "resolution FALSE-POSITIVE (legacy)", iss: matchableIssue{Status: "RESOLVED", Resolution: "FALSE-POSITIVE"}, want: true},
		{name: "resolution false-positive lowercase", iss: matchableIssue{Status: "RESOLVED", Resolution: "false-positive"}, want: true},
		{name: "resolution WONTFIX (legacy)", iss: matchableIssue{Status: "RESOLVED", Resolution: "WONTFIX"}, want: true},
		// manualSeverity trigger.
		{name: "manualSeverity only", iss: matchableIssue{Status: "OPEN", ManualSeverity: true}, want: true},
		// User tags trigger (matchableIssue.Tags is already
		// user-only — rule defaults are subtracted at load time).
		{name: "user tags only", iss: matchableIssue{Status: "OPEN", Tags: []string{"flagged-by-team"}}, want: true},
		// Comments trigger.
		{name: "comments only", iss: matchableIssue{Status: "OPEN", Comments: []issueComment{comment}}, want: true},
		// Anti-triggers per #350 spec.
		{name: "OPEN, no triage signals — skip", iss: matchableIssue{Status: "OPEN"}, want: false},
		{name: "CONFIRMED with nothing else — skip (excluded per spec)", iss: matchableIssue{Status: "CONFIRMED"}, want: false},
		{name: "assignee only — skip (dropped per spec)", iss: matchableIssue{Status: "OPEN", Assignee: "bob"}, want: false},
		{name: "RESOLVED+FIXED with no other signal — skip", iss: matchableIssue{Status: "RESOLVED", Resolution: "FIXED"}, want: false},
		// Combinations — any one signal flips it.
		{name: "CONFIRMED + comment — sync", iss: matchableIssue{Status: "CONFIRMED", Comments: []issueComment{comment}}, want: true},
		{name: "assignee + tags — sync (via tags)", iss: matchableIssue{Status: "OPEN", Assignee: "bob", Tags: []string{"flagged"}}, want: true},
		{name: "manualSeverity + OPEN no other — sync", iss: matchableIssue{Status: "OPEN", ManualSeverity: true}, want: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := hasManualChanges(tc.iss)
			if got != tc.want {
				t.Errorf("hasManualChanges(%+v) = %v, want %v", tc.iss, got, tc.want)
			}
		})
	}
}

// #352-followup: issue.tags from /api/issues/search is the union of
// the rule's tags+sysTags and any user-added tags. The "rule defaults"
// alone trigger `len(tags) > 0` on essentially every issue, so we
// subtract them at load time.
func TestRuleTagDefaultsUserTagsOnly(t *testing.T) {
	r := &ruleTagDefaults{
		bySrv: map[string]map[string]map[string]struct{}{
			"https://server1": {
				"java:S1481": setOf("unused", "convention"),
				"java:S2095":  setOf("cwe", "denial-of-service", "security"),
			},
		},
	}

	tests := []struct {
		name      string
		serverURL string
		ruleKey   string
		allTags   []string
		want      []string
	}{
		{
			name: "all tags are rule defaults — returns empty",
			serverURL: "https://server1", ruleKey: "java:S1481",
			allTags: []string{"unused", "convention"},
			want:    []string{},
		},
		{
			name: "rule defaults stripped, user tags retained",
			serverURL: "https://server1", ruleKey: "java:S2095",
			allTags: []string{"cwe", "denial-of-service", "security", "flagged-by-team", "needs-investigation"},
			want:    []string{"flagged-by-team", "needs-investigation"},
		},
		{
			name: "rule not indexed — fallback returns input unchanged",
			serverURL: "https://server1", ruleKey: "kotlin:S100",
			allTags: []string{"convention"},
			want:    []string{"convention"},
		},
		{
			name: "server not indexed — fallback returns input unchanged",
			serverURL: "https://other-server", ruleKey: "java:S1481",
			allTags: []string{"unused"},
			want:    []string{"unused"},
		},
		{
			name: "empty input — returns empty",
			serverURL: "https://server1", ruleKey: "java:S1481",
			allTags: nil,
			want:    nil,
		},
		{
			name: "user tags only (no rule defaults present)",
			serverURL: "https://server1", ruleKey: "java:S1481",
			allTags: []string{"team-quarantine"},
			want:    []string{"team-quarantine"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := r.UserTagsOnly(tc.serverURL, tc.ruleKey, tc.allTags)
			if !equalStrings(got, tc.want) {
				t.Errorf("UserTagsOnly(%q, %q, %v) = %v, want %v", tc.serverURL, tc.ruleKey, tc.allTags, got, tc.want)
			}
		})
	}

	// Nil receiver — guards a caller that hasn't loaded the index.
	t.Run("nil receiver — passthrough", func(t *testing.T) {
		var nilR *ruleTagDefaults
		got := nilR.UserTagsOnly("https://server1", "java:S1481", []string{"cwe"})
		if !equalStrings(got, []string{"cwe"}) {
			t.Errorf("nil receiver should pass through, got %v", got)
		}
	})
}

func setOf(tags ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		out[t] = struct{}{}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// #356: stripProjectKeyPrefix isolates the file-path part of a
// SonarQube component identifier. The substitution is essential for
// the targeted cloud search: we send componentKeys=<cloudKey>:<file>
// derived from the source's <sourceKey>:<file> string.
func TestStripProjectKeyPrefix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "leading project key", in: "myproject:src/main/java/Foo.java", want: "src/main/java/Foo.java"},
		{name: "nested colon stays after first", in: "myproject:com:foo/Bar.java", want: "com:foo/Bar.java"},
		{name: "no project prefix", in: "src/main/java/Foo.java", want: "src/main/java/Foo.java"},
		{name: "empty", in: "", want: ""},
		{name: "colon only", in: "myproject:", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := stripProjectKeyPrefix(tc.in)
			if got != tc.want {
				t.Errorf("stripProjectKeyPrefix(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// Per-reason breakdown surfaced at the start of each per-project
// sync. An issue can trip multiple signals (e.g. ACCEPTED + tags); it
// must be counted once per signal so operators can see what's driving
// the migration cost. Branch count is a separate signal taken from
// distinct branch names on the source issues.
func TestClassifyActionableReasonsAndBranches(t *testing.T) {
	comment := issueComment{Login: "u", Markdown: "noted"}
	issues := []matchableIssue{
		{Status: "ACCEPTED", Branch: "main"},                                                       // accepted
		{Status: "RESOLVED", Resolution: "FALSE-POSITIVE", Branch: "main"},                          // accepted_or_fp (via resolution)
		{Status: "FALSE_POSITIVE", Branch: "develop"},                                               // accepted_or_fp (modern enum)
		{Status: "OPEN", Tags: []string{"flagged"}, Branch: "develop"},                              // custom_tags
		{Status: "OPEN", ManualSeverity: true, Branch: "feat-1"},                                    // manual_severity
		{Status: "OPEN", Comments: []issueComment{comment}, Branch: "feat-2"},                       // comments
		{Status: "ACCEPTED", Tags: []string{"audited"}, Branch: "main"},                             // accepted + tags (counted in both)
		{Status: "ACCEPTED", Comments: []issueComment{comment}, ManualSeverity: true, Branch: ""},   // accepted + manualSeverity + comments
	}
	b := classifyActionableReasons(issues)
	if want := 5; b.acceptedOrFP != want {
		t.Errorf("acceptedOrFP: want %d, got %d", want, b.acceptedOrFP)
	}
	if want := 2; b.customTags != want {
		t.Errorf("customTags: want %d, got %d", want, b.customTags)
	}
	if want := 2; b.manualSeverity != want {
		t.Errorf("manualSeverity: want %d, got %d", want, b.manualSeverity)
	}
	if want := 2; b.comments != want {
		t.Errorf("comments: want %d, got %d", want, b.comments)
	}
	// Distinct non-empty branches: main, develop, feat-1, feat-2 = 4.
	if want := 4; countDistinctBranches(issues) != want {
		t.Errorf("countDistinctBranches: want %d, got %d", want, countDistinctBranches(issues))
	}
}

// #356: case a/b/c classification — exactly the semantics from the
// issue. 1 cloud counterpart on the source line → synced (a); 0
// counterparts on the line → not_found (c); 2+ on the same line →
// line_mismatch (b). Off-line candidates are ignored.
func TestClassifyIssueCandidatesByLine(t *testing.T) {
	cand := func(key string, line int) matchableIssue {
		return matchableIssue{Key: key, Line: line}
	}
	tests := []struct {
		name        string
		candidates  []matchableIssue
		sourceLine  int
		wantKey     string
		wantOutcome syncOutcome
	}{
		{
			name:        "exactly one match on line — synced (a)",
			candidates:  []matchableIssue{cand("cloud-1", 42)},
			sourceLine:  42,
			wantKey:     "cloud-1",
			wantOutcome: syncOutcomeSynced,
		},
		{
			name:        "one match among off-line candidates — synced (a)",
			candidates:  []matchableIssue{cand("cloud-a", 40), cand("cloud-b", 42), cand("cloud-c", 44)},
			sourceLine:  42,
			wantKey:     "cloud-b",
			wantOutcome: syncOutcomeSynced,
		},
		{
			name:        "two matches on same line — line_mismatch (b)",
			candidates:  []matchableIssue{cand("cloud-a", 42), cand("cloud-b", 42)},
			sourceLine:  42,
			wantOutcome: syncOutcomeLineMismatch,
		},
		{
			name:        "three matches on same line, one off-line — line_mismatch (b)",
			candidates:  []matchableIssue{cand("cloud-a", 42), cand("cloud-b", 42), cand("cloud-c", 42), cand("cloud-d", 99)},
			sourceLine:  42,
			wantOutcome: syncOutcomeLineMismatch,
		},
		{
			name:        "no matches on line, candidates elsewhere — not_found (c)",
			candidates:  []matchableIssue{cand("cloud-a", 40), cand("cloud-b", 44)},
			sourceLine:  42,
			wantOutcome: syncOutcomeNotFound,
		},
		{
			name:        "empty candidates — not_found (c)",
			candidates:  nil,
			sourceLine:  42,
			wantOutcome: syncOutcomeNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, outcome := classifyIssueCandidatesByLine(tc.candidates, tc.sourceLine)
			if outcome != tc.wantOutcome {
				t.Errorf("outcome = %v, want %v", outcome, tc.wantOutcome)
			}
			if tc.wantOutcome == syncOutcomeSynced && got.Key != tc.wantKey {
				t.Errorf("pick = %q, want %q", got.Key, tc.wantKey)
			}
		})
	}
}
