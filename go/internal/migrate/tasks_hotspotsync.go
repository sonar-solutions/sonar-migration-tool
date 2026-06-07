// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sonar-solutions/sonar-migration-tool/internal/common"
	"github.com/sonar-solutions/sonar-migration-tool/internal/structure"
)

// hotspotMetadataSyncTasks returns the task definitions for syncing hotspot
// metadata (status, resolution, comments) from SonarQube Server to Cloud.
func hotspotMetadataSyncTasks() []TaskDef {
	return []TaskDef{{
		Name:         "syncHotspotMetadata",
		Editions:     common.AllEditions,
		Dependencies: []string{"importProjectData"},
		Run:          runSyncHotspotMetadata,
	}}
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// matchableHotspot is a normalised hotspot representation used for FIFO
// matching between source (SonarQube Server) and target (SonarQube Cloud).
type matchableHotspot struct {
	Key        string
	RuleKey    string
	Component  string
	Line       int
	Status     string
	Resolution string
	Comments   []hotspotComment
}

// hotspotComment captures a single comment attached to a hotspot.
type hotspotComment struct {
	Login     string
	HTMLText  string
	Markdown  string
	CreatedAt string
}

// hotspotPair links a source hotspot to its matched Cloud counterpart.
type hotspotPair struct {
	source matchableHotspot
	cloud  matchableHotspot
}

// ---------------------------------------------------------------------------
// Actionable filtering (source-side, #350 / #356)
// ---------------------------------------------------------------------------

// hotspotHasManualChanges mirrors hasManualChanges for issues: returns
// true when the source hotspot carries metadata worth migrating to
// Cloud. Same criteria as the previous filterActionableHotspotPairs
// (#350) — REVIEWED with a real review resolution, or any comment —
// but applied source-side BEFORE we look at Cloud (#356).
func hotspotHasManualChanges(h matchableHotspot) bool {
	status := strings.ToUpper(h.Status)
	resolution := strings.ToUpper(h.Resolution)
	reviewed := status == "REVIEWED" && (resolution == "SAFE" || resolution == "ACKNOWLEDGED" || resolution == "FIXED")
	return reviewed || len(h.Comments) > 0
}

// ---------------------------------------------------------------------------
// Resolution mapping
// ---------------------------------------------------------------------------

// mapHotspotResolution converts a SonarQube Server hotspot resolution into
// the equivalent SonarQube Cloud resolution value.
// ACKNOWLEDGED does not exist in SonarCloud; the closest equivalent is SAFE.
func mapHotspotResolution(resolution string) string {
	switch strings.ToUpper(resolution) {
	case "SAFE":
		return "SAFE"
	case "FIXED":
		return "FIXED"
	case "ACKNOWLEDGED":
		return "SAFE"
	default:
		return "SAFE"
	}
}

// ---------------------------------------------------------------------------
// Main task entry point
// ---------------------------------------------------------------------------

// runSyncHotspotMetadata is the Run function for the syncHotspotMetadata task.
// It iterates over every project created during migration and synchronises
// hotspot statuses and comments from the SonarQube Server extract to Cloud.
func runSyncHotspotMetadata(ctx context.Context, e *Executor) error {
	return forEachMigrateItem(ctx, e, "syncHotspotMetadata", "createProjects",
		func(ctx context.Context, item json.RawMessage, w *common.ChunkWriter) error {
			cloudKey := extractField(item, "cloud_project_key")
			orgKey := extractField(item, "sonarcloud_org_key")
			serverURL := extractField(item, "server_url")
			serverKey := extractField(item, "key")

			if cloudKey == "" || orgKey == "" {
				return nil
			}

			result := syncProjectHotspots(ctx, e, syncHotspotInput{
				CloudKey:  cloudKey,
				OrgKey:    orgKey,
				ServerURL: serverURL,
				ServerKey: serverKey,
			})

			record, _ := json.Marshal(map[string]any{
				"cloud_project_key": cloudKey,
				"synced":            result.Stats.A,
				"line_mismatch":     result.Stats.B,
				"not_found":         result.Stats.C,
				"actionable":        result.Stats.Actionable,
				"error":             result.Error,
			})
			return w.WriteOne(record)
		})
}

// ---------------------------------------------------------------------------
// Per-project sync
// ---------------------------------------------------------------------------

type syncHotspotInput struct {
	CloudKey  string
	OrgKey    string
	ServerURL string
	ServerKey string
}

// syncHotspotResult holds the per-project sync outcome. Stats carries
// the a/b/c breakdown (#356); Error captures a fatal lookup failure
// that prevented the project from being processed at all.
type syncHotspotResult struct {
	Stats projectSyncStats
	Error string
}

// syncProjectHotspots synchronises hotspot metadata for a single
// project using the targeted per-actionable-source-hotspot search
// approach introduced in #356. Replaces the previous fetch-all + FIFO
// match scheme.
func syncProjectHotspots(ctx context.Context, e *Executor, input syncHotspotInput) syncHotspotResult {
	projStart := time.Now()
	counter := NewTaskCounter("syncHotspotMetadata:" + input.CloudKey)
	defer func() { counter.LogSummary(e.Logger, time.Since(projStart)) }()

	var result syncHotspotResult

	// 1. Load + pre-filter source hotspots to the actionable set.
	sourceHotspots, err := loadMatchableHotspots(e, input.ServerURL, input.ServerKey)
	if err != nil {
		result.Error = err.Error()
		logAPIWarn(e.Logger, "syncHotspotMetadata: load source hotspots failed", err, "project", input.CloudKey)
		return result
	}
	if len(sourceHotspots) == 0 {
		return result
	}
	var actionable []matchableHotspot
	for _, h := range sourceHotspots {
		if hotspotHasManualChanges(h) {
			actionable = append(actionable, h)
		}
	}
	result.Stats.Actionable = int64(len(actionable))
	if len(actionable) == 0 {
		e.Logger.Debug("syncHotspotMetadata: no actionable source hotspots after filter", "project", input.CloudKey, "source_total", len(sourceHotspots))
		return result
	}

	// 2. Wait for Cloud indexing — proves the CE task is done.
	_ = waitForCloudIndexing(ctx, func() (int, error) {
		return e.Cloud.Hotspots.Count(ctx, input.CloudKey, input.OrgKey)
	})

	e.Logger.Info("syncHotspotMetadata: syncing pairs",
		"project", input.CloudKey,
		"source_total", len(sourceHotspots),
		"actionable", len(actionable),
	)

	// 3 + 4. Per-actionable-source: targeted search + resolve by
	// (ruleKey, line). Race-safety: actionable is read-only, each
	// goroutine takes one hotspot by value, stats counters are
	// atomic.
	var a, b, c atomic.Int64
	label := "Project key " + input.CloudKey + " hotspot sync:"
	runProjectSyncLoop(ctx, e, actionable, label, 10,
		func(gctx context.Context, src matchableHotspot) {
			outcome := resolveAndSyncHotspot(gctx, e, input.CloudKey, input.OrgKey, input.ServerKey, src, counter)
			switch outcome {
			case syncOutcomeSynced:
				a.Add(1)
			case syncOutcomeLineMismatch:
				b.Add(1)
			case syncOutcomeNotFound:
				c.Add(1)
			}
		})
	result.Stats.A = a.Load()
	result.Stats.B = b.Load()
	result.Stats.C = c.Load()
	return result
}

// resolveAndSyncHotspot searches Cloud for hotspots in the source
// hotspot's file, then resolves by (ruleKey, line). Returns the case
// a/b/c/lookup outcome.
func resolveAndSyncHotspot(ctx context.Context, e *Executor, cloudKey, orgKey, sourceKey string, src matchableHotspot, counter *TaskCounter) syncOutcome {
	filePath := src.Component
	prefix := sourceKey + ":"
	if strings.HasPrefix(filePath, prefix) {
		filePath = filePath[len(prefix):]
	}
	if filePath == "" || src.RuleKey == "" || src.Line <= 0 {
		e.Logger.Debug("syncHotspotMetadata: source hotspot not matchable", "key", src.Key, "rule", src.RuleKey, "component", src.Component, "line", src.Line)
		return syncOutcomeNotFound
	}
	candidates, err := findCloudHotspotCandidates(ctx, e, cloudKey, orgKey, filePath)
	if err != nil {
		logAPIWarn(e.Logger, "syncHotspotMetadata: cloud candidate lookup failed", err,
			"project", cloudKey, "source_key", src.Key, "file", filePath)
		return syncOutcomeLookupError
	}
	target, outcome := classifyHotspotCandidatesByLine(candidates, src.RuleKey, src.Line)
	switch outcome {
	case syncOutcomeSynced:
		pair := hotspotPair{source: src, cloud: target}
		if err := syncOneHotspot(ctx, e, pair); err != nil {
			counter.Fail()
			logAPIWarn(e.Logger, "syncHotspotMetadata: hotspot sync failed", err,
				"source_key", src.Key, "cloud_key", target.Key)
		} else {
			counter.Success()
		}
	case syncOutcomeNotFound:
		e.Logger.Debug("syncHotspotMetadata: no cloud counterpart on source line", "source_key", src.Key, "rule", src.RuleKey, "file", filePath, "line", src.Line)
	case syncOutcomeLineMismatch:
		keys := make([]string, 0)
		for _, c := range candidates {
			if c.RuleKey == src.RuleKey && c.Line == src.Line {
				keys = append(keys, c.Key)
			}
		}
		e.Logger.Debug("syncHotspotMetadata: multiple cloud counterparts on source line, skipping", "source_key", src.Key, "rule", src.RuleKey, "file", filePath, "line", src.Line, "candidates", keys)
	}
	return outcome
}

// classifyHotspotCandidatesByLine is the hotspot counterpart of
// classifyIssueCandidatesByLine. The hotspot search isn't scoped by
// rule on the cloud side, so we filter on (ruleKey, line) here.
func classifyHotspotCandidatesByLine(candidates []matchableHotspot, sourceRule string, sourceLine int) (matchableHotspot, syncOutcome) {
	var pick matchableHotspot
	matches := 0
	for _, c := range candidates {
		if c.RuleKey == sourceRule && c.Line == sourceLine {
			matches++
			if matches == 1 {
				pick = c
			}
		}
	}
	switch matches {
	case 0:
		return matchableHotspot{}, syncOutcomeNotFound
	case 1:
		return pick, syncOutcomeSynced
	default:
		return matchableHotspot{}, syncOutcomeLineMismatch
	}
}

// findCloudHotspotCandidates queries /api/hotspots/search?files=…
// for cloud hotspots in a single file. Hotspots are scarce enough
// that we can skip the rule filter here and resolve in memory; the
// /api/hotspots/search endpoint also does not accept a rules
// parameter.
//
// Uses e.Raw (the cloud-side raw client) because the typed
// HotspotsClient's SearchAll doesn't expose a per-file filter.
func findCloudHotspotCandidates(ctx context.Context, e *Executor, cloudKey, orgKey, filePath string) ([]matchableHotspot, error) {
	params := url.Values{}
	params.Set("projectKey", cloudKey)
	params.Set("files", filePath)
	if orgKey != "" {
		params.Set("organization", orgKey)
	}
	items, err := e.Raw.GetPaginated(ctx, common.PaginatedOpts{
		Path:      "api/hotspots/search",
		Params:    params,
		ResultKey: "hotspots",
		PageLimit: 5, // hotspots-in-one-file is small; cap for safety
	})
	if err != nil {
		return nil, err
	}
	out := make([]matchableHotspot, 0, len(items))
	for _, raw := range items {
		h := parseMatchableHotspot(raw)
		if h.Key == "" {
			continue
		}
		out = append(out, h)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Per-hotspot sync
// ---------------------------------------------------------------------------

// syncOneHotspot synchronises a single hotspot's status and comments.
// Operations are sequential within each hotspot: status first, then comments.
func syncOneHotspot(ctx context.Context, e *Executor, pair hotspotPair) error {
	// 1. Sync status: if source is REVIEWED, change Cloud hotspot status.
	if strings.ToUpper(pair.source.Status) == "REVIEWED" {
		resolution := mapHotspotResolution(pair.source.Resolution)
		if err := e.Cloud.Hotspots.ChangeStatus(ctx, pair.cloud.Key, "REVIEWED", resolution); err != nil {
			return fmt.Errorf("change status: %w", err)
		}
	}

	// 2. Sync comments: fetch Cloud detail first for idempotency check.
	if len(pair.source.Comments) == 0 {
		return nil
	}

	cloudComments := fetchCloudHotspotComments(ctx, e, pair.cloud.Key)
	for _, comment := range pair.source.Comments {
		if isAlreadyMigratedComment(comment, cloudComments) {
			continue
		}

		text := formatMigratedHotspotComment(comment)
		if text == "" {
			continue
		}

		if err := e.Cloud.Hotspots.AddComment(ctx, pair.cloud.Key, text); err != nil {
			logAPIWarn(e.Logger, "syncHotspotMetadata: add comment failed", err,
				"cloud_key", pair.cloud.Key,
				"comment_author", comment.Login,
			)
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}

	return nil
}

// fetchCloudHotspotComments retrieves comments for a Cloud hotspot via the
// /api/hotspots/show endpoint. Returns nil on failure (non-fatal — worst case
// is duplicate comments, not data loss).
func fetchCloudHotspotComments(ctx context.Context, e *Executor, cloudKey string) []hotspotComment {
	detail, err := e.Cloud.Hotspots.Show(ctx, cloudKey)
	if err != nil {
		e.Logger.Debug("syncHotspotMetadata: could not fetch cloud hotspot detail",
			"cloud_key", cloudKey, "err", err)
		return nil
	}
	comments := make([]hotspotComment, 0, len(detail.Comment))
	for _, c := range detail.Comment {
		comments = append(comments, hotspotComment{
			Login:    c.Login,
			HTMLText: c.HTMLText,
			Markdown: c.Markdown,
		})
	}
	return comments
}

// migratedCommentPrefix is prepended to every migrated comment.
const migratedCommentPrefix = "[Migrated from SonarQube]"

// formatMigratedHotspotComment builds the comment text to post to Cloud.
func formatMigratedHotspotComment(c hotspotComment) string {
	body := c.Markdown
	if body == "" {
		body = c.HTMLText
	}
	if body == "" {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(migratedCommentPrefix)
	sb.WriteString("\n\n")
	if c.Login != "" {
		sb.WriteString("**Author:** ")
		sb.WriteString(c.Login)
		sb.WriteString("\n")
	}
	if c.CreatedAt != "" {
		sb.WriteString("**Date:** ")
		sb.WriteString(c.CreatedAt)
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(body)
	return sb.String()
}

// isAlreadyMigratedComment checks whether a source comment has already been
// migrated to Cloud by looking for the migration prefix in Cloud comments.
func isAlreadyMigratedComment(source hotspotComment, cloudComments []hotspotComment) bool {
	body := source.Markdown
	if body == "" {
		body = source.HTMLText
	}
	if body == "" {
		return true // empty comment, treat as already handled
	}

	for _, cc := range cloudComments {
		ccText := cc.Markdown
		if ccText == "" {
			ccText = cc.HTMLText
		}
		if strings.Contains(ccText, migratedCommentPrefix) && strings.Contains(ccText, body) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Loading source hotspots from extract data
// ---------------------------------------------------------------------------

// loadMatchableHotspots reads the getProjectHotspotsFull extract items and
// converts them into normalised matchableHotspot structs.
func loadMatchableHotspots(e *Executor, serverURL, serverKey string) ([]matchableHotspot, error) {
	items, err := readExtractItems(e, "getProjectHotspotsFull")
	if err != nil {
		return nil, fmt.Errorf("loadMatchableHotspots: %w", err)
	}

	var hotspots []matchableHotspot
	for _, item := range items {
		if item.ServerURL != serverURL {
			continue
		}

		// Filter by project key.
		projKey := extractField(item.Data, "project")
		if projKey == "" {
			projKey = extractField(item.Data, "projectKey")
		}
		if projKey != serverKey {
			continue
		}

		h := parseMatchableHotspot(item.Data)
		if h.Key == "" {
			continue
		}
		hotspots = append(hotspots, h)
	}
	return hotspots, nil
}

// parseMatchableHotspot extracts a matchableHotspot from a raw JSON object.
func parseMatchableHotspot(data json.RawMessage) matchableHotspot {
	key := extractField(data, "key")
	component := extractField(data, "component")
	status := extractField(data, "status")
	resolution := extractField(data, "resolution")
	line := extractHotspotLine(data)

	// ruleKey may be at top level or nested inside a "rule" object.
	ruleKey := extractField(data, "ruleKey")
	if ruleKey == "" {
		ruleKey = extractNestedField(data, "rule", "key")
	}

	comments := parseHotspotComments(data)

	return matchableHotspot{
		Key:        key,
		RuleKey:    ruleKey,
		Component:  component,
		Line:       line,
		Status:     status,
		Resolution: resolution,
		Comments:   comments,
	}
}

// extractHotspotLine reads the "line" field from a hotspot JSON object.
func extractHotspotLine(data json.RawMessage) int {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return 0
	}
	raw, ok := obj["line"]
	if !ok {
		return 0
	}
	var v int
	if json.Unmarshal(raw, &v) == nil {
		return v
	}
	// Try as string.
	var s string
	if json.Unmarshal(raw, &s) == nil {
		n, _ := strconv.Atoi(s)
		return n
	}
	return 0
}

// extractNestedField reads obj[outerKey][innerKey] as a string.
func extractNestedField(data json.RawMessage, outerKey, innerKey string) string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return ""
	}
	nested, ok := obj[outerKey]
	if !ok {
		return ""
	}
	return extractField(nested, innerKey)
}

// ---------------------------------------------------------------------------
// Parsing comments from extract data
// ---------------------------------------------------------------------------

// parseHotspotComments extracts the comment array from a hotspot detail JSON.
// The SonarQube API uses "comment" (singular) as the field name for the array.
func parseHotspotComments(data json.RawMessage) []hotspotComment {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}

	// The field name is "comment" (singular) in the hotspot detail response.
	raw, ok := obj["comment"]
	if !ok {
		// Also try "comments" (plural) in case extract data uses a different format.
		raw, ok = obj["comments"]
		if !ok {
			return nil
		}
	}

	var rawComments []json.RawMessage
	if err := json.Unmarshal(raw, &rawComments); err != nil {
		return nil
	}

	comments := make([]hotspotComment, 0, len(rawComments))
	for _, rc := range rawComments {
		c := hotspotComment{
			Login:     extractField(rc, "login"),
			HTMLText:  extractField(rc, "htmlText"),
			Markdown:  extractField(rc, "markdown"),
			CreatedAt: extractField(rc, "createdAt"),
		}
		if c.HTMLText != "" || c.Markdown != "" {
			comments = append(comments, c)
		}
	}
	return comments
}

// ---------------------------------------------------------------------------
// Compile-time interface assertion to ensure ExtractItem is used correctly.
// ---------------------------------------------------------------------------

var _ = (structure.ExtractItem{}).ServerURL
