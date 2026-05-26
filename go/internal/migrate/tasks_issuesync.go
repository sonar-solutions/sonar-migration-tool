package migrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"golang.org/x/sync/errgroup"
)

// issueMetadataSyncTasks returns the task definition for synchronising
// issue metadata (status transitions, comments, tags) from the extracted
// SonarQube Server data to the newly-created SonarQube Cloud issues.
//
// The task depends on importScanHistory because Cloud issues only exist
// after the scan report has been processed by the CE.
func issueMetadataSyncTasks() []TaskDef {
	return []TaskDef{{
		Name:         "syncIssueMetadata",
		Editions:     common.AllEditions,
		Dependencies: []string{"importScanHistory"},
		Run:          runSyncIssueMetadata,
	}}
}

// ---------------------------------------------------------------------------
// Domain types
// ---------------------------------------------------------------------------

// matchableIssue is a normalised issue representation used to pair source
// (SQS extract) issues with their Cloud counterparts. The struct is
// intentionally flat so that matching and filtering logic stays simple.
type matchableIssue struct {
	Key        string
	Rule       string
	Component  string
	Line       int
	Status     string
	Resolution string
	Tags       []string
	Comments   []issueComment
	Assignee   string
}

// issueComment is a normalised comment attached to a matchableIssue.
type issueComment struct {
	Login     string
	HTMLText  string
	Markdown  string
	CreatedAt string
}

// issuePair binds a source SQS issue to its Cloud counterpart after matching.
type issuePair struct {
	source matchableIssue
	cloud  matchableIssue
}

// ---------------------------------------------------------------------------
// Matching helpers
// ---------------------------------------------------------------------------

// buildIssueMatchKey produces a canonical key for matching issues across
// environments. The component field from SonarQube includes the project
// key as a prefix ("myproject:src/Foo.java"); we strip it so only the
// relative file path remains. The key format is "rule|filePath|line".
//
// Returns "" if the issue lacks a rule, component, or has a non-positive line.
func buildIssueMatchKey(iss matchableIssue) string {
	if iss.Rule == "" || iss.Component == "" || iss.Line <= 0 {
		return ""
	}
	// Strip "projectKey:" prefix from component.
	filePath := iss.Component
	if idx := strings.Index(filePath, ":"); idx >= 0 {
		filePath = filePath[idx+1:]
	}
	return fmt.Sprintf("%s|%s|%d", iss.Rule, filePath, iss.Line)
}

// matchIssues performs FIFO matching: for every source issue, take the first
// Cloud candidate with the same match key. Each Cloud issue is consumed at
// most once, preventing one-to-many duplication.
//
// The candidate map is built from cloudIssues (key -> []matchableIssue).
// Source issues are iterated in order; the first available candidate for
// each key is popped from the front of the slice (FIFO).
//
// All data structures are fully built before this function returns; no
// mutation occurs during any subsequent concurrent phase.
func matchIssues(sourceIssues, cloudIssues []matchableIssue) []issuePair {
	// Build candidate map: matchKey -> ordered slice of cloud issues.
	candidates := make(map[string][]matchableIssue, len(cloudIssues))
	for _, ci := range cloudIssues {
		k := buildIssueMatchKey(ci)
		if k == "" {
			continue
		}
		candidates[k] = append(candidates[k], ci)
	}

	// FIFO consume: iterate source issues and take the first available
	// cloud candidate for each match key.
	var pairs []issuePair
	for _, si := range sourceIssues {
		k := buildIssueMatchKey(si)
		if k == "" {
			continue
		}
		bucket := candidates[k]
		if len(bucket) == 0 {
			continue
		}
		// Pop the first candidate (FIFO).
		pairs = append(pairs, issuePair{source: si, cloud: bucket[0]})
		candidates[k] = bucket[1:]
	}
	return pairs
}

// ---------------------------------------------------------------------------
// Transition logic
// ---------------------------------------------------------------------------

