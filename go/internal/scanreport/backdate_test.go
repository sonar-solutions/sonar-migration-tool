package scanreport

import (
	"testing"
	"time"

	pb "github.com/sonar-solutions/sonar-migration-tool/internal/scanreport/proto"
)

func TestBackdateChangesetsEmpty(t *testing.T) {
	changesets := make(map[string]*pb.Changesets)
	BackdateChangesets(nil, changesets, time.Now())
	if len(changesets) != 0 {
		t.Error("expected no changesets for empty issues")
	}
}

func TestBackdateChangesetsBasic(t *testing.T) {
	now := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)
	issueDate := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	issues := []ExtractedIssue{
		{Key: "iss1", Component: "file.go", CreationDate: issueDate, StartLine: 5, EndLine: 10},
	}

	changesets := map[string]*pb.Changesets{
		"file.go": {
			ComponentRef: 2,
			Changeset: []*pb.Changesets_Changeset{
				{Revision: "original", Author: "dev@co.com", Date: now.UnixMilli()},
			},
			ChangesetIndexByLine: make([]int32, 20), // 20 lines
		},
	}

	BackdateChangesets(issues, changesets, now)

	cs := changesets["file.go"]
	if len(cs.Changeset) == 0 {
		t.Fatal("expected at least one changeset entry")
	}

	// Lines 5-10 should have the issue date, not the original date.
	for ln := int32(5); ln <= 10; ln++ {
		idx := cs.ChangesetIndexByLine[ln-1]
		entry := cs.Changeset[idx]
		if entry.Date != issueDate.UnixMilli() {
			t.Errorf("line %d: expected date %d, got %d", ln, issueDate.UnixMilli(), entry.Date)
		}
	}

	// Lines outside range should have index 0 (oldest date).
	if cs.ChangesetIndexByLine[0] != 0 {
		t.Errorf("line 1: expected index 0, got %d", cs.ChangesetIndexByLine[0])
	}
}

func TestBackdateChangesetsOldestWins(t *testing.T) {
	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	fallback := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Two issues overlap on lines 5-8.
	issues := []ExtractedIssue{
		{Key: "iss1", Component: "f.go", CreationDate: newer, StartLine: 3, EndLine: 8},
		{Key: "iss2", Component: "f.go", CreationDate: old, StartLine: 5, EndLine: 10},
	}

	changesets := map[string]*pb.Changesets{
		"f.go": {
			ComponentRef:         2,
			Changeset:            []*pb.Changesets_Changeset{{Date: fallback.UnixMilli()}},
			ChangesetIndexByLine: make([]int32, 15),
		},
	}

	BackdateChangesets(issues, changesets, fallback)

	cs := changesets["f.go"]
	// Lines 5-8 overlap: oldest (old) should win.
	for ln := int32(5); ln <= 8; ln++ {
		idx := cs.ChangesetIndexByLine[ln-1]
		entry := cs.Changeset[idx]
		if entry.Date != old.UnixMilli() {
			t.Errorf("line %d: expected oldest date %d, got %d", ln, old.UnixMilli(), entry.Date)
		}
	}
}

func TestBackdateChangesetsFallbackDate(t *testing.T) {
	fallback := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	issues := []ExtractedIssue{
		{Key: "iss1", Component: "f.go", StartLine: 1, EndLine: 1}, // zero CreationDate
	}

	changesets := map[string]*pb.Changesets{
		"f.go": {
			ComponentRef:         2,
			Changeset:            []*pb.Changesets_Changeset{{Date: fallback.UnixMilli()}},
			ChangesetIndexByLine: make([]int32, 5),
		},
	}

	BackdateChangesets(issues, changesets, fallback)

	cs := changesets["f.go"]
	idx := cs.ChangesetIndexByLine[0]
	entry := cs.Changeset[idx]
	if entry.Date != fallback.UnixMilli() {
		t.Errorf("expected fallback date %d, got %d", fallback.UnixMilli(), entry.Date)
	}
}

func TestBackdateChangesetsSkipsZeroLines(t *testing.T) {
	issues := []ExtractedIssue{
		{Key: "iss1", Component: "f.go", StartLine: 0, EndLine: 0}, // no line info
	}

	changesets := map[string]*pb.Changesets{
		"f.go": {
			ComponentRef:         2,
			Changeset:            []*pb.Changesets_Changeset{{Date: 1000}},
			ChangesetIndexByLine: make([]int32, 5),
		},
	}

	BackdateChangesets(issues, changesets, time.Now())

	// Changesets should remain unchanged (no issue lines to backdate).
	cs := changesets["f.go"]
	if len(cs.Changeset) != 1 || cs.Changeset[0].Date != 1000 {
		t.Error("expected changesets unchanged for zero-line issues")
	}
}

func TestGroupIssuesIntoBatches(t *testing.T) {
	// Create 12000 issues across 3 files.
	var issues []ExtractedIssue
	for i := range 4000 {
		issues = append(issues, ExtractedIssue{Key: "a", Component: "fileA.go", StartLine: int32(i + 1), EndLine: int32(i + 1)})
	}
	for i := range 4000 {
		issues = append(issues, ExtractedIssue{Key: "b", Component: "fileB.go", StartLine: int32(i + 1), EndLine: int32(i + 1)})
	}
	for i := range 4000 {
		issues = append(issues, ExtractedIssue{Key: "c", Component: "fileC.go", StartLine: int32(i + 1), EndLine: int32(i + 1)})
	}

	batches := groupIssuesIntoBatches(issues)
	if len(batches) < 2 {
		t.Errorf("expected at least 2 batches for 12000 issues, got %d", len(batches))
	}

	totalIssues := 0
	for _, b := range batches {
		totalIssues += len(b)
		if len(b) > issueBatchSize {
			t.Errorf("batch exceeds %d: got %d", issueBatchSize, len(b))
		}
	}
	if totalIssues != 12000 {
		t.Errorf("expected 12000 total issues, got %d", totalIssues)
	}
}

func TestCollectSortedDates(t *testing.T) {
	m := map[int32]int64{
		1: 300,
		2: 100,
		3: 200,
		4: 100,
	}
	dates := collectSortedDates(m)
	if len(dates) != 3 {
		t.Fatalf("expected 3 unique dates, got %d", len(dates))
	}
	if dates[0] != 100 || dates[1] != 200 || dates[2] != 300 {
		t.Errorf("expected [100 200 300], got %v", dates)
	}
}
