package server

import (
	"context"
	"net/url"

	"github.com/sonar-solutions/sq-api-go/types"
)

// TokensClient provides methods for /api/user_tokens endpoints.
type TokensClient struct{ baseClient }

// Search returns all tokens for a user from /api/user_tokens/search.
func (t *TokensClient) Search(ctx context.Context, login string) ([]types.UserToken, error) {
	params := url.Values{}
	params.Set("login", login)

	var result types.UserTokensResponse
	if err := t.get(ctx, "api/user_tokens/search", params, &result); err != nil {
		return nil, err
	}
	return result.UserTokens, nil
}
