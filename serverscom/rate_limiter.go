package serverscom

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// rateLimit429MaxRetries caps retries on 429. Total wait at interval 1s
// with exponential backoff is roughly 1+2+4 = 7s before giving up.
const rateLimit429MaxRetries = 3

// RateLimitedHTTPClient spaces requests at least requestInterval seconds apart
// and retries on 429 with exponential backoff. Servers.com Public API advertises
// 2000 req/min per token (`x-ratelimit-limit: 2000`); the 1-second floor here is
// conservative — same shape as terraform-provider-shodan.
type RateLimitedHTTPClient struct {
	client          *http.Client
	requestInterval int64
	lastRequest     time.Time
	mu              sync.Mutex
}

func NewRateLimitedHTTPClient(client *http.Client, requestIntervalSeconds int64) *RateLimitedHTTPClient {
	if requestIntervalSeconds < 1 {
		requestIntervalSeconds = 1
	}
	return &RateLimitedHTTPClient{
		client:          client,
		requestInterval: requestIntervalSeconds,
		lastRequest:     time.Time{},
	}
}

func (r *RateLimitedHTTPClient) Do(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	minInterval := time.Duration(r.requestInterval) * time.Second

	// Buffer the body so we can replay on retry. GET requests have no body,
	// but POST/PUT/DELETE may, and io.Reader bodies are single-shot.
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to buffer request body for retry: %w", err)
		}
		req.Body.Close()
	}

	var resp *http.Response
	var lastErr error

	for attempt := 0; attempt <= rateLimit429MaxRetries; attempt++ {
		if !r.lastRequest.IsZero() {
			timeSinceLast := time.Since(r.lastRequest)
			if timeSinceLast < minInterval {
				time.Sleep(minInterval - timeSinceLast)
			}
		}

		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		r.lastRequest = time.Now()
		resp, lastErr = r.client.Do(req)
		if lastErr != nil {
			return nil, lastErr
		}

		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}

		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if attempt == rateLimit429MaxRetries {
			return resp, nil
		}

		backoff := minInterval * time.Duration(1<<attempt)
		time.Sleep(backoff)
	}

	return resp, nil
}
