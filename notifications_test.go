package notitiago

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSend(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPost, "/developers/v1/send-notification")
		want := map[string]any{
			"identifier": "user-1042", "title": "Your order has shipped", "body": "It should arrive on Thursday.",
			"route": "/orders/1042", "tag": "order-1042", "data": map[string]any{"order_id": "1042"},
			"ttl": float64(3600), "urgency": "normal", "target": "ANDROID", "topic": "orders",
		}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, `218756`)
	})
	index, err := c.Send(context.Background(), SendRequest{
		Identifier: "user-1042", Topic: "orders", Target: TargetAndroid,
		Content: Content{
			Title: "Your order has shipped", Body: "It should arrive on Thursday.", Route: "/orders/1042", Tag: "order-1042",
			Data: map[string]string{"order_id": "1042"}, TTL: Ptr(time.Hour), Urgency: UrgencyNormal,
		},
	})
	if err != nil || index != 218756 {
		t.Fatalf("index=%d err=%v", index, err)
	}
}

func TestSendOmitsEmptyFields(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := readJSON(t, r)
		if len(got) != 2 || got["identifier"] != "u" || got["title"] != "t" {
			t.Errorf("body = %v", got)
		}
		ok(w, `1`)
	})
	if _, err := c.Send(context.Background(), SendRequest{Identifier: "u", Content: Content{Title: "t"}}); err != nil {
		t.Fatal(err)
	}
}

func TestSendZeroTTLIsSent(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if ttl, has := readJSON(t, r)["ttl"]; !has || ttl != float64(0) {
			t.Errorf("ttl = %v present=%v", ttl, has)
		}
		ok(w, `1`)
	})
	if _, err := c.Send(context.Background(), SendRequest{Identifier: "u", Content: Content{Title: "t", TTL: Ptr(time.Duration(0))}}); err != nil {
		t.Fatal(err)
	}
}

func TestSendValidation(t *testing.T) {
	c := newOfflineClient(t)
	good := Content{Title: "t"}
	mut := func(f func(*Content)) Content { x := good; f(&x); return x }
	cases := map[string]SendRequest{
		"no identifier": {Content: good},
		"blank title":   {Identifier: "u", Content: Content{Title: "  "}},
		"bad target":    {Identifier: "u", Target: "IOS", Content: good},
		"bad urgency":   {Identifier: "u", Content: mut(func(c *Content) { c.Urgency = "urgent" })},
		"negative ttl":  {Identifier: "u", Content: mut(func(c *Content) { c.TTL = Ptr(-time.Second) })},
		"ttl over 28d":  {Identifier: "u", Content: mut(func(c *Content) { c.TTL = Ptr(29 * 24 * time.Hour) })},
		"reserved key":  {Identifier: "u", Content: mut(func(c *Content) { c.Data = map[string]string{"lwn_id": "x"} })},
	}
	for name, r := range cases {
		if _, err := c.Send(context.Background(), r); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}

	local := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { ok(w, `1`) })
	req := SendRequest{Identifier: "u", Content: mut(func(c *Content) { c.TTL = Ptr(28 * 24 * time.Hour) })}
	if _, err := local.Send(context.Background(), req); err != nil {
		t.Errorf("exactly 28 days must be accepted: %v", err)
	}
}

func TestSendUnknownTopic(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "This project doesn't exist") })
	_, err := c.Send(context.Background(), SendRequest{Identifier: "u", Topic: "nope", Content: Content{Title: "t"}})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestBroadcast(t *testing.T) {
	when := time.Date(2026, 10, 4, 8, 30, 0, 0, time.UTC)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPost, "/developers/v1/broadcast-notification")
		want := map[string]any{
			"topics": []any{"youth_church"}, "title": "Service starts in 30 minutes", "url": "https://example.org/live",
			"scheduled_at": "2026-10-04T08:30:00Z", "name": "Live",
		}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, `86`)
	})
	index, err := c.Broadcast(context.Background(), BroadcastRequest{
		Name: "Live", Topics: []string{"youth_church"}, ScheduledAt: &when,
		Content: Content{Title: "Service starts in 30 minutes", URL: "https://example.org/live"},
	})
	if err != nil || index != 86 {
		t.Fatalf("index=%d err=%v", index, err)
	}
}

