// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

// Package scanreport builds and submits SonarScanner-compatible reports
// to SonarCloud's Compute Engine for project data migration.
package scanreport

import (
	"sort"
	"time"

	pb "github.com/sonar-solutions/sonar-migration-tool/internal/scanreport/proto"
)

const (
	issueBatchSize = 5000
	oneDayMs       = int64(24 * time.Hour / time.Millisecond)
	stubAuthor     = "sonar-migration-tool@sonarcloud.io"
)

// ExtractedIssue holds the fields needed from an extracted SonarQube issue
// for changeset backdating.
type ExtractedIssue struct {
	Key          string
	Component    string
	CreationDate time.Time
	StartLine    int32
	EndLine      int32
}

// BackdateChangesets rewrites SCM changeset data so that the CE assigns each
// issue its original creation date. It builds per-file, per-line date maps
// from issue creation dates and reconstructs changeset entries accordingly.
//
// The changesets map is keyed by component key and modified in place.
func BackdateChangesets(issues []ExtractedIssue, changesets map[string]*pb.Changesets, fallbackDate time.Time) {
	if len(issues) == 0 {
		return
	}

	fallbackMs := fallbackDate.UnixMilli()
	overrides := buildSafetySplitOverrides(issues, fallbackMs)
	fileLineDates := buildFileLineDates(issues, overrides, fallbackMs)

	for compKey, lineDateMap := range fileLineDates {
		cs, ok := changesets[compKey]
		if !ok {
			continue
		}
		if len(cs.ChangesetIndexByLine) == 0 {
			continue
		}
		rebuildChangesetForFile(cs, lineDateMap)
	}
}

// rebuildChangesetForFile overrides the changeset DATE for each line that
// carries an issue, so SonarCloud's CE assigns each issue its original
// SonarQube creation date — while PRESERVING the real SCM author and revision
// already attributed to that line (from BuildChangesetsFromBlame). Lines
// without issues keep their real blame untouched, so the Code view still shows
// the true author/date for the rest of the file.
//
// Because buildFileLineDates fills every line of a multi-line issue's range
// with the same (oldest-wins) date, all of an issue's lines share one date and
// the CE's MAX-over-lines collapses to that date — no inflation.
func rebuildChangesetForFile(cs *pb.Changesets, lineDateMap map[int32]int64) {
	type csKey struct {
		rev, author string
		date        int64
	}
	index := make(map[csKey]int32, len(cs.Changeset))
	for i, e := range cs.Changeset {
		index[csKey{e.Revision, e.Author, e.Date}] = int32(i)
	}
	findOrAppend := func(rev, author string, date int64) int32 {
		k := csKey{rev, author, date}
		if i, ok := index[k]; ok {
			return i
		}
		i := int32(len(cs.Changeset))
		cs.Changeset = append(cs.Changeset, &pb.Changesets_Changeset{
			Revision: rev,
			Author:   author,
			Date:     date,
		})
		index[k] = i
		return i
	}

	n := len(cs.ChangesetIndexByLine)
	for ln, dateMs := range lineDateMap {
		i := int(ln) - 1
		if i < 0 || i >= n {
			continue
		}
		// Carry over the real author/revision on this line; only the date
		// changes to the issue's creation date. Falls back to the synthetic
		// stub author when the line had no blame entry.
		rev, author := "", stubAuthor
		if base := cs.ChangesetIndexByLine[i]; int(base) >= 0 && int(base) < len(cs.Changeset) {
			rev = cs.Changeset[base].Revision
			author = cs.Changeset[base].Author
		}
		cs.ChangesetIndexByLine[i] = findOrAppend(rev, author, dateMs)
	}
}

// collectSortedDates returns sorted unique date values from a line-date map.
func collectSortedDates(lineDateMap map[int32]int64) []int64 {
	seen := make(map[int64]struct{})
	for _, d := range lineDateMap {
		seen[d] = struct{}{}
	}
	dates := make([]int64, 0, len(seen))
	for d := range seen {
		dates = append(dates, d)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i] < dates[j] })
	return dates
}

