// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package migrate

import (
	"strings"
	"testing"
)

// TestCombineBranchesAsRegex covers #249 behaviour: empty → empty,
// single value passes through (already a regex on SQS), two or more
// values wrapped as "(a|b|c)".
func TestCombineBranchesAsRegex(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want string
	}{
		{"nil input", nil, ""},
		{"empty slice", []string{}, ""},
		{"single value passes through", []string{"main"}, "main"},
		{"two values wrapped", []string{"main", "develop"}, "(main|develop)"},
		{"three values wrapped", []string{"main", "develop", "release-.*"}, "(main|develop|release-.*)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CombineBranchesAsRegex(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestDbCleanerBranchesTransformNote covers when a note should be
// surfaced in the report: only when 2+ values were combined.
func TestDbCleanerBranchesTransformNote(t *testing.T) {
	cases := []struct {
		name       string
		in         []string
		regex      string
		wantEmpty  bool
		wantSubstr string
	}{
		{"single value emits no note", []string{"main"}, "main", true, ""},
		{"empty emits no note", nil, "", true, ""},
		{"two values emits a note mentioning the regex",
			[]string{"main", "develop"}, "(main|develop)", false, "(main|develop)"},
		{"two values mentions the target key",
			[]string{"main", "develop"}, "(main|develop)", false, "sonar.branch.longLivedBranches.regex"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DbCleanerBranchesTransformNote(tc.in, tc.regex)
			if tc.wantEmpty {
				if got != "" {
					t.Errorf("expected empty note, got %q", got)
				}
				return
			}
			if !strings.Contains(got, tc.wantSubstr) {
				t.Errorf("note must contain %q, got %q", tc.wantSubstr, got)
			}
		})
	}
}
