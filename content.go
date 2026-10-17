package notitiago

import (
	"net/url"
	"strings"
	"time"
)

const (
	maxTTL         = 2419200 * time.Second
	reservedPrefix = "lwn_"
	maxTopics      = 50
)

// Content is what a notification shows and carries. It is shared by Send,
// Broadcast and CreateCampaign.
type Content struct {
	// Title is the first line people see. Keep it under about 65 characters.
	Title string
	// Body is the message under the title.
	Body string
	// Image is a large picture under the message, as an https URL.
	Image string
	// Icon replaces the site icon in browser notifications.
	Icon string
	// URL is opened when a browser notification is clicked. Android apps get
	// it as data.url.
	URL string
	// Route is a screen in your app. Android apps get it as data.route.
	Route string
	// Tag makes notifications with the same tag replace each other.
	Tag string
	// Data is extra key-value pairs for your app. Keys starting with
	// lwn_ are reserved.
	Data map[string]string
	// TTL is how long the push service keeps trying an offline device, up to
	// 28 days. Nil uses the service default of 3 days.
	TTL *time.Duration
	// Urgency is the delivery priority. Empty means high.
	Urgency Urgency
}

type contentWire struct {
	Title   string            `json:"title"`
	Body    string            `json:"body,omitempty"`
	Image   string            `json:"image,omitempty"`
	Icon    string            `json:"icon,omitempty"`
	URL     string            `json:"url,omitempty"`
	Route   string            `json:"route,omitempty"`
	Tag     string            `json:"tag,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
	TTL     *int64            `json:"ttl,omitempty"`
	Urgency Urgency           `json:"urgency,omitempty"`
}

func (c Content) wire() contentWire {
	w := contentWire{
		Title: c.Title, Body: c.Body, Image: c.Image, Icon: c.Icon, URL: c.URL,
		Route: c.Route, Tag: c.Tag, Data: c.Data, Urgency: c.Urgency,
	}
	if c.TTL != nil {
		secs := int64(*c.TTL / time.Second)
		w.TTL = &secs
	}
	return w
}

func (c Content) validate() error {
	if strings.TrimSpace(c.Title) == "" {
		return invalidArg("title is required")
	}
	switch c.Urgency {
	case "", UrgencyHigh, UrgencyNormal, UrgencyLow, UrgencyVeryLow:
	default:
		return invalidArg("urgency must be high, normal, low or very-low")
	}
	if c.TTL != nil && (*c.TTL < 0 || *c.TTL > maxTTL) {
		return invalidArg("ttl must be between 0 and 28 days")
	}
	for k := range c.Data {
		if strings.HasPrefix(k, reservedPrefix) {
			return invalidArg("data key %q uses the reserved %s prefix", k, reservedPrefix)
		}
	}
	return nil
}

func (c Content) validateStrict() error {
	if err := c.validate(); err != nil {
		return err
	}
	switch {
	case len(c.Title) > 240:
		return invalidArg("title must be 240 characters or fewer")
	case len(c.Body) > 4000:
		return invalidArg("body must be 4000 characters or fewer")
	case len(c.Route) > 500:
		return invalidArg("route must be 500 characters or fewer")
	case len(c.Tag) > 120:
		return invalidArg("tag must be 120 characters or fewer")
	}
	for name, v := range map[string]string{"image": c.Image, "icon": c.Icon, "url": c.URL} {
		if v != "" && !validHTTPURL(v) {
			return invalidArg("%s must be an http or https URL", name)
		}
	}
	return nil
}

func validTarget(t Target) bool {
	return t == "" || t == TargetAll || t == TargetWeb || t == TargetAndroid
}

func validHTTPURL(s string) bool {
	u, err := url.Parse(s)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func validTopicList(topics []string) error {
	if len(topics) > maxTopics {
		return invalidArg("at most %d topics are allowed", maxTopics)
	}
	for _, t := range topics {
		if strings.TrimSpace(t) == "" {
			return invalidArg("topic names must not be empty")
		}
	}
	return nil
}
