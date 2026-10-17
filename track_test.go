package notitiago

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestTrack(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/public/v1/track/clicked" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "" {
			t.Error("tracking must not send the api key")
		}
		want := map[string]any{"id": "01a0", "sig": "q8Kp"}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, `{"recorded":true}`)
	})
	recorded, err := c.Track(context.Background(), TrackClicked, "01a0", "q8Kp")
	if err != nil || !recorded {
		t.Fatalf("recorded=%v err=%v", recorded, err)
	}
}

func TestTrackWorksWithoutAPIKey(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { ok(w, `{"recorded":false}`) }, WithAPIKey(""))
	recorded, err := c.Track(context.Background(), TrackDelivered, "id", "sig")
	if err != nil || recorded {
		t.Fatalf("recorded=%v err=%v", recorded, err)
	}
}

func TestTrackBadSignature(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fail(w, 403, "Invalid tracking signature") })
	if _, err := c.Track(context.Background(), TrackClicked, "id", "bad"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestTrackValidation(t *testing.T) {
	c := newOfflineClient(t)
	cases := map[string]func() error{
		"bad event": func() error { _, err := c.Track(context.Background(), "opened", "id", "sig"); return err },
		"no id":     func() error { _, err := c.Track(context.Background(), TrackClicked, "", "sig"); return err },
		"no sig":    func() error { _, err := c.Track(context.Background(), TrackClicked, "id", " "); return err },
	}
	for name, fn := range cases {
		if err := fn(); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestClickURL(t *testing.T) {
	c := New(WithBaseURL("https://api.example.test/"))
	if got := c.ClickURL("01a0", "q8/Kp"); got != "https://api.example.test/t/01a0/q8%2FKp" {
		t.Fatalf("got %s", got)
	}
}
