package notitiago

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const campaignFixture = `{
  "id":"01a0fbc2-e8ce-70bb-bc49-7b8e0b53cb7c","index":72,"project_id":"p1","name":"Sunday service is live","source":"api",
  "status":"sending","target":"ALL","title":"Sunday service is live","body":"Tap to watch.","image":null,"icon":null,
  "url":"https://example.org/live","route":null,"tag":"live-service","data":{"stream_id":"8812"},"ttl_seconds":259200,
  "urgency":"high","tracked":true,"failure_reason":null,"topics":[{"id":"t1","name":"default"}],
  "stats":{"total":5342,"sent":3120,"failed":41,"delivered":2604,"clicked":388,"pending":2181,"retryable":0,
           "delivery_rate":0.8346,"click_rate":0.1243,"failure_rate":0.0076},
  "scheduled_at":null,"started_at":"2026-09-08T16:40:02Z","completed_at":null,"cancelled_at":null,
  "created_at":"2026-09-08T16:40:00Z","updated_at":"2026-09-08T16:41:13Z"
}`

func campaignWithStatus(status string) string {
	return strings.Replace(campaignFixture, `"status":"sending"`, `"status":"`+status+`"`, 1)
}

func TestCreateCampaign(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPost, "/developers/v1/campaigns")
		want := map[string]any{
			"name": "Choir rehearsal moved", "topics": []any{"choir"}, "title": "Rehearsal moved to Saturday",
			"body": "Same time, main auditorium.", "tag": "rehearsal", "draft": true,
		}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, campaignFixture)
	})
	cmp, err := c.CreateCampaign(context.Background(), CampaignRequest{
		Name: "Choir rehearsal moved", Topics: []string{"choir"}, Draft: true,
		Content: Content{Title: "Rehearsal moved to Saturday", Body: "Same time, main auditorium.", Tag: "rehearsal"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmp.Index != 72 || cmp.Status != CampaignSending || cmp.Stats.Total != 5342 || cmp.Stats.DeliveryRate != 0.8346 ||
		len(cmp.Topics) != 1 || cmp.Topics[0].Name != "default" || cmp.StartedAt == nil || cmp.CompletedAt != nil || cmp.Data["stream_id"] != "8812" {
		t.Fatalf("campaign = %+v", cmp)
	}
}

func TestCreateCampaignStrictValidation(t *testing.T) {
	c := newOfflineClient(t)
	mut := func(f func(*Content)) CampaignRequest {
		ct := Content{Title: "t"}
		f(&ct)
		return CampaignRequest{Content: ct}
	}
	cases := map[string]CampaignRequest{
		"long title": mut(func(c *Content) { c.Title = strings.Repeat("a", 241) }),
		"long body":  mut(func(c *Content) { c.Body = strings.Repeat("a", 4001) }),
		"long route": mut(func(c *Content) { c.Route = strings.Repeat("a", 501) }),
		"long tag":   mut(func(c *Content) { c.Tag = strings.Repeat("a", 121) }),
		"bad image":  mut(func(c *Content) { c.Image = "not a url" }),
		"bad icon":   mut(func(c *Content) { c.Icon = "ftp://x/y.png" }),
		"bad url":    mut(func(c *Content) { c.URL = "/relative" }),
		"long name":  {Name: strings.Repeat("a", 121), Content: Content{Title: "t"}},
		"bad target": {Target: "x", Content: Content{Title: "t"}},
	}
	for name, r := range cases {
		if _, err := c.CreateCampaign(context.Background(), r); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}

func TestGetAndCancelCampaign(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/developers/v1/campaigns/72":
			ok(w, campaignFixture)
		case r.Method == http.MethodPost && r.URL.Path == "/developers/v1/campaigns/72/cancel":
			ok(w, campaignWithStatus("cancelled"))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
	})
	got, err := c.GetCampaign(context.Background(), IndexRef(72))
	if err != nil || got.Status != CampaignSending {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	cancelled, err := c.CancelCampaign(context.Background(), "72")
	if err != nil || cancelled.Status != CampaignCancelled {
		t.Fatalf("cancelled=%+v err=%v", cancelled, err)
	}
}

func TestCancelFinishedCampaign(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fail(w, 400, "This campaign can no longer be cancelled")
	})
	_, err := c.CancelCampaign(context.Background(), "72")
	if !errors.Is(err, ErrBadRequest) || !strings.Contains(err.Error(), "no longer") {
		t.Fatalf("got %v", err)
	}
}

func TestCampaignRefValidation(t *testing.T) {
	c := newOfflineClient(t)
	if _, err := c.GetCampaign(context.Background(), ""); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("get: %v", err)
	}
	if _, err := c.CancelCampaign(context.Background(), " "); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("cancel: %v", err)
	}
}