// getFallbackTransition maps an SQS issue's resolution and status to the
// Cloud transition name required to move the Cloud issue into an equivalent
// state.
//
// Resolution takes priority (it is the most specific signal). When the
// resolution is empty or unrecognised, we fall back to the status field.
//
// Returns "" when no transition is needed (e.g. OPEN issues).
func getFallbackTransition(resolution, status string) string {
	// Resolution-based priority — most specific.
	switch strings.ToUpper(resolution) {
	case "FALSE-POSITIVE":
		return "falsepositive"
	case "WONTFIX":
		return "wontfix"
	}

	// Status-based fallback.
	switch strings.ToUpper(status) {
	case "CONFIRMED":
		return "confirm"
	case "REOPENED":
		return "reopen"
	case "OPEN":
		return ""
	case "RESOLVED", "CLOSED":
		return "resolve"
	case "ACCEPTED":
		return "wontfix"
	case "FALSE_POSITIVE":
		return "falsepositive"
	case "IN_SANDBOX":
		return ""
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// Filtering
// ---------------------------------------------------------------------------

// metadataSyncTag is the idempotency marker applied to Cloud issues after
// their metadata has been synchronised. Its presence prevents redundant
// re-application on subsequent runs.
const metadataSyncTag = "metadata-synchronized"

// hasManualChanges returns true when the source issue carries metadata
// that should be propagated to Cloud — i.e. the issue was manually
// triaged on the source server. Issues that have never been touched
// (OPEN, no comments, no tags, no assignee) are skipped to avoid
// unnecessary API calls.
func hasManualChanges(iss matchableIssue) bool {
	// Non-migrated comments (we consider all source comments relevant).
	if len(iss.Comments) > 0 {
		return true
	}
	// Tags.
	if len(iss.Tags) > 0 {
		return true
	}
	// Assignee.
	if iss.Assignee != "" {
		return true
	}
	// Non-OPEN status.
	if strings.ToUpper(iss.Status) != "OPEN" {
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Backoff / Cloud indexing wait
// ---------------------------------------------------------------------------

// waitForCloudIndexing polls fetchFn with exponential backoff until a
// non-zero total is returned or the maximum number of retries is exhausted.
//
// This accommodates the delay between CE task completion and the issues
// becoming searchable via /api/issues/search. If the max retries are
// exhausted without results the function returns nil (non-fatal) so the
// sync proceeds with zero matches — the alternative would be a hard
// failure that blocks later projects unnecessarily.
func waitForCloudIndexing(ctx context.Context, fetchFn func() (int, error)) error {
	const (
		initialDelay = 10 * time.Second
		maxDelay     = 60 * time.Second
		maxRetries   = 10
	)

	delay := initialDelay
	for attempt := 0; attempt < maxRetries; attempt++ {
		total, err := fetchFn()
		if err != nil {
			return err
		}
		if total > 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		// Exponential backoff capped at maxDelay.
		delay = delay * 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}
	// Non-fatal: proceed with 0 matches.
	return nil
}

// ---------------------------------------------------------------------------
// Expected error classification
// ---------------------------------------------------------------------------

// isExpectedTransitionError returns true for API errors that are harmless
// and expected during transition replay. The most common case is a 400 on
// the "reopen" transition when the Cloud issue is already OPEN.
func isExpectedTransitionError(err error, transition string) bool {
	var apiErr *sqapi.APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	// Cloud returns 400 when a transition is invalid for the current
	// issue state (e.g. reopen on an already-OPEN issue).
	return apiErr.StatusCode == 400 && transition == "reopen"
}

// ---------------------------------------------------------------------------
// Main task entry point
// ---------------------------------------------------------------------------

// runSyncIssueMetadata iterates every migrated project and synchronises
// the issue metadata (transitions, comments, tags) from the SQS extract
// to the corresponding Cloud issues.
func runSyncIssueMetadata(ctx context.Context, e *Executor) error {
	counter := NewTaskCounter("syncIssueMetadata")
	err := forEachMigrateItem(ctx, e, "syncIssueMetadata", "createProjects",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			cloudKey := extractField(item, "cloud_project_key")
			orgKey := extractField(item, "sonarcloud_org_key")
			serverURL := extractField(item, "server_url")
			serverKey := extractField(item, "key")
			if cloudKey == "" || orgKey == "" {
				return nil
			}
			return syncProjectIssues(ctx, e, cloudKey, orgKey, serverURL, serverKey, counter)
		})
	counter.LogSummary(e.Logger)
	return err
}

// ---------------------------------------------------------------------------
// Per-project sync
// ---------------------------------------------------------------------------

// syncProjectIssues handles the full issue-metadata sync for a single project:
//
//  1. Load source issues from extract
//  2. Wait for Cloud indexing
//  3. Fetch Cloud issues
//  4. Match (FIFO)
//  5. Pre-filter (hasManualChanges)
//  6. Sync pairs with bounded concurrency
//  7. Log summary
func syncProjectIssues(ctx context.Context, e *Executor, cloudKey, orgKey, serverURL, serverKey string, counter *TaskCounter) error {
	// 1. Load source issues from extract data.
	sourceIssues := loadMatchableIssues(e, serverURL, serverKey)
	if len(sourceIssues) == 0 {
		e.Logger.Debug("syncIssueMetadata: no source issues", "project", cloudKey)
		return nil
	}

	// 2. Wait for Cloud indexing to complete before fetching.
	err := waitForCloudIndexing(ctx, func() (int, error) {
		params := url.Values{}
		params.Set("componentKeys", cloudKey)
		params.Set("organization", orgKey)
		params.Set("ps", "1")
		params.Set("p", "1")
		// Use a lightweight single-page probe to check whether any
		// issues have been indexed yet. SearchAll respects ps=1 so we
		// get at most one issue back.
		issues, fetchErr := e.Cloud.Issues.SearchAll(ctx, params)
		if fetchErr != nil {
			return 0, fetchErr
		}
		return len(issues), nil
	})
	if err != nil {
		logAPIWarn(e.Logger, "syncIssueMetadata: indexing wait failed", err, "project", cloudKey)
		return nil // non-fatal
	}

	// 3. Fetch all Cloud issues for the project.
	cloudIssues, err := loadCloudMatchableIssues(ctx, e, cloudKey, orgKey)
	if err != nil {
		logAPIWarn(e.Logger, "syncIssueMetadata: fetch cloud issues failed", err, "project", cloudKey)
		return nil // non-fatal — skip project
	}
	if len(cloudIssues) == 0 {
		e.Logger.Debug("syncIssueMetadata: no cloud issues to match", "project", cloudKey)
		return nil
	}

	// 4. Match issues (FIFO). Built entirely before launching goroutines.
	matchedPairs := matchIssues(sourceIssues, cloudIssues)
	if len(matchedPairs) == 0 {
		e.Logger.Debug("syncIssueMetadata: no matched pairs", "project", cloudKey)
		return nil
	}

	// 5. Pre-filter: only sync pairs where the source has manual changes.
	var actionable []issuePair
	for _, p := range matchedPairs {
		if hasManualChanges(p.source) {
			actionable = append(actionable, p)
		}
	}
	if len(actionable) == 0 {
		e.Logger.Debug("syncIssueMetadata: no actionable pairs after filter", "project", cloudKey)
		return nil
	}

	e.Logger.Info("syncIssueMetadata: syncing pairs",
		"project", cloudKey,
		"matched", len(matchedPairs),
		"actionable", len(actionable),
	)

	// 6. Sync pairs with bounded concurrency.
	//
	// RACE-CONDITION SAFETY:
	//   - actionable slice is read-only during this phase.
	//   - Each goroutine receives exactly ONE issuePair by value.
	//   - counter uses atomic operations (existing pattern).
	//   - No shared mutable state is accessed.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(cap(e.Sem))
	for _, pair := range actionable {
		g.Go(func() error {
			if gctx.Err() != nil {
				return gctx.Err()
			}
			syncOnePair(gctx, e, pair, counter)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	return nil
}

// ---------------------------------------------------------------------------
// Per-pair sync
// ---------------------------------------------------------------------------

// syncOnePair synchronises a single source-to-cloud issue pair.
// Operations are strictly sequential within the pair to maintain
// ordering guarantees (transition before comments before tags).
//
// Idempotency: if the cloud issue already carries the metadataSyncTag
// the pair is skipped entirely.
func syncOnePair(ctx context.Context, e *Executor, pair issuePair, counter *TaskCounter) {
	// Idempotency check: skip if already tagged.
	if slices.Contains(pair.cloud.Tags, metadataSyncTag) {
		return
	}

	cloudKey := pair.cloud.Key

	// --- Transition ---
	transition := getFallbackTransition(pair.source.Resolution, pair.source.Status)
	if transition != "" {
		e.Logger.Debug("syncIssueMetadata: transition",
			"issue", cloudKey, "transition", transition)
		if err := e.Cloud.Issues.DoTransition(ctx, cloudKey, transition); err != nil {
			if !isExpectedTransitionError(err, transition) {
				logAPIWarn(e.Logger, "syncIssueMetadata: transition failed", err,
					"issue", cloudKey, "transition", transition)
				counter.Fail()
				return
			}
			// Expected error (e.g. reopen on OPEN) — continue.
		}
	}

	// --- Comments ---
	for _, c := range pair.source.Comments {
		text := c.Markdown
		if text == "" {
			text = c.HTMLText
		}
		if text == "" {
			continue
		}
		// Prefix with original author and timestamp for audit trail.
		prefix := fmt.Sprintf("[Migrated from %s", c.Login)
		if c.CreatedAt != "" {
			prefix += " on " + c.CreatedAt
		}
		prefix += "]\n\n"

		e.Logger.Debug("syncIssueMetadata: add comment",
			"issue", cloudKey, "login", c.Login)
		if err := e.Cloud.Issues.AddComment(ctx, cloudKey, prefix+text); err != nil {
			logAPIWarn(e.Logger, "syncIssueMetadata: add comment failed", err,
				"issue", cloudKey, "login", c.Login)
			counter.Fail()
			return
		}
	}

	// --- Tags ---
	// Merge source tags with the idempotency marker.
	tags := make([]string, 0, len(pair.source.Tags)+1)
	tags = append(tags, pair.source.Tags...)
	if !slices.Contains(tags, metadataSyncTag) {
		tags = append(tags, metadataSyncTag)
	}

	e.Logger.Debug("syncIssueMetadata: set tags",
		"issue", cloudKey, "tags", tags)
	if err := e.Cloud.Issues.SetTags(ctx, cloudKey, tags); err != nil {
		logAPIWarn(e.Logger, "syncIssueMetadata: set tags failed", err,
			"issue", cloudKey)
		counter.Fail()
		return
	}

	counter.Success()
}

// ---------------------------------------------------------------------------
// Extract loaders
// ---------------------------------------------------------------------------

// loadMatchableIssues reads the extracted SQS issues for a project and
// converts them to matchableIssue values. Issues with CLOSED status or
// FIXED resolution are excluded because they have no Cloud counterpart
// (the scan report does not reproduce them).
func loadMatchableIssues(e *Executor, serverURL, serverKey string) []matchableIssue {
	items, err := readExtractItems(e, "getProjectIssuesFull")
	if err != nil {
		return nil
	}

	var issues []matchableIssue
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}
		if extractField(item.Data, "projectKey") != serverKey {
			continue
		}

		status := strings.ToUpper(extractField(item.Data, "status"))
		resolution := strings.ToUpper(extractField(item.Data, "resolution"))

		// Exclude CLOSED and FIXED — these won't exist in Cloud.
		if status == "CLOSED" {
			continue
		}
		if resolution == "FIXED" {
			continue
		}

		line := int(extractInt32Field(item.Data, "line"))

		comments := parseIssueComments(item.Data)
		tags := extractStringArray(item.Data, "tags")

		issues = append(issues, matchableIssue{
			Key:        extractField(item.Data, "key"),
			Rule:       extractField(item.Data, "rule"),
			Component:  extractField(item.Data, "component"),
			Line:       line,
			Status:     extractField(item.Data, "status"),
			Resolution: extractField(item.Data, "resolution"),
			Tags:       tags,
			Comments:   comments,
			Assignee:   extractField(item.Data, "assignee"),
		})
	}
	return issues
}

// loadCloudMatchableIssues fetches all issues from Cloud for a given
// project and converts them to matchableIssue values.
func loadCloudMatchableIssues(ctx context.Context, e *Executor, cloudKey, orgKey string) ([]matchableIssue, error) {
	params := url.Values{}
	params.Set("componentKeys", cloudKey)
	params.Set("organization", orgKey)
	// Fetch all statuses to enable accurate matching.
	params.Set("statuses", "OPEN,CONFIRMED,REOPENED,RESOLVED,CLOSED")

	apiIssues, err := e.Cloud.Issues.SearchAll(ctx, params)
	if err != nil {
		return nil, err
	}

	issues := make([]matchableIssue, 0, len(apiIssues))
	for _, ai := range apiIssues {
		comments := make([]issueComment, 0, len(ai.Comments))
		for _, c := range ai.Comments {
			comments = append(comments, issueComment{
				Login:     c.Login,
				HTMLText:  c.HTMLText,
				Markdown:  c.Markdown,
				CreatedAt: c.CreatedAt,
			})
		}
		issues = append(issues, matchableIssue{
			Key:        ai.Key,
			Rule:       ai.Rule,
			Component:  ai.Component,
			Line:       ai.Line,
			Status:     ai.Status,
			Resolution: ai.Resolution,
			Tags:       ai.Tags,
			Comments:   comments,
			Assignee:   ai.Assignee,
		})
	}
	return issues, nil
}

// ---------------------------------------------------------------------------
// JSON helpers
// ---------------------------------------------------------------------------

// parseIssueComments extracts the "comments" array from an issue's raw
// JSON data and returns them as issueComment values.
func parseIssueComments(data json.RawMessage) []issueComment {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}
	commentsRaw, ok := obj["comments"]
	if !ok {
		return nil
	}
	var raw []struct {
		Login     string `json:"login"`
		HTMLText  string `json:"htmlText"`
		Markdown  string `json:"markdown"`
		CreatedAt string `json:"createdAt"`
	}
	if err := json.Unmarshal(commentsRaw, &raw); err != nil {
		return nil
	}
	comments := make([]issueComment, 0, len(raw))
	for _, r := range raw {
		comments = append(comments, issueComment{
			Login:     r.Login,
			HTMLText:  r.HTMLText,
			Markdown:  r.Markdown,
			CreatedAt: r.CreatedAt,
		})
	}
	return comments
}
