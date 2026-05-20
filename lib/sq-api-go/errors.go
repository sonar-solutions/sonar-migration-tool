package sqapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

// Message returns the human-readable error message(s) extracted from the
// SonarQube JSON error response body. Falls back to the raw body if parsing fails.
func (e *APIError) Message() string {
	if e.Body == "" {
		return ""
	}
	var obj struct {
		Errors []struct {
			Msg string `json:"msg"`
		} `json:"errors"`
	}
	if json.Unmarshal([]byte(e.Body), &obj) != nil || len(obj.Errors) == 0 {
		return e.Body
	}
	msgs := make([]string, 0, len(obj.Errors))
	for _, item := range obj.Errors {
		if item.Msg != "" {
			msgs = append(msgs, item.Msg)
		}
	}
	if len(msgs) == 0 {
		return e.Body
	}
	return strings.Join(msgs, "; ")
}

// Endpoint returns the API path from the full URL (strips scheme and host).
func (e *APIError) Endpoint() string {
	if u, err := url.Parse(e.URL); err == nil {
		return u.Path
	}
	return e.URL
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
