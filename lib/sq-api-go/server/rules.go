package server

import (
	"context"
	"net/url"

	sqapi "github.com/sonar-solutions/sq-api-go"
	"github.com/sonar-solutions/sq-api-go/types"
)

// RulesClient provides methods for the /api/rules endpoints.
type RulesClient struct{ baseClient }

// Search returns a Paginator over all rules from /api/rules/search.
func (r *RulesClient) Search(ctx context.Context) *sqapi.Paginator[types.Rule] {
	return sqapi.NewPaginator(func(ctx context.Context, page, pageSize int) ([]types.Rule, int, error) {
		params := url.Values{}
		params.Set("p", itoa(page))
		params.Set("ps", itoa(pageSize))

		var result types.RulesSearchResponse
		if err := r.get(ctx, "api/rules/search", params, &result); err != nil {
			return nil, 0, err
		}
		return result.Rules, result.Paging.Total, nil
	}, 0)
}

// Show fetches a single rule by key from /api/rules/show.
func (r *RulesClient) Show(ctx context.Context, ruleKey string) (*types.Rule, error) {
	params := url.Values{}
	params.Set("key", ruleKey)

	var result types.RuleShowResponse
	if err := r.get(ctx, "api/rules/show", params, &result); err != nil {
		return nil, err
	}
	return &result.Rule, nil
}

// Repositories returns all rule repositories from /api/rules/repositories.
func (r *RulesClient) Repositories(ctx context.Context) ([]types.RuleRepository, error) {
	var result types.RepositoriesResponse
	if err := r.get(ctx, "api/rules/repositories", nil, &result); err != nil {
		return nil, err
	}
	return result.Repositories, nil
}
