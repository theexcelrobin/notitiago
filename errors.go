package notitiago

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoAPIKey        = errors.New("notitiago: api key is not configured")
	ErrInvalidArgument = errors.New("notitiago: invalid argument")
)

// APIError is returned when the service answers with an error.
type APIError struct {
	Status     int
	Message    string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return "notitiago: status " + strconv.Itoa(e.Status)
	}
	return fmt.Sprintf("notitiago: status %d: %s", e.Status, e.Message)
}

// Is lets errors.Is match an APIError against the sentinels below by status.
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	return ok && t.Status != 0 && t.Status == e.Status
}

var (
	ErrBadRequest   = &APIError{Status: http.StatusBadRequest}
	ErrUnauthorized = &APIError{Status: http.StatusUnauthorized}
	ErrForbidden    = &APIError{Status: http.StatusForbidden}
	ErrNotFound     = &APIError{Status: http.StatusNotFound}
	ErrRateLimited  = &APIError{Status: http.StatusTooManyRequests}
)

type envelope struct {
	Success bool            `json:"success"`
	Version string          `json:"version"`
	Message string          `json:"message"`
	Payload json.RawMessage `json:"payload"`
}

type rateLimitBody struct {
	Error string `json:"error"`
	Retry string `json:"retry"`
}

func newAPIError(status int, header http.Header, body []byte) *APIError {
	e := &APIError{Status: status}

	var env envelope
	var rl rateLimitBody
	switch {
	case json.Unmarshal(body, &env) == nil && env.Message != "":
		e.Message = env.Message
	case json.Unmarshal(body, &rl) == nil && rl.Error != "":
		e.Message = rl.Error
		if d, err := time.ParseDuration(rl.Retry); err == nil && d > 0 {
			e.RetryAfter = d
		}
	default:
		e.Message = snippet(body)
	}

	if secs, err := strconv.Atoi(header.Get("Retry-After")); err == nil && secs > 0 {
		e.RetryAfter = time.Duration(secs) * time.Second
	}
	return e
}

func snippet(body []byte) string {
	s := strings.TrimSpace(string(body))
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	if s == "{}" || s == "null" {
		return ""
	}
	return s
}

func invalidArg(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidArgument, fmt.Sprintf(format, args...))
}
