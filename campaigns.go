package notitiago

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPollInterval = 3 * time.Second
	minPollInterval     = time.Second
)

// CampaignRequest creates a campaign.
type CampaignRequest struct {
	// Name labels the campaign, up to 120 characters. Empty uses the topics
	// and the send time.
	Name string
	// Topics are names or IDs, up to 50. Empty means DefaultTopic.
	Topics []string
	// Target limits the platforms. Empty means TargetAll.
	Target Target
	// ScheduledAt sends later instead of now.
	ScheduledAt *time.Time
	// Draft saves the campaign without sending it.
	Draft bool
	Content
}

type campaignWire struct {
	Name        string     `json:"name,omitempty"`
	Topics      []string   `json:"topics,omitempty"`
	Target      Target     `json:"target,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Draft       bool       `json:"draft,omitempty"`
	contentWire
}

// CreateCampaign creates a campaign and returns it. Links are validated
// strictly: Image, Icon and URL must be http or https URLs.
func (c *Client) CreateCampaign(ctx context.Context, r CampaignRequest) (*Campaign, error) {
	if len(r.Name) > 120 {
		return nil, invalidArg("name must be 120 characters or fewer")
	}
	if !validTarget(r.Target) {
		return nil, invalidArg("target must be ALL, WEB or ANDROID")
	}
	if err := validTopicList(r.Topics); err != nil {
		return nil, err
	}
	if err := r.Content.validateStrict(); err != nil {
		return nil, err
	}

	var cmp Campaign
	wire := campaignWire{
		Name: r.Name, Topics: r.Topics, Target: r.Target, ScheduledAt: r.ScheduledAt,
		Draft: r.Draft, contentWire: r.Content.wire(),
	}
	if err := c.do(ctx, request{method: http.MethodPost, path: c.api("/campaigns"), body: wire}, &cmp); err != nil {
		return nil, err
	}
	return &cmp, nil
}

// GetCampaign returns a campaign with live counters. ref is the campaign ID or
// its index; see IndexRef.
func (c *Client) GetCampaign(ctx context.Context, ref string) (*Campaign, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, invalidArg("campaign reference is required")
	}
	var cmp Campaign
	path := c.api("/campaigns/" + url.PathEscape(ref))
	if err := c.do(ctx, request{method: http.MethodGet, path: path}, &cmp); err != nil {
		return nil, err
	}
	return &cmp, nil
}

// CancelCampaign stops a scheduled or sending campaign and returns it.
// Notifications already handed to a push service cannot be recalled.
func (c *Client) CancelCampaign(ctx context.Context, ref string) (*Campaign, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, invalidArg("campaign reference is required")
	}
	var cmp Campaign
	path := c.api("/campaigns/" + url.PathEscape(ref) + "/cancel")
	if err := c.do(ctx, request{method: http.MethodPost, path: path}, &cmp); err != nil {
		return nil, err
	}
	return &cmp, nil
}

// WaitForCampaign polls a campaign every interval until it has completed,
// failed or been cancelled. Use a context with a deadline. When the context
// ends it returns the last campaign seen, which can be nil, with the context
// error. An interval below one second is raised to one second and zero means
// three seconds.
func (c *Client) WaitForCampaign(ctx context.Context, ref string, interval time.Duration) (*Campaign, error) {
	if interval == 0 {
		interval = defaultPollInterval
	}
	interval = max(interval, minPollInterval)

	var last *Campaign
	for {
		cmp, err := c.GetCampaign(ctx, ref)
		if err != nil {
			return last, err
		}
		last = cmp
		if cmp.Status.Finished() {
			return cmp, nil
		}

		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(interval):
		}
	}
}

// Cursor locates a page of results. Pass the one from the previous page.
type Cursor struct {
	CursorCreatedAt time.Time `json:"cursor_created_at"`
	CursorID        string    `json:"cursor_id"`
	Limit           int       `json:"limit"`
}

// ListCampaignsParams filters ListCampaigns.
type ListCampaignsParams struct {
	Status []CampaignStatus
	Source []Source
	// TopicID only returns campaigns sent to this topic.
	TopicID string
	// Search matches the name and message.
	Search string
	// Limit is the page size, 1 to 100. Zero means the service default of 10.
	Limit int
	// Cursor continues from a previous page.
	Cursor *Cursor
}

// CampaignPage is one page of campaigns, newest first. Next is nil on the
// last page.
type CampaignPage struct {
	Campaigns []Campaign `json:"campaigns"`
	Next      *Cursor    `json:"page"`
}

// ListCampaigns returns campaigns in the project, newest first.
func (c *Client) ListCampaigns(ctx context.Context, p ListCampaignsParams) (*CampaignPage, error) {
	if p.Limit < 0 || p.Limit > 100 {
		return nil, invalidArg("limit must be between 1 and 100")
	}

	q := url.Values{}
	if len(p.Status) > 0 {
		q.Set("status", joinList(p.Status))
	}
	if len(p.Source) > 0 {
		q.Set("source", joinList(p.Source))
	}
	if p.TopicID != "" {
		q.Set("topic_id", p.TopicID)
	}
	if p.Search != "" {
		q.Set("search", p.Search)
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Cursor != nil {
		q.Set("cursor_created_at", p.Cursor.CursorCreatedAt.Format(time.RFC3339Nano))
		q.Set("cursor_id", p.Cursor.CursorID)
	}

	var page CampaignPage
	if err := c.do(ctx, request{method: http.MethodGet, path: c.api("/campaigns"), query: q}, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

// EachCampaign calls fn for every matching campaign, newest first, following
// the cursor across pages. It stops at the first error from fn.
func (c *Client) EachCampaign(ctx context.Context, p ListCampaignsParams, fn func(Campaign) error) error {
	for {
		page, err := c.ListCampaigns(ctx, p)
		if err != nil {
			return err
		}
		for _, cmp := range page.Campaigns {
			if err := fn(cmp); err != nil {
				return err
			}
		}
		if page.Next == nil || len(page.Campaigns) == 0 {
			return nil
		}
		p.Cursor = page.Next
	}
}

func joinList[T ~string](items []T) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = string(it)
	}
	return strings.Join(parts, ",")
}
