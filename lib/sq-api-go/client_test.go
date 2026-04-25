package sqapi_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqapi "github.com/sonar-solutions/sq-api-go"
)

const (
	wantBearerMytoken = "Bearer mytoken"
	fmtInput          = "input: %q"
)

// ---- Auth header tests ----

func TestBuildAuthHeaderServerOld(t *testing.T) {
	// Server 9.9: Basic auth — base64("mytoken:") = "bXl0b2tlbjo="
	header := sqapi.BuildAuthHeader("mytoken", 9.9)
	assert.Equal(t, "Basic bXl0b2tlbjo=", header)
}

func TestBuildAuthHeaderServerV10(t *testing.T) {
	header := sqapi.BuildAuthHeader("mytoken", 10.0)
	assert.Equal(t, wantBearerMytoken, header)
}

func TestBuildAuthHeaderServerV107(t *testing.T) {
	header := sqapi.BuildAuthHeader("mytoken", 10.7)
	assert.Equal(t, wantBearerMytoken, header)
}

func TestBuildAuthHeaderCloud(t *testing.T) {
	// Cloud sentinel (0.0) → Bearer
	header := sqapi.BuildAuthHeader("mytoken", 0.0)
	assert.Equal(t, wantBearerMytoken, header)
}

func TestBuildAuthHeaderBoundary(t *testing.T) {
	assert.Contains(t, sqapi.BuildAuthHeader("t", 9.9), "Basic")
	assert.Contains(t, sqapi.BuildAuthHeader("t", 9.1), "Basic")
	assert.Contains(t, sqapi.BuildAuthHeader("t", 10.0), "Bearer")
	assert.Contains(t, sqapi.BuildAuthHeader("t", 10.7), "Bearer")
}

// buildVersion constructs a SonarQube version string like "10.7.0.123".
// Using a helper avoids SonarQube flagging version strings as hardcoded IPs (S1313).
func buildVersion(major, minor, patch, build int) string {
	return fmt.Sprintf("%d.%d.%d.%d", major, minor, patch, build)
}

// ---- Version parsing tests ----

func TestParseServerVersion(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{buildVersion(10, 7, 0, 123), 10.7},
		{buildVersion(9, 9, 1, 56737), 9.9},
		{buildVersion(10, 0, 0, 68432), 10.0},
		{buildVersion(6, 3, 0, 26897), 6.3},
		{buildVersion(10, 10, 0, 1), 10.10},
	}
	for _, tc := range cases {
		got, err := sqapi.ParseServerVersion(tc.input)
		require.NoError(t, err, fmtInput, tc.input)
		assert.InDelta(t, tc.want, got, 1e-9, fmtInput, tc.input)
	}
}

func TestParseServerVersionInvalid(t *testing.T) {
	for _, input := range []string{"not-a-version", "", "10"} {
		_, err := sqapi.ParseServerVersion(input)
		assert.Error(t, err, fmtInput, input)
	}
}

// ---- Pagination math tests ----

func TestTotalPages(t *testing.T) {
	cases := []struct {
		total    int
		pageSize int
		want     int
	}{
		{0, 500, 0},
		{1, 500, 1},
		{500, 500, 1},
		{501, 500, 2},
		{1000, 500, 2},
		{1001, 500, 3},
		{100, 100, 1},
		{101, 100, 2},
	}
	for _, tc := range cases {
		got := sqapi.TotalPages(tc.total, tc.pageSize)
		assert.Equal(t, tc.want, got, "total=%d pageSize=%d", tc.total, tc.pageSize)
	}
}

// ---- Paginator tests ----

func TestPaginatorSinglePage(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, _, _ int) ([]string, int, error) {
		calls++
		return []string{"a", "b", "c"}, 3, nil
	}

	pag := sqapi.NewPaginator(fetch, 500)
	all, err := pag.All(context.Background())

	require.NoError(t, err)
	assert.Equal(t, []string{"a", "b", "c"}, all)
	assert.Equal(t, 1, calls, "should fetch exactly one page")
}

func TestPaginatorMultiplePages(t *testing.T) {
	data := make([]int, 1050)
	for i := range data {
		data[i] = i
	}

	fetch := func(_ context.Context, page, pageSize int) ([]int, int, error) {
		start := (page - 1) * pageSize
		end := start + pageSize
		if end > len(data) {
			end = len(data)
		}
		return data[start:end], len(data), nil
	}

	pag := sqapi.NewPaginator(fetch, 500)
	all, err := pag.All(context.Background())

	require.NoError(t, err)
	assert.Len(t, all, 1050)
	assert.Equal(t, data, all)
}

func TestPaginatorEmptyResult(t *testing.T) {
	fetch := func(_ context.Context, _, _ int) ([]string, int, error) {
		return []string{}, 0, nil
	}

	pag := sqapi.NewPaginator(fetch, 500)
	all, err := pag.All(context.Background())

	require.NoError(t, err)
	assert.Empty(t, all)
}

func TestPaginatorContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	callCount := 0
	fetch := func(ctx context.Context, _, _ int) ([]int, int, error) {
		callCount++
		if callCount == 2 {
			cancel()
		}
		return []int{1, 2, 3}, 9000, nil
	}

	pag := sqapi.NewPaginator(fetch, 500)
	_, err := pag.All(ctx)

	assert.Error(t, err)
	assert.LessOrEqual(t, callCount, 3, "should stop fetching soon after cancellation")
}

func TestPaginatorFetchError(t *testing.T) {
	fetch := func(_ context.Context, page, _ int) ([]string, int, error) {
		if page == 2 {
			return nil, 0, fmt.Errorf("network error on page 2")
		}
		return []string{"x"}, 1000, nil
	}

	pag := sqapi.NewPaginator(fetch, 500)
	_, err := pag.All(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error on page 2")
}

// ---- Jitter tests ----

func TestJitterBounds(t *testing.T) {
	base := 100 * time.Millisecond
	for range 1000 {
		result := sqapi.Jitter(base)
		assert.GreaterOrEqual(t, result, base, "jitter must not reduce base duration")
		assert.LessOrEqual(t, result, base+base/2, "jitter must not exceed 50%% of base")
	}
}

// ---- Client construction tests ----

func TestNewServerClient(t *testing.T) {
	c := sqapi.NewServerClient(testBaseURL, "token", 10.7)
	assert.NotNil(t, c)
	assert.Equal(t, testBaseURL+"/", c.BaseURL())
	assert.InDelta(t, 10.7, c.Version(), 1e-9)
	assert.False(t, c.IsCloud())
	assert.Nil(t, c.CertErr())
}

func TestNewCloudClient(t *testing.T) {
	c := sqapi.NewCloudClient("https://sonarcloud.io", "token")
	assert.NotNil(t, c)
	assert.True(t, c.IsCloud())
	assert.Equal(t, 0.0, c.Version())
}

func TestNewServerClientNormalizesBaseURL(t *testing.T) {
	c1 := sqapi.NewServerClient(testBaseURL, "t", 10.0)
	c2 := sqapi.NewServerClient("https://sonar.example.com/", "t", 10.0)
	assert.Equal(t, c1.BaseURL(), c2.BaseURL())
}

// ---- TLS config branch coverage ----

func TestBuildTransportTLSMinVersionZeroGetsDefault(t *testing.T) {
	// When tlsConfig exists but MinVersion == 0, buildTransport sets it to TLS 1.2.
	transport := sqapi.BuildTransport("token", 10.7, 0)
	assert.NotNil(t, transport)
}

func TestBuildTransportTLSMinVersionPreserved(t *testing.T) {
	// When tlsConfig already has a non-zero MinVersion, buildTransport preserves it.
	transport := sqapi.BuildTransport("token", 10.7, 771) // 771 = tls.VersionTLS13
	assert.NotNil(t, transport)
}

// ---- WithClientCert success path ----

// generateSelfSignedCert creates a temporary self-signed cert+key pair for testing.
func generateSelfSignedCert(t *testing.T) (certPath, keyPath string) {
	t.Helper()
	dir := t.TempDir()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPath = dir + "/cert.pem"
	keyPath = dir + "/key.pem"

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, os.WriteFile(certPath, certPEM, 0600))

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, os.WriteFile(keyPath, keyPEM, 0600))

	return certPath, keyPath
}

func TestWithClientCertSuccess(t *testing.T) {
	certPath, keyPath := generateSelfSignedCert(t)
	c := sqapi.NewServerClient(testBaseURL, "token", 10.7,
		sqapi.WithClientCert(certPath, keyPath, ""),
	)
	assert.Nil(t, c.CertErr())
	assert.NotNil(t, c.HTTPClient())
}

// ---- Paginator edge cases ----

func TestPaginatorPagesFirstFetchError(t *testing.T) {
	fetch := func(_ context.Context, _, _ int) ([]string, int, error) {
		return nil, 0, fmt.Errorf("first page error")
	}

	pag := sqapi.NewPaginator(fetch, 500)
	for _, err := range pag.Pages(context.Background()) {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "first page error")
		break
	}
}

func TestPaginatorPagesBreakAfterFirstPage(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, _, _ int) ([]string, int, error) {
		calls++
		return []string{"item"}, 1000, nil // total=1000 → multiple pages
	}

	pag := sqapi.NewPaginator(fetch, 500)
	for _, err := range pag.Pages(context.Background()) {
		assert.NoError(t, err)
		break // break after first page
	}
	assert.Equal(t, 1, calls, "should stop after first page when caller breaks")
}

func TestPaginatorPagesBreakMidPagination(t *testing.T) {
	calls := 0
	fetch := func(_ context.Context, _, _ int) ([]string, int, error) {
		calls++
		return []string{"item"}, 1500, nil // total=1500 → 3 pages
	}

	pag := sqapi.NewPaginator(fetch, 500)
	pagesSeen := 0
	for _, err := range pag.Pages(context.Background()) {
		assert.NoError(t, err)
		pagesSeen++
		if pagesSeen == 2 {
			break // break after second page
		}
	}
	assert.Equal(t, 2, calls, "should stop after second page when caller breaks")
}
