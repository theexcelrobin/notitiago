package notitiago

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestRequiresAPIKey(t *testing.T) {
	if _, err := New().WebKey(context.Background()); !errors.Is(err, ErrNoAPIKey) {
		t.Fatalf("got %v", err)
	}
}

func TestRequestHeaders(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "test-key" || r.Header.Get("User-Agent") != "notitiago/"+Version {
			t.Errorf("bad headers: %v", r.Header)
		}
		ok(w, `"BJ6s"`)
	})
	key, err := c.WebKey(context.Background())
	if err != nil || key != "BJ6s" {
		t.Fatalf("key=%q err=%v", key, err)
	}
}

func TestErrorParsing(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		header map[string]string
		is     error
		isNot  error
		msg    string
		retry  time.Duration
	}{
		{"envelope", 400, `{"success":false,"version":"v1","message":"Unknown topic: choir","payload":null}`, nil, ErrBadRequest, ErrNotFound, "Unknown topic: choir", 0},
		{"unauthorized", 401, `{"success":false,"message":"Invalid API key"}`, nil, ErrUnauthorized, ErrForbidden, "Invalid API key", 0},
		{"rate limit body", 429, `{"error":"Too Many Requests","retry":"12.4s"}`, nil, ErrRateLimited, ErrBadRequest, "Too Many Requests", 12400 * time.Millisecond},
		{"rate limit header wins", 429, `{"error":"Too Many Requests","retry":"12.4s"}`, map[string]string{"Retry-After": "13"}, ErrRateLimited, ErrBadRequest, "Too Many Requests", 13 * time.Second},
		{"plain text", 502, `bad gateway`, nil, nil, ErrNotFound, "bad gateway", 0},
		{"empty object", 500, `{}`, nil, nil, ErrNotFound, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				for k, v := range tt.header {
					w.Header().Set(k, v)
				}
				writeJSON(w, tt.status, tt.body)
			})
			_, err := c.WebKey(context.Background())

			var apiErr *APIError
			if !errors.As(err, &apiErr) {
				t.Fatalf("want *APIError, got %v", err)
			}
			if apiErr.Status != tt.status || apiErr.Message != tt.msg || apiErr.RetryAfter != tt.retry {
				t.Fatalf("unexpected error %+v", apiErr)
			}
			if tt.is != nil && !errors.Is(err, tt.is) {
				t.Fatalf("errors.Is(%v) = false", tt.is)
			}
			if errors.Is(err, tt.isNot) {
				t.Fatalf("must not match %v", tt.isNot)
			}
		})
	}
}

func TestSuccessFalseInA200IsAnError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, `{"success":false,"message":"nope"}`)
	})
	_, err := c.WebKey(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Message != "nope" {
		t.Fatalf("got %v", err)
	}
}

func TestNonJSONSuccessResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, `<html>`) })
	if _, err := c.WebKey(context.Background()); err == nil {
		t.Fatal("want decode error")
	}
}

func TestRetryOn429(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			writeJSON(w, 429, `{"error":"Too Many Requests","retry":"1s"}`)
			return
		}
		ok(w, `"key"`)
	}, WithMaxRetries(1))

	key, err := c.WebKey(context.Background())
	if err != nil || key != "key" || calls.Load() != 2 {
		t.Fatalf("key=%q err=%v calls=%d", key, err, calls.Load())
	}
}

func TestNoRetryByDefaultOrForServerErrors(t *testing.T) {
	for _, status := range []int{429, 500} {
		var calls atomic.Int32
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			writeJSON(w, status, `{}`)
		})
		if _, err := c.WebKey(context.Background()); err == nil || calls.Load() != 1 {
			t.Fatalf("status %d: err=%v calls=%d", status, err, calls.Load())
		}
	}

	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		writeJSON(w, 500, `{}`)
	}, WithMaxRetries(3))
	if _, err := c.WebKey(context.Background()); err == nil || calls.Load() != 1 {
		t.Fatalf("5xx must not retry: err=%v calls=%d", err, calls.Load())
	}
}

func TestRetryStopsOnContextCancel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		writeJSON(w, 429, `{}`)
	}, WithMaxRetries(3))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.WebKey(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
}

func TestPtr(t *testing.T) {
	p := Ptr("x")
	if *p != "x" {
		t.Fatal("Ptr broken")
	}
}

func TestIndexRef(t *testing.T) {
	if IndexRef(218756) != "218756" {
		t.Fatal("IndexRef broken")
	}
}
