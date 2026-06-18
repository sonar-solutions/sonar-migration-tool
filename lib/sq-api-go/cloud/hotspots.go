// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package cloud

import (
	"context"
	"fmt"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// HotspotsClient wraps the /api/hotspots/* endpoints.
type HotspotsClient struct{ baseClient }

// SearchAll paginates /api/hotspots/search and returns all hotspots for a project.
func (c *HotspotsClient) SearchAll(ctx context.Context, projectKey, org string) ([]types.Hotspot, error) {
	params := url.Values{
		"projectKey": {projectKey},
		"ps":         {"500"},
	}
	if org != "" {
		params.Set("organization", org)
	}
	var all []types.Hotspot
	for page := 1; ; page++ {
		params.Set("p", fmt.Sprintf("%d", page))
		var resp types.HotspotsSearchResponse
		if err := c.getJSON(ctx, "api/hotspots/search?"+params.Encode(), &resp); err != nil {
			return all, err
		}
		all = append(all, resp.Hotspots...)
		if len(all) >= resp.Paging.Total || len(resp.Hotspots) == 0 {
			break
		}
	}
	return all, nil
}

// Count returns the total number of hotspots matching the given project
// without fetching all of them. Makes a single API call with ps=1.
func (c *HotspotsClient) Count(ctx context.Context, projectKey, org string) (int, error) {
	params := url.Values{
		"projectKey": {projectKey},
		"ps":         {"1"},
		"p":          {"1"},
	}
	if org != "" {
		params.Set("organization", org)
	}
	var resp types.HotspotsSearchResponse
	if err := c.getJSON(ctx, "api/hotspots/search?"+params.Encode(), &resp); err != nil {
		return 0, err
	}
	return resp.Paging.Total, nil
}

// Show fetches full detail for a single hotspot including comments.
func (c *HotspotsClient) Show(ctx context.Context, hotspotKey string) (*types.HotspotDetail, error) {
	var resp types.HotspotDetail
	if err := c.getJSON(ctx, "api/hotspots/show?hotspot="+url.QueryEscape(hotspotKey), &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ChangeStatus changes the status and resolution of a hotspot.
func (c *HotspotsClient) ChangeStatus(ctx context.Context, hotspotKey, status, resolution string) error {
	form := url.Values{}
	form.Set("hotspot", hotspotKey)
	form.Set("status", status)
	if resolution != "" {
		form.Set("resolution", resolution)
	}
	return c.postForm(ctx, "api/hotspots/change_status", form, nil)
}

// AddComment adds a comment to a hotspot. Note the text parameter is named
// "comment" for api/hotspots/add_comment (unlike api/issues/add_comment,
// which uses "text") — sending "text" yields a 400 "The 'comment' parameter
// is missing".
func (c *HotspotsClient) AddComment(ctx context.Context, hotspotKey, text string) error {
	form := url.Values{}
	form.Set("hotspot", hotspotKey)
	form.Set("comment", text)
	return c.postForm(ctx, "api/hotspots/add_comment", form, nil)
}
