package notitiago

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// IndexRef turns a notification or campaign index into the reference the
// lookup methods take. Those methods accept an ID or an index.
func IndexRef(index int64) string { return strconv.FormatInt(index, 10) }

// SendRequest sends a notification to every device of one identifier.
type SendRequest struct {
	// Identifier is who to send to, as used when subscribing.
	Identifier string
	// Topic restricts the match to devices in this topic. Empty means DefaultTopic.
	Topic string
	// Target limits the platforms. Empty means TargetAll.
	Target Target
	Content
}

type sendWire struct {
	Identifier string `json:"identifier"`
	Topic      string `json:"topic,omitempty"`
	Target     Target `json:"target,omitempty"`
	contentWire
}

// Send queues a notification for one person and returns the index of the
// first notification created, one per matching device. Delivery happens in the
// background with retries, so look the result up with GetNotification.
//
// When no device matches, the call still succeeds and the notification is
// stored as failed with the code no_subscription.
func (c *Client) Send(ctx context.Context, r SendRequest) (int64, error) {
	if strings.TrimSpace(r.Identifier) == "" {
		return 0, invalidArg("identifier is required")
	}
	if !validTarget(r.Target) {
		return 0, invalidArg("target must be ALL, WEB or ANDROID")
	}
	if err := r.Content.validate(); err != nil {
		return 0, err
	}

	var index int64
	wire := sendWire{Identifier: r.Identifier, Topic: r.Topic, Target: r.Target, contentWire: r.Content.wire()}
	err := c.do(ctx, request{method: http.MethodPost, path: c.api("/send-notification"), body: wire}, &index)
	if err != nil {
		return 0, err
	}
	return index, nil
}

// BroadcastRequest sends to everyone in one or more topics.
type BroadcastRequest struct {
	// Name labels the campaign in the dashboard, up to 120 characters.
	Name string
	// Topics are names or IDs, up to 50. Empty uses Topic, then DefaultTopic.
	Topics []string
	// Topic is a single topic, kept for older callers.
	Topic string
	// Target limits the platforms. Empty means TargetAll.
	Target Target
	// ScheduledAt sends later. A time in the past sends immediately.
	ScheduledAt *time.Time
	Content
}

type broadcastWire struct {
	Name        string     `json:"name,omitempty"`
	Topic       string     `json:"topic,omitempty"`
	Topics      []string   `json:"topics,omitempty"`
	Target      Target     `json:"target,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	contentWire
}

// Broadcast sends to topics by creating a campaign, and returns the campaign
// index. People in several of the topics get it once.
func (c *Client) Broadcast(ctx context.Context, r BroadcastRequest) (int64, error) {
	if len(r.Name) > 120 {
		return 0, invalidArg("name must be 120 characters or fewer")
	}
	if !validTarget(r.Target) {
		return 0, invalidArg("target must be ALL, WEB or ANDROID")
	}
	if err := validTopicList(r.Topics); err != nil {
		return 0, err
	}
	if err := r.Content.validate(); err != nil {
		return 0, err
	}

	var index int64
	wire := broadcastWire{
		Name: r.Name, Topic: r.Topic, Topics: r.Topics, Target: r.Target,
		ScheduledAt: r.ScheduledAt, contentWire: r.Content.wire(),
	}
	err := c.do(ctx, request{method: http.MethodPost, path: c.api("/broadcast-notification"), body: wire}, &index)
	if err != nil {
		return 0, err
	}
	return index, nil
}

// GetNotification returns a notification with its delivery attempts and
// events. ref is the notification ID or its index; see IndexRef.
func (c *Client) GetNotification(ctx context.Context, ref string) (*Notification, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, invalidArg("notification reference is required")
	}
	var n Notification
	path := c.api("/notifications/" + url.PathEscape(ref))
	if err := c.do(ctx, request{method: http.MethodGet, path: path}, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// UpdateRequest changes a stored notification. Nil fields keep their value.
type UpdateRequest struct {
	Title *string
	Body  *string
	Image *string
	Tag   *string
	// Data replaces the custom data when set.
	Data map[string]string
	// Topic, when set, also sends the updated notification to that topic as a
	// new campaign. URL and Route are used only for that resend.
	Topic string
	URL   string
	Route string
}

type updateWire struct {
	Title *string           `json:"title,omitempty"`
	Body  *string           `json:"body,omitempty"`
	Image *string           `json:"image,omitempty"`
	Tag   *string           `json:"tag,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
	Topic string            `json:"topic,omitempty"`
	URL   string            `json:"url,omitempty"`
	Route string            `json:"route,omitempty"`
}

// UpdateNotification changes what is stored for a notification and returns its
// index. It does not change anything already shown on a device; to replace
// that, send the new version with the same Tag. id is the notification ID, not
// its index.
func (c *Client) UpdateNotification(ctx context.Context, id string, r UpdateRequest) (int64, error) {
	if strings.TrimSpace(id) == "" {
		return 0, invalidArg("notification id is required")
	}
	for k := range r.Data {
		if strings.HasPrefix(k, reservedPrefix) {
			return 0, invalidArg("data key %q uses the reserved %s prefix", k, reservedPrefix)
		}
	}

	var index int64
	wire := updateWire(r)
	path := c.api("/update-notification/" + url.PathEscape(id))
	if err := c.do(ctx, request{method: http.MethodPatch, path: path, body: wire}, &index); err != nil {
		return 0, err
	}
	return index, nil
}

// DeleteNotification removes a notification and its delivery history. It
// cannot recall one already on a device. id is the notification ID, not its index.
func (c *Client) DeleteNotification(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return invalidArg("notification id is required")
	}
	path := c.api("/delete-notification/" + url.PathEscape(id))
	return c.do(ctx, request{method: http.MethodDelete, path: path}, nil)
}
