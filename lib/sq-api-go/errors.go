package sqapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError is returned when the SonarQube API responds with an HTTP error status.
type APIError struct {
	// StatusCode is the HTTP status code returned by the server.
	StatusCode int
	// Method is the HTTP method used in the request.
	Method string
	// URL is the request URL.
	URL string
	// Body is the raw response body, if available.
	Body string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("sonarqube api error: %s %s → %d: %s", e.Method, e.URL, e.StatusCode, e.Body)
}

// IsNotFound reports whether err is an APIError with status 404.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound
}

// IsUnauthorized reports whether err is an APIError with status 401.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusUnauthorized
}

// IsForbidden reports whether err is an APIError with status 403.
func IsForbidden(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden
}

// IsAlreadyExists reports whether err is an APIError with status 400
// whose body indicates the resource already exists.
func IsAlreadyExists(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		return false
	}
	lower := strings.ToLower(apiErr.Body)
	return strings.Contains(lower, "already exists")
}
