package sqapi

import (
	"encoding/base64"
	"fmt"
	"net/http"
)

const (
	// versionBearerThreshold is the first Server version that uses Bearer tokens.
	// Server versions below this use HTTP Basic auth.
	versionBearerThreshold = 10.0

	// cloudSentinel is the version value that indicates a SonarQube Cloud client.
	cloudSentinel = 0.0
)

// authTransport is an http.RoundTripper that injects the correct Authorization
// header for every request, based on the target SonarQube version.
// The header value is pre-computed once at construction time.
type authTransport struct {
	inner  http.RoundTripper
	header string
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request so we never mutate the caller's copy.
	r2 := req.Clone(req.Context())
	r2.Header.Set("Authorization", t.header)
	return t.inner.RoundTrip(r2)
}

// buildAuthHeader returns the Authorization header value for the given token
// and server version.
//
//   - version == cloudSentinel (0.0): Bearer token (Cloud)
//   - version >= versionBearerThreshold (10.0): Bearer token (Server ≥10)
//   - otherwise: Basic auth — base64(token + ":")
func buildAuthHeader(token string, version float64) string {
	if version == cloudSentinel || version >= versionBearerThreshold {
		return fmt.Sprintf("Bearer %s", token)
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(token + ":"))
	return fmt.Sprintf("Basic %s", encoded)
}
