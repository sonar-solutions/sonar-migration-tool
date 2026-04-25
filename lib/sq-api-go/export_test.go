// This file is in package sqapi (not sqapi_test) so it can access unexported
// identifiers. It is compiled only during 'go test' because the filename ends
// in _test.go. These exports are not part of the public API.
package sqapi

import (
	"crypto/tls"
	"net/http"
	"time"
)

var (
	BuildAuthHeader = buildAuthHeader
	TotalPages      = totalPages
	Jitter          = jitter
)

// BuildTransport exposes buildTransport for testing TLS config branches.
func BuildTransport(token string, version float64, tlsMinVersion uint16) http.RoundTripper {
	cfg := defaultClientConfig()
	if tlsMinVersion > 0 {
		cfg.tlsConfig = &tls.Config{MinVersion: tlsMinVersion} //nolint:gosec
	} else {
		cfg.tlsConfig = &tls.Config{} //nolint:gosec
	}
	return buildTransport(cfg, token, version)
}

// NewAuthTransport wraps inner with the authorization header injection transport.
func NewAuthTransport(inner http.RoundTripper, header string) http.RoundTripper {
	return &authTransport{inner: inner, header: header}
}

// NewRetryTransport wraps inner with the retry transport using the given backoff schedule.
func NewRetryTransport(inner http.RoundTripper, backoff []time.Duration) http.RoundTripper {
	return &retryTransport{inner: inner, backoff: backoff}
}
