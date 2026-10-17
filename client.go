package notitiago

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	Version = "0.1.0"

	DefaultBaseURL = "https://api.lwnotifications.com"

	apiPrefix       = "/developers/v1"
	defaultTimeout  = 30 * time.Second
	maxResponseSize = 10 << 20
)

// Client talks to the notification service. It is safe for concurrent use.
type Client struct {
	apiKey     string
	baseURL    string
	userAgent  string
	timeout    time.Duration
	maxRetries int
	http       *http.Client
}

// New creates a Client.
func New(opts ...Option) *Client {
	c := &Client{
		baseURL:   DefaultBaseURL,
		userAgent: "notitiago/" + Version,
		timeout:   defaultTimeout,
		http:      &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type request struct {
	method      string
	path        string
	query       url.Values
	body        any
	stream      io.Reader
	contentType string
	public      bool
}

func (c *Client) do(ctx context.Context, r request, out any) error {
	if !r.public && c.apiKey == "" {
		return ErrNoAPIKey
	}

	var payload []byte
	if r.body != nil {
		b, err := json.Marshal(r.body)
		if err != nil {
			return err
		}
		payload = b
	}

	retries := c.maxRetries
	if r.stream != nil {
		retries = 0
	}

	for attempt := 0; ; attempt++ {
		err := c.send(ctx, r, payload, out)

		var apiErr *APIError
		if err == nil || attempt >= retries || !errors.As(err, &apiErr) || apiErr.Status != http.StatusTooManyRequests {
			return err
		}

		wait := apiErr.RetryAfter
		if wait <= 0 {
			wait = time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

func (c *Client) send(ctx context.Context, r request, payload []byte, out any) error {
	if c.timeout > 0 && r.stream == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	target := c.baseURL + r.path
	if len(r.query) > 0 {
		target += "?" + r.query.Encode()
	}

	var body io.Reader
	switch {
	case r.stream != nil:
		body = r.stream
	case payload != nil:
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, r.method, target, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	switch {
	case r.contentType != "":
		req.Header.Set("Content-Type", r.contentType)
	case payload != nil:
		req.Header.Set("Content-Type", "application/json")
	}
	if !r.public {
		req.Header.Set("X-Api-Key", c.apiKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return newAPIError(resp.StatusCode, resp.Header, data)
	}

	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return fmt.Errorf("notitiago: decode response: %w", err)
	}
	if !env.Success {
		return &APIError{Status: resp.StatusCode, Message: env.Message}
	}
	if out == nil || len(env.Payload) == 0 || string(env.Payload) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Payload, out); err != nil {
		return fmt.Errorf("notitiago: decode payload: %w", err)
	}
	return nil
}

func (c *Client) api(path string) string { return apiPrefix + path }
