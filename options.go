package notitiago

import (
	"net/http"
	"strings"
	"time"
)

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the project API key sent in the X-Api-Key header.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = strings.TrimSpace(key) }
}

// WithBaseURL overrides the service host, for self-hosted or test servers.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

// WithHTTPClient replaces the default http.Client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.http = hc
		}
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

// WithTimeout sets the per-request timeout for API calls. Zero disables it.
// File uploads are bounded by their context only.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.timeout = d }
}

// WithMaxRetries retries a request up to n more times after a 429, waiting
// for the delay the service asks for. Nothing else is retried, so a send is
// never delivered twice. Uploads are not retried.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.maxRetries = n
		}
	}
}
