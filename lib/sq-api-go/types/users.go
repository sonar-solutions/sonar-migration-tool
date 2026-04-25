package types

// User represents a single user returned by /api/users/search.
type User struct {
	Login            string   `json:"login"`
	Name             string   `json:"name"`
	Email            string   `json:"email"`
	Active           bool     `json:"active"`
	Local            bool     `json:"local"`
	ExternalIdentity string   `json:"externalIdentity"`
	ExternalProvider string   `json:"externalProvider"`
	Groups           []string `json:"groups"`
}

// UsersSearchResponse is the response envelope for /api/users/search.
type UsersSearchResponse struct {
	PagedResponse
	Users []User `json:"users"`
}
