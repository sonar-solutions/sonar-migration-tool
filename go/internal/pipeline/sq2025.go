package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// SQ2025Pipeline handles SonarQube Server 2025.1 and later.
//   - Uses modern "issueStatuses" parameter
//   - No metric-key batching (all keys sent in a single request)
//   - Uses /api/v2/authorizations/groups with fallback to standard API
//   - IN_SANDBOX issues are detected in results, logged as warnings, and skipped
type SQ2025Pipeline struct {
	client *sqapi.Client
}

func newSQ2025(client *sqapi.Client) *SQ2025Pipeline { return &SQ2025Pipeline{client: client} }

var _ Pipeline = (*SQ2025Pipeline)(nil)

func (p *SQ2025Pipeline) Version() string { return "sq-2025" }

func (p *SQ2025Pipeline) IssueSearchParam() string { return "issueStatuses" }

// IssueStatusValues returns the queryable issue statuses. IN_SANDBOX is NOT
// included because it may not be a valid search value; issues with that status
// are detected in results and handled by ExtractIssues.
func (p *SQ2025Pipeline) IssueStatusValues() []string {
	return []string{"OPEN", "CONFIRMED", "FALSE_POSITIVE", "ACCEPTED", "FIXED"}
}

// SupportsMetricBatching reports that SQ 2025.1+ requires no metric batching:
// all keys are sent in a single request.
func (p *SQ2025Pipeline) SupportsMetricBatching() (bool, int) { return false, 0 }

func (p *SQ2025Pipeline) ExtractIssues(ctx context.Context, projectKey string) ([]Issue, error) {
	issues, err := fetchAllIssues(ctx, p.client, projectKey, p.IssueSearchParam(), p.IssueStatusValues())
	if err != nil {
		return nil, err
	}
	// IN_SANDBOX has no SonarQube Cloud equivalent: log and skip.
	filtered := issues[:0]
	for _, iss := range issues {
		if iss.Status == "IN_SANDBOX" {
			slog.Warn("skipping IN_SANDBOX issue (no SonarQube Cloud equivalent)",
				"key", iss.Key, "component", iss.Component)
			continue
		}
		filtered = append(filtered, iss)
	}
	return filtered, nil
}

func (p *SQ2025Pipeline) ExtractHotspots(ctx context.Context, projectKey string) ([]Hotspot, error) {
	return fetchAllHotspots(ctx, p.client, projectKey)
}

// ExtractMetrics sends all metricKeys in a single request (no batching for
// SQ 2025.1+).
func (p *SQ2025Pipeline) ExtractMetrics(ctx context.Context, projectKey string, metricKeys []string) ([]ComponentMetrics, error) {
	return fetchAllMetrics(ctx, p.client, projectKey, metricKeys, 0)
}

// ExtractGroups attempts the V2 authorizations groups API first, falling back
// to the standard /api/user_groups/search on 404 or any error.
func (p *SQ2025Pipeline) ExtractGroups(ctx context.Context) ([]Group, error) {
	groups, err := p.fetchGroupsV2(ctx)
	if err != nil {
		slog.Warn("V2 groups API unavailable, falling back to standard API", "err", err)
		return fetchAllGroups(ctx, p.client)
	}
	return groups, nil
}

func (p *SQ2025Pipeline) fetchGroupsV2(ctx context.Context) ([]Group, error) {
	var all []Group
	for page := 1; ; page++ {
		items, total, err := p.fetchGroupsV2Page(ctx, page, 500)
		if err != nil {
			return nil, err
		}
		all = append(all, items...)
		if len(items) == 0 || len(all) >= total {
			break
		}
	}
	return all, nil
}

func (p *SQ2025Pipeline) fetchGroupsV2Page(ctx context.Context, page, pageSize int) ([]Group, int, error) {
	params := url.Values{
		"pageIndex": {strconv.Itoa(page)},
		"pageSize":  {strconv.Itoa(pageSize)},
	}
	u := p.client.BaseURL() + "/api/v2/authorizations/groups?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("building V2 groups request: %w", err)
	}
	resp, err := p.client.HTTPClient().Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("fetching V2 groups: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, 0, fmt.Errorf("V2 groups API not available (404)")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("V2 groups API returned HTTP %d", resp.StatusCode)
	}
	var result struct {
		Groups []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"groups"`
		Page struct {
			PageIndex int `json:"pageIndex"`
			PageSize  int `json:"pageSize"`
			Total     int `json:"total"`
		} `json:"page"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decoding V2 groups response: %w", err)
	}
	groups := make([]Group, 0, len(result.Groups))
	for _, g := range result.Groups {
		groups = append(groups, Group{Name: g.Name, Description: g.Description})
	}
	return groups, result.Page.Total, nil
}

func (p *SQ2025Pipeline) EnrichCleanCode(_ context.Context, issues []Issue, _ *sqapi.Client) ([]Issue, error) {
	return issues, nil
}
