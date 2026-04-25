package sqapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

const testBaseURL = "https://sonar.example.com"

// ---- HTTPClient ----

func TestHTTPClientNotNil(t *testing.T) {
	c := sqapi.NewServerClient(testBaseURL, "token", 10.7)
	assert.NotNil(t, c.HTTPClient())
}

func TestHTTPClientMakesRequest(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := sqapi.NewServerClient(ts.URL, "my-token", 10.7)
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
	require.NoError(t, err)
	resp, err := c.HTTPClient().Do(req)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "Bearer my-token", gotAuth)
}

// ---- APIError ----

func TestAPIErrorErrorString(t *testing.T) {
	err := &sqapi.APIError{
		StatusCode: 403,
		Method:     http.MethodGet,
		URL:        "https://sonar.example.com/api/projects/search",
		Body:       `{"errors":[{"msg":"Insufficient privileges"}]}`,
	}
	msg := err.Error()
	assert.Contains(t, msg, "403")
	assert.Contains(t, msg, "GET")
	assert.Contains(t, msg, "api/projects/search")
}

// ---- Error predicate functions ----

func TestIsNotFoundTrue(t *testing.T) {
	err := &sqapi.APIError{StatusCode: http.StatusNotFound}
	assert.True(t, sqapi.IsNotFound(err))
}

func TestIsNotFoundFalse(t *testing.T) {
	assert.False(t, sqapi.IsNotFound(errors.New("other error")))
	assert.False(t, sqapi.IsNotFound(&sqapi.APIError{StatusCode: 403}))
}

func TestIsUnauthorizedTrue(t *testing.T) {
	err := &sqapi.APIError{StatusCode: http.StatusUnauthorized}
	assert.True(t, sqapi.IsUnauthorized(err))
}

func TestIsUnauthorizedFalse(t *testing.T) {
	assert.False(t, sqapi.IsUnauthorized(errors.New("other")))
	assert.False(t, sqapi.IsUnauthorized(&sqapi.APIError{StatusCode: 403}))
}

func TestIsForbiddenTrue(t *testing.T) {
	err := &sqapi.APIError{StatusCode: http.StatusForbidden}
	assert.True(t, sqapi.IsForbidden(err))
}

func TestIsForbiddenFalse(t *testing.T) {
	assert.False(t, sqapi.IsForbidden(errors.New("other")))
	assert.False(t, sqapi.IsForbidden(&sqapi.APIError{StatusCode: 404}))
}

func TestIsAlreadyExistsTrue(t *testing.T) {
	err := &sqapi.APIError{StatusCode: 400, Body: `{"errors":[{"msg":"Group with name 'X' already exists"}]}`}
	assert.True(t, sqapi.IsAlreadyExists(err))
}

func TestIsAlreadyExistsCaseInsensitive(t *testing.T) {
	err := &sqapi.APIError{StatusCode: 400, Body: `{"errors":[{"msg":"Quality profile ALREADY EXISTS: java/MyProfile"}]}`}
	assert.True(t, sqapi.IsAlreadyExists(err))
}

func TestIsAlreadyExistsFalse(t *testing.T) {
	assert.False(t, sqapi.IsAlreadyExists(errors.New("other")))
	assert.False(t, sqapi.IsAlreadyExists(&sqapi.APIError{StatusCode: 400, Body: "invalid parameter"}))
	assert.False(t, sqapi.IsAlreadyExists(&sqapi.APIError{StatusCode: 404, Body: "already exists"}))
}

// ---- Options ----

func TestWithMaxConnections(t *testing.T) {
	c := sqapi.NewServerClient(testBaseURL, "token", 10.7, sqapi.WithMaxConnections(10))
	assert.NotNil(t, c)
}

func TestWithTimeout(t *testing.T) {
	c := sqapi.NewServerClient(testBaseURL, "token", 10.7, sqapi.WithTimeout(30))
	assert.NotNil(t, c)
}

func TestWithClientCertMissingCertFile(t *testing.T) {
	c := sqapi.NewServerClient(testBaseURL, "token", 10.7,
		sqapi.WithClientCert("/nonexistent/cert.pem", "/nonexistent/key.pem", ""),
	)
	require.Error(t, c.CertErr())
	assert.Contains(t, c.CertErr().Error(), "cert.pem")
}

func TestWithClientCertMissingKeyFile(t *testing.T) {
	certFile := t.TempDir() + "/cert.pem"
	require.NoError(t, os.WriteFile(certFile, []byte("invalid pem content"), 0600))

	c := sqapi.NewServerClient(testBaseURL, "token", 10.7,
		sqapi.WithClientCert(certFile, "/nonexistent/key.pem", ""),
	)
	require.Error(t, c.CertErr())
}

func TestWithClientCertInvalidKeyPair(t *testing.T) {
	dir := t.TempDir()
	certFile := dir + "/cert.pem"
	keyFile := dir + "/key.pem"
	require.NoError(t, os.WriteFile(certFile, []byte("not valid pem"), 0600))
	require.NoError(t, os.WriteFile(keyFile, []byte("not valid pem"), 0600))

	c := sqapi.NewServerClient(testBaseURL, "token", 10.7,
		sqapi.WithClientCert(certFile, keyFile, ""),
	)
	require.Error(t, c.CertErr())
	assert.Contains(t, c.CertErr().Error(), "loading client certificate")
}

// ---- authTransport ----

func TestAuthTransportInjectsHeader(t *testing.T) {
	var gotHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	transport := sqapi.NewAuthTransport(http.DefaultTransport, "Bearer test-token")
	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "Bearer test-token", gotHeader)
}

// ---- retryTransport ----

func TestRetryTransportSuccessFirstAttempt(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	transport := sqapi.NewRetryTransport(http.DefaultTransport, []time.Duration{0, 0})
	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRetryTransportRetriesOn503(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	transport := sqapi.NewRetryTransport(http.DefaultTransport, []time.Duration{0})
	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, attempts)
}

func TestRetryTransportRetriesOn429(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	transport := sqapi.NewRetryTransport(http.DefaultTransport, []time.Duration{0, 0})
	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestRetryTransportExhaustsRetries(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	transport := sqapi.NewRetryTransport(http.DefaultTransport, []time.Duration{0, 0})
	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestRetryTransportDoesNotRetryOn4xx(t *testing.T) {
	attempts := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	transport := sqapi.NewRetryTransport(http.DefaultTransport, []time.Duration{0, 0})
	client := &http.Client{Transport: transport}
	resp, err := client.Get(ts.URL)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 1, attempts, "4xx should not trigger a retry")
}

// ---- NewPaginator default page size ----

func TestNewPaginatorDefaultPageSize(t *testing.T) {
	var receivedPageSize int
	fetch := func(_ context.Context, _, pageSize int) ([]string, int, error) {
		receivedPageSize = pageSize
		return []string{"a"}, 1, nil
	}
	pag := sqapi.NewPaginator(fetch, 0) // 0 → should default to PageSize (500)
	_, err := pag.All(context.Background())
	require.NoError(t, err)
	assert.Equal(t, sqapi.PageSize, receivedPageSize)
}