func TestBroadcastValidation(t *testing.T) {
	c := newOfflineClient(t)
	cases := map[string]BroadcastRequest{
		"no title":    {},
		"long name":   {Name: strings.Repeat("a", 121), Content: Content{Title: "t"}},
		"bad target":  {Target: "x", Content: Content{Title: "t"}},
		"blank topic": {Topics: []string{""}, Content: Content{Title: "t"}},
	}
	for name, r := range cases {
		if _, err := c.Broadcast(context.Background(), r); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

const notificationFixture = `{
  "id":"01a0fc69-f782-7cdd-856c-b6d85175cc79","index":218756,"topic":"default","target":"ALL","project":"",
  "title":"Your order has shipped","body":"It should arrive on Thursday.","image":null,"tag":"order-1042",
  "data":{"order_id":"1042"},"campaign_id":null,"identifier":"user-1042","platform":"android","source":"api",
  "status":"sent","attempt_count":2,"sent_at":"2026-10-02T09:14:41Z","delivered_at":"2026-10-02T09:14:43Z","clicked_at":null,
  "created_at":"2026-10-02T09:14:08Z","updated_at":"2026-10-02T09:14:43Z","campaign_name":null,"icon":null,"url":null,
  "route":"/orders/1042","ttl_seconds":259200,"urgency":"high","error_code":null,"error_message":null,
  "next_attempt_at":null,"failed_at":null,
  "attempts":[
    {"id":"a1","attempt_no":1,"provider":"fcm","outcome":"transient_error","error_code":"unavailable","error_message":"down","http_status":503,"provider_message_id":null,"latency_ms":412,"created_at":"2026-10-02T09:14:09Z"},
    {"id":"a2","attempt_no":2,"provider":"fcm","outcome":"success","error_code":null,"error_message":null,"http_status":200,"provider_message_id":"projects/x/messages/1","latency_ms":96,"created_at":"2026-10-02T09:14:41Z"}
  ],
  "events":[{"id":"e1","type":"delivered","platform":"android","user_agent":null,"created_at":"2026-10-02T09:14:43Z"}]
}`

func TestGetNotification(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodGet, "/developers/v1/notifications/218756")
		ok(w, notificationFixture)
	})
	n, err := c.GetNotification(context.Background(), IndexRef(218756))
	if err != nil {
		t.Fatal(err)
	}
	if n.Index != 218756 || n.Status != NotificationSent || n.Route != "/orders/1042" || n.Body == "" || n.Image != "" || n.CampaignID != "" {
		t.Fatalf("notification = %+v", n)
	}
	if n.SentAt == nil || n.DeliveredAt == nil || n.ClickedAt != nil || n.FailedAt != nil {
		t.Fatalf("times = %+v", n)
	}
	if len(n.Attempts) != 2 || n.Attempts[0].ErrorCode != ErrCodeUnavailable || !n.Attempts[0].ErrorCode.Temporary() || n.Attempts[1].LatencyMs != 96 || n.Attempts[1].ProviderMessageID == "" {
		t.Fatalf("attempts = %+v", n.Attempts)
	}
	if len(n.Events) != 1 || n.Events[0].Type != "delivered" || n.Data["order_id"] != "1042" || n.Urgency != UrgencyHigh {
		t.Fatalf("events/data = %+v", n)
	}
}

func TestGetNotificationEscapesRefAndValidates(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.EscapedPath() != "/developers/v1/notifications/a%2Fb" {
			t.Errorf("path = %s", r.URL.EscapedPath())
		}
		fail(w, 404, "Notification not found")
	})
	if _, err := c.GetNotification(context.Background(), "a/b"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	if _, err := c.GetNotification(context.Background(), " "); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("got %v", err)
	}
}

func TestFailedNotificationErrorCodes(t *testing.T) {
	if !ErrCodeGone.RemovesDevice() || ErrCodeGone.Temporary() || !ErrCodeQuotaExceeded.Temporary() || ErrCodePayloadTooLarge.Temporary() || ErrCodePayloadTooLarge.RemovesDevice() {
		t.Fatal("error code helpers misbehave")
	}
}

func TestUpdateNotification(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPatch, "/developers/v1/update-notification/01a0")
		want := map[string]any{"title": "Out for delivery", "tag": "order-1042", "topic": "orders", "url": "https://example.org/o"}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, `218756`)
	})
	index, err := c.UpdateNotification(context.Background(), "01a0", UpdateRequest{
		Title: Ptr("Out for delivery"), Tag: Ptr("order-1042"), Topic: "orders", URL: "https://example.org/o",
	})
	if err != nil || index != 218756 {
		t.Fatalf("index=%d err=%v", index, err)
	}
}

func TestUpdateNotificationSendsEmptyStringToClearAField(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got := readJSON(t, r)
		if body, has := got["body"]; !has || body != "" {
			t.Errorf("an empty pointer must be sent as an empty string: %v", got)
		}
		if _, has := got["image"]; has {
			t.Errorf("a nil pointer must be omitted: %v", got)
		}
		ok(w, `1`)
	})
	if _, err := c.UpdateNotification(context.Background(), "id", UpdateRequest{Body: Ptr("")}); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateNotificationValidation(t *testing.T) {
	c := newOfflineClient(t)
	if _, err := c.UpdateNotification(context.Background(), "", UpdateRequest{}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("got %v", err)
	}
	if _, err := c.UpdateNotification(context.Background(), "id", UpdateRequest{Data: map[string]string{"lwn_sig": "x"}}); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteNotification(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodDelete, "/developers/v1/delete-notification/01a0")
		ok(w, `"Success"`)
	})
	if err := c.DeleteNotification(context.Background(), "01a0"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteNotification(context.Background(), ""); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("got %v", err)
	}
}
