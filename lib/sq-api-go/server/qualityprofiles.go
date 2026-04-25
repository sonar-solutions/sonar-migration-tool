package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// QualityProfilesClient provides methods for the /api/qualityprofiles endpoints.
type QualityProfilesClient struct{ baseClient }

// Search returns all quality profiles from /api/qualityprofiles/search.
func (q *QualityProfilesClient) Search(ctx context.Context) ([]types.QualityProfile, error) {
	var result types.QualityProfilesSearchResponse
	if err := q.get(ctx, "api/qualityprofiles/search", nil, &result); err != nil {
		return nil, err
	}
	return result.Profiles, nil
}

// Backup returns the raw XML backup of a quality profile from
// /api/qualityprofiles/backup.
func (q *QualityProfilesClient) Backup(ctx context.Context, language, profileName string) ([]byte, error) {
	params := url.Values{}
	params.Set("language", language)
	params.Set("qualityProfile", profileName)
	return q.getBytes(ctx, "api/qualityprofiles/backup", params)
}

// SearchGroups returns a Paginator over groups associated with a quality profile
// from /api/qualityprofiles/search_groups.
func (q *QualityProfilesClient) SearchGroups(ctx context.Context, language, profileName string) *sqapi.Paginator[types.ProfileGroup] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.ProfileGroup, int, error) {
		params := url.Values{}
		params.Set("language", language)
		params.Set("qualityProfile", profileName)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.ProfileGroupsResponse
		if err := q.get(ctx, "api/qualityprofiles/search_groups", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Groups, result.Paging.Total, nil
	}, 0)
}

// SearchUsers returns a Paginator over users associated with a quality profile
// from /api/qualityprofiles/search_users.
func (q *QualityProfilesClient) SearchUsers(ctx context.Context, language, profileName string) *sqapi.Paginator[types.ProfileUser] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.ProfileUser, int, error) {
		params := url.Values{}
		params.Set("language", language)
		params.Set("qualityProfile", profileName)
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.ProfileUsersResponse
		if err := q.get(ctx, "api/qualityprofiles/search_users", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Users, result.Paging.Total, nil
	}, 0)
}