func TestCampaignStatusFinished(t *testing.T) {
	finished := map[CampaignStatus]bool{
		CampaignCompleted: true, CampaignFailed: true, CampaignCancelled: true,
		CampaignDraft: false, CampaignScheduled: false, CampaignQueued: false, CampaignSending: false,
	}
	for s, want := range finished {
		if s.Finished() != want {
			t.Errorf("%s.Finished() = %v", s, !want)
		}
	}
}

func TestWaitForCampaign(t *testing.T) {
	var calls atomic.Int32
	statuses := []string{"queued", "sending", "completed"}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		ok(w, campaignWithStatus(statuses[calls.Add(1)-1]))
	})
	start := time.Now()
	cmp, err := c.WaitForCampaign(context.Background(), "72", time.Millisecond)
	if err != nil || cmp.Status != CampaignCompleted || calls.Load() != 3 {
		t.Fatalf("cmp=%+v err=%v calls=%d", cmp, err, calls.Load())
	}
	if time.Since(start) < 2*time.Second {
		t.Fatal("interval below one second must be raised to one second")
	}
}

func TestWaitForCampaignContextCancel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { ok(w, campaignFixture) })
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	cmp, err := c.WaitForCampaign(ctx, "72", time.Hour)
	if !errors.Is(err, context.DeadlineExceeded) || cmp == nil || cmp.Status != CampaignSending {
		t.Fatalf("cmp=%+v err=%v", cmp, err)
	}
}

func TestWaitForCampaignStopsOnAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "Campaign not found") })
	if _, err := c.WaitForCampaign(context.Background(), "72", time.Second); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
}

func TestListCampaignsQuery(t *testing.T) {
	cursor := &Cursor{CursorCreatedAt: time.Date(2026, 9, 8, 16, 40, 0, 0, time.UTC), CursorID: "abc", Limit: 20}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodGet, "/developers/v1/campaigns")
		q := r.URL.Query()
		want := map[string]string{
			"status": "sending,completed", "source": "api,dashboard", "topic_id": "t1", "search": "live", "limit": "20",
			"cursor_created_at": "2026-09-08T16:40:00Z", "cursor_id": "abc",
		}
		for k, v := range want {
			if q.Get(k) != v {
				t.Errorf("%s = %q, want %q", k, q.Get(k), v)
			}
		}
		ok(w, `{"campaigns":[`+campaignFixture+`],"page":{"cursor_created_at":"2026-09-08T16:40:00Z","cursor_id":"01a0","limit":20}}`)
	})
	page, err := c.ListCampaigns(context.Background(), ListCampaignsParams{
		Status: []CampaignStatus{CampaignSending, CampaignCompleted}, Source: []Source{SourceAPI, SourceDashboard},
		TopicID: "t1", Search: "live", Limit: 20, Cursor: cursor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Campaigns) != 1 || page.Next == nil || page.Next.CursorID != "01a0" || page.Next.Limit != 20 {
		t.Fatalf("page = %+v", page)
	}
}

func TestListCampaignsLastPageHasNilNext(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("no filters must send no query, got %q", r.URL.RawQuery)
		}
		ok(w, `{"campaigns":[],"page":null}`)
	})
	page, err := c.ListCampaigns(context.Background(), ListCampaignsParams{})
	if err != nil || page.Next != nil || len(page.Campaigns) != 0 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
}

func TestListCampaignsValidatesLimit(t *testing.T) {
	c := newOfflineClient(t)
	for _, limit := range []int{-1, 101} {
		if _, err := c.ListCampaigns(context.Background(), ListCampaignsParams{Limit: limit}); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("limit %d: got %v", limit, err)
		}
	}
}

func TestEachCampaignFollowsCursor(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("cursor_id") {
		case "":
			ok(w, `{"campaigns":[{"id":"a","index":3},{"id":"b","index":2}],"page":{"cursor_created_at":"2026-09-08T16:40:00Z","cursor_id":"b","limit":2}}`)
		case "b":
			ok(w, `{"campaigns":[{"id":"c","index":1}],"page":null}`)
		default:
			t.Errorf("unexpected cursor %q", r.URL.Query().Get("cursor_id"))
		}
	})
	var ids []string
	err := c.EachCampaign(context.Background(), ListCampaignsParams{Limit: 2}, func(cmp Campaign) error {
		ids = append(ids, cmp.ID)
		return nil
	})
	if err != nil || fmt.Sprint(ids) != "[a b c]" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
}

func TestEachCampaignStopsOnCallbackError(t *testing.T) {
	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		ok(w, `{"campaigns":[{"id":"a"},{"id":"b"}],"page":{"cursor_created_at":"2026-09-08T16:40:00Z","cursor_id":"b","limit":2}}`)
	})
	boom := errors.New("boom")
	err := c.EachCampaign(context.Background(), ListCampaignsParams{}, func(cmp Campaign) error { return boom })
	if !errors.Is(err, boom) || calls.Load() != 1 {
		t.Fatalf("err=%v calls=%d", err, calls.Load())
	}
}
