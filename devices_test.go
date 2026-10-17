package notitiago

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestSubscribeWeb(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPost, "/developers/v1/subscribe")
		want := map[string]any{
			"target": "WEB", "identifier": "user-1042", "topics": []any{"youth_church", "choir"},
			"web":      map[string]any{"endpoint": "https://fcm.googleapis.com/x", "keys": map[string]any{"p256dh": "P", "auth": "A"}},
			"recreate": true,
		}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, `"user-1042"`)
	})
	id, err := c.Subscribe(context.Background(), SubscribeRequest{
		Target: TargetWeb, Identifier: "user-1042", Topics: []string{"youth_church", "choir"}, Recreate: true,
		Web: &WebDevice{Endpoint: "https://fcm.googleapis.com/x", Keys: WebKeys{P256DH: "P", Auth: "A"}},
	})
	if err != nil || id != "user-1042" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestSubscribeAndroidOmitsOptionals(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := readJSON(t, r)
		if _, has := got["identifier"]; has {
			t.Error("empty identifier must be omitted so one is generated")
		}
		if _, has := got["web"]; has {
			t.Error("web must be omitted")
		}
		if _, has := got["recreate"]; has {
			t.Error("recreate must be omitted when false")
		}
		if got["android"].(map[string]any)["fcm_token"] != "tok" {
			t.Errorf("body = %v", got)
		}
		ok(w, `"generated-id"`)
	})
	id, err := c.Subscribe(context.Background(), SubscribeRequest{Target: TargetAndroid, Android: &AndroidDevice{FCMToken: "tok"}})
	if err != nil || id != "generated-id" {
		t.Fatalf("id=%q err=%v", id, err)
	}
}

func TestSubscribeAcceptsBrowserSubscriptionJSON(t *testing.T) {
	var d WebDevice
	raw := `{"endpoint":"https://x","expirationTime":null,"keys":{"p256dh":"P","auth":"A"}}`
	if err := json.Unmarshal([]byte(raw), &d); err != nil || d.Endpoint != "https://x" || d.Keys.Auth != "A" {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestSubscribeValidation(t *testing.T) {
	c := newOfflineClient(t)
	tooMany := make([]string, 51)
	for i := range tooMany {
		tooMany[i] = "t"
	}
	cases := map[string]SubscribeRequest{
		"no target":          {},
		"target all":         {Target: TargetAll},
		"android no payload": {Target: TargetAndroid},
		"android empty":      {Target: TargetAndroid, Android: &AndroidDevice{}},
		"web no payload":     {Target: TargetWeb},
		"web missing keys":   {Target: TargetWeb, Web: &WebDevice{Endpoint: "https://x"}},
		"too many topics":    {Target: TargetAndroid, Android: &AndroidDevice{FCMToken: "t"}, Topics: tooMany},
		"blank topic":        {Target: TargetAndroid, Android: &AndroidDevice{FCMToken: "t"}, Topics: []string{" "}},
	}
	for name, r := range cases {
		if _, err := c.Subscribe(context.Background(), r); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestSubscribeUnknownTopicError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fail(w, 400, "Unknown topic: choir") })
	_, err := c.Subscribe(context.Background(), SubscribeRequest{Target: TargetAndroid, Android: &AndroidDevice{FCMToken: "t"}, Topics: []string{"choir"}})
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestUnsubscribe(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPost, "/developers/v1/unsubscribe")
		want := map[string]any{"target": "WEB", "identifier": "user-1042", "topics": []any{"choir"}}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, `"user-1042"`)
	})
	id, err := c.Unsubscribe(context.Background(), UnsubscribeRequest{Target: TargetWeb, Identifier: "user-1042", Topics: []string{"choir"}})
	if err != nil || id != "user-1042" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if _, err := c.Unsubscribe(context.Background(), UnsubscribeRequest{Target: TargetWeb}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("got %v", err)
	}
}

func TestWebKey(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodGet, "/developers/v1/web-key")
		ok(w, `"BJ6sJl5mE6w7"`)
	})
	key, err := c.WebKey(context.Background())
	if err != nil || key != "BJ6sJl5mE6w7" {
		t.Fatalf("key=%q err=%v", key, err)
	}
}
