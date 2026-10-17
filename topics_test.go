package notitiago

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"
)

func TestCreateTopic(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodPost, "/developers/v1/create-topic")
		want := map[string]any{"name": "youth_church", "description": "Updates for the youth fellowship"}
		if got := readJSON(t, r); !reflect.DeepEqual(got, want) {
			t.Errorf("body = %v", got)
		}
		ok(w, `{"id":"t1","name":"youth_church","project":"youth-church-app","description":"Updates for the youth fellowship","android_count":0,"web_count":0,"subscriber_count":0,"created_at":"2026-10-02T10:00:00Z","updated_at":"2026-10-02T10:00:00Z"}`)
	})
	topic, err := c.CreateTopic(context.Background(), "youth_church", "Updates for the youth fellowship")
	if err != nil || topic.ID != "t1" || topic.Name != "youth_church" || topic.CreatedAt.Year() != 2026 {
		t.Fatalf("topic=%+v err=%v", topic, err)
	}
}

func TestCreateTopicOmitsEmptyDescription(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if _, has := readJSON(t, r)["description"]; has {
			t.Error("empty description must be omitted")
		}
		ok(w, `{"id":"t1","name":"choir"}`)
	})
	if _, err := c.CreateTopic(context.Background(), "choir", ""); err != nil {
		t.Fatal(err)
	}
}

func TestCreateTopicNameValidation(t *testing.T) {
	c := newOfflineClient(t)
	for _, name := range []string{"", "ab", "has space", "dash-ed", "waytoolongtopicnamehere1", "émoji"} {
		if _, err := c.CreateTopic(context.Background(), name, ""); !errors.Is(err, ErrInvalidArgument) {
			t.Errorf("%q: got %v", name, err)
		}
	}
	for _, name := range []string{"abc", "Choir_2026", "a1b2c3d4e5f6g7h8i9j0"} {
		c2 := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { ok(w, `{"id":"t"}`) })
		if _, err := c2.CreateTopic(context.Background(), name, ""); err != nil {
			t.Errorf("%q must be valid: %v", name, err)
		}
	}
}

func TestCreateTopicTaken(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fail(w, 400, "Failed to create topic. Try another unique name")
	})
	if _, err := c.CreateTopic(context.Background(), "choir", ""); !errors.Is(err, ErrBadRequest) {
		t.Fatalf("got %v", err)
	}
}

func TestDeleteTopic(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		expect(t, r, http.MethodDelete, "/developers/v1/delete-topic/t1")
		ok(w, `"Success"`)
	})
	if err := c.DeleteTopic(context.Background(), "t1"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteTopic(context.Background(), ""); !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("got %v", err)
	}
}