// buildSafetySplitOverrides splits any calendar day with >issueBatchSize issues
// into sub-groups with synthetic 1-day-apart dates.
// Returns a map of issueKey -> overridden dateMs.
func buildSafetySplitOverrides(issues []ExtractedIssue, fallbackMs int64) map[string]int64 {
	overrides := make(map[string]int64)

	type dayGroup struct {
		dateMs int64
		issues []ExtractedIssue
	}
	groups := make(map[int64]*dayGroup)

	for _, iss := range issues {
		dateMs := issueDateMs(iss, fallbackMs)
		dayKey := dateMs / oneDayMs
		g, ok := groups[dayKey]
		if !ok {
			g = &dayGroup{dateMs: dayKey * oneDayMs}
			groups[dayKey] = g
		}
		g.issues = append(g.issues, iss)
	}

	for _, g := range groups {
		if len(g.issues) <= issueBatchSize {
			continue
		}
		batches := groupIssuesIntoBatches(g.issues)
		total := len(batches)
		for batchIdx := range total - 1 {
			syntheticDate := g.dateMs - int64(total-1-batchIdx)*oneDayMs
			for _, iss := range batches[batchIdx] {
				overrides[iss.Key] = syntheticDate
			}
		}
	}
	return overrides
}

// groupIssuesIntoBatches splits sorted issues into batches of <=issueBatchSize
// without splitting files across batches.
func groupIssuesIntoBatches(issues []ExtractedIssue) [][]ExtractedIssue {
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].Component < issues[j].Component
	})

	var batches [][]ExtractedIssue
	var current []ExtractedIssue
	var currentFile string
	var fileBuffer []ExtractedIssue

	flush := func() {
		if len(fileBuffer) == 0 {
			return
		}
		if len(current)+len(fileBuffer) > issueBatchSize && len(current) > 0 {
			batches = append(batches, current)
			current = nil
		}
		current = append(current, fileBuffer...)
		fileBuffer = nil
	}

	for i := range issues {
		if issues[i].Component != currentFile {
			flush()
			currentFile = issues[i].Component
		}
		fileBuffer = append(fileBuffer, issues[i])
	}
	flush()
	if len(current) > 0 {
		batches = append(batches, current)
	}
	return batches
}

// buildFileLineDates builds a per-file, per-line date map from issues.
// For overlapping lines, the oldest date wins (prevents CE MAX inflation).
func buildFileLineDates(issues []ExtractedIssue, overrides map[string]int64, fallbackMs int64) map[string]map[int32]int64 {
	result := make(map[string]map[int32]int64)
	for _, iss := range issues {
		applyIssueDates(iss, overrides, fallbackMs, result)
	}
	return result
}

func applyIssueDates(iss ExtractedIssue, overrides map[string]int64, fallbackMs int64, result map[string]map[int32]int64) {
	startLine := iss.StartLine
	endLine := iss.EndLine
	if endLine == 0 {
		endLine = startLine
	}
	if startLine <= 0 || iss.Component == "" {
		return
	}

	dateMs := overrides[iss.Key]
	if dateMs == 0 {
		dateMs = issueDateMs(iss, fallbackMs)
	}

	lineDateMap, ok := result[iss.Component]
	if !ok {
		lineDateMap = make(map[int32]int64)
		result[iss.Component] = lineDateMap
	}

	for ln := startLine; ln <= endLine; ln++ {
		if existing, ok := lineDateMap[ln]; !ok || dateMs < existing {
			lineDateMap[ln] = dateMs
		}
	}
}

// issueDateMs returns the issue creation date as epoch milliseconds,
// falling back to the provided default.
func issueDateMs(iss ExtractedIssue, fallbackMs int64) int64 {
	if iss.CreationDate.IsZero() {
		return fallbackMs
	}
	return iss.CreationDate.UnixMilli()
}
