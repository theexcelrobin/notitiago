package notitiago

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc, opts ...Option) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	base := []Option{WithBaseURL(srv.URL), WithAPIKey("test-key")}
	return New(append(base, opts...)...)
}

type forbidTransport struct{ t *testing.T }

func (f forbidTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	f.t.Errorf("unexpected network call: %s %s", r.Method, r.URL)
	return nil, io.ErrUnexpectedEOF
}

func newOfflineClient(t *testing.T) *Client {
	t.Helper()
	return New(WithAPIKey("k"), WithHTTPClient(&http.Client{Transport: forbidTransport{t}}))
}

func ok(w http.ResponseWriter, payload string) {
	writeJSON(w, 200, `{"success":true,"version":"v1","message":"","payload":`+payload+`}`)
}

func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, `{"success":false,"version":"v1","message":"`+message+`","payload":null}`)
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func readJSON(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return m
}

func expect(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Errorf("got %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
	if r.Header.Get("X-Api-Key") != "test-key" {
		t.Errorf("missing api key header: %v", r.Header)
	}
}
