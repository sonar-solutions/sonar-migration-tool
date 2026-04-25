package sqapi

import (
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

// defaultBackoff is the base wait duration between retry attempts.
// Attempt 1 fails → wait defaultBackoff[0] before attempt 2, etc.
var defaultBackoff = []time.Duration{
	100 * time.Millisecond,
	200 * time.Millisecond,
	400 * time.Millisecond,
}

// retryableStatusCodes are HTTP status codes that warrant a retry.
// 429 (rate limit) and 5xx (server errors) are retried.
// 4xx client errors (except 429) are not retried — they indicate a caller mistake.
var retryableStatusCodes = map[int]bool{
	http.StatusTooManyRequests:     true,
	http.StatusInternalServerError: true,
	http.StatusBadGateway:          true,
	http.StatusServiceUnavailable:  true,
	http.StatusGatewayTimeout:      true,
}

// retryTransport is an http.RoundTripper that retries failed requests with
// exponential backoff plus up to 50% random jitter.
//
// Total attempts = len(backoff) + 1.
type retryTransport struct {
	inner   http.RoundTripper
	backoff []time.Duration
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var (
		resp *http.Response
		err  error
	)

	totalAttempts := len(t.backoff) + 1

	for attempt := range totalAttempts {
		resp, err = t.inner.RoundTrip(req)

		if err == nil && !retryableStatusCodes[resp.StatusCode] {
			return resp, nil
		}

		// Drain and close the body before retrying so the connection can be reused.
		if resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		if attempt < len(t.backoff) {
			time.Sleep(jitter(t.backoff[attempt]))
		}
	}

	return resp, err
}

// jitter adds a random amount up to 50% of d to d.
func jitter(d time.Duration) time.Duration {
	return d + time.Duration(float64(d)*0.5*rand.Float64())
}
