package types

// UserToken represents a single API token returned by /api/user_tokens/search.
type UserToken struct {
	Name               string `json:"name"`
	Type               string `json:"type"`
	CreatedAt          string `json:"createdAt"`
	LastConnectionDate string `json:"lastConnectionDate"`
	ExpirationDate     string `json:"expirationDate"`
	IsExpired          bool   `json:"isExpired"`
}

// UserTokensResponse is the response envelope for /api/user_tokens/search.
type UserTokensResponse struct {
	Login      string      `json:"login"`
	UserTokens []UserToken `json:"userTokens"`
}
