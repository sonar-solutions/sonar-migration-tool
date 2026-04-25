// Package server provides typed SonarQube Server API clients for the endpoints
// used by the sonar-migration-tool.
//
// Usage:
//
//	base := sqapi.NewServerClient("https://sonar.example.com", "squ_token", 10.7)
//	sc := server.New(base)
//
//	projects, err := sc.Projects.Search(ctx).All(ctx)
//	info, err := sc.System.Info(ctx)
package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

// Client is the entry point for all SonarQube Server API calls.
// Each field is a sub-client for a specific API domain.
type Client struct {
	Projects        *ProjectsClient
	Users           *UsersClient
	Groups          *GroupsClient
	Rules           *RulesClient
	System          *SystemClient
	QualityGates    *QualityGatesClient
	QualityProfiles *QualityProfilesClient
	Permissions     *PermissionsClient
	Branches        *BranchesClient
	PullRequests    *PullRequestsClient
	Analyses        *AnalysesClient
	Issues          *IssuesClient
	Hotspots        *HotspotsClient
	Measures        *MeasuresClient
	Settings        *SettingsClient
	Plugins         *PluginsClient
	Views           *ViewsClient
	Webhooks        *WebhooksClient
	Tokens          *TokensClient
	NewCode         *NewCodeClient
	ALM             *AlmClient
}

// New wraps a base sqapi.Client with typed Server endpoint methods.
func New(c *sqapi.Client) *Client {
	b := baseClient{c: c}
	return &Client{
		Projects:        &ProjectsClient{b},
		Users:           &UsersClient{b},
		Groups:          &GroupsClient{b},
		Rules:           &RulesClient{b},
		System:          &SystemClient{b},
		QualityGates:    &QualityGatesClient{b},
		QualityProfiles: &QualityProfilesClient{b},
		Permissions:     &PermissionsClient{b},
		Branches:        &BranchesClient{b},
		PullRequests:    &PullRequestsClient{b},
		Analyses:        &AnalysesClient{b},
		Issues:          &IssuesClient{b},
		Hotspots:        &HotspotsClient{b},
		Measures:        &MeasuresClient{b},
		Settings:        &SettingsClient{b},
		Plugins:         &PluginsClient{b},
		Views:           &ViewsClient{b},
		Webhooks:        &WebhooksClient{b},
		Tokens:          &TokensClient{b},
		NewCode:         &NewCodeClient{b},
		ALM:             &AlmClient{b},
	}
}

// itoa converts an int to its decimal string representation.
// Used to build pagination query parameters.
func itoa(n int) string { return strconv.Itoa(n) }

// baseClient is embedded by all sub-clients to share HTTP + JSON logic.
// path arguments must not include a leading slash — baseURL already ends with '/'.
type baseClient struct{ c *sqapi.Client }

// get executes a GET request, decodes the JSON response into out, and returns
// an sqapi.APIError for any non-2xx HTTP status.
func (b *baseClient) get(ctx context.Context, path string, params url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.c.BaseURL()+path, nil)
	if err != nil {
		return err
	}
	if len(params) > 0 {
		req.URL.RawQuery = params.Encode()
	}

	resp, err := b.c.HTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &sqapi.APIError{
			StatusCode: resp.StatusCode,
			Method:     http.MethodGet,
			URL:        req.URL.String(),
			Body:       string(body),
		}
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

// getBytes executes a GET request and returns the raw response body.
// Used for endpoints that return non-JSON content (e.g. profile XML backups).
func (b *baseClient) getBytes(ctx context.Context, path string, params url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.c.BaseURL()+path, nil)
	if err != nil {
		return nil, err
	}
	if len(params) > 0 {
		req.URL.RawQuery = params.Encode()
	}

	resp, err := b.c.HTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, &sqapi.APIError{
			StatusCode: resp.StatusCode,
			Method:     http.MethodGet,
			URL:        req.URL.String(),
			Body:       string(body),
		}
	}

	return body, nil
}
