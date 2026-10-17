package notitiago

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var topicName = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)

type createTopicWire struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateTopic adds a topic people can be subscribed to. The name is 3 to 20
// letters, digits or underscores and must be unique in the project.
func (c *Client) CreateTopic(ctx context.Context, name, description string) (*Topic, error) {
	if !topicName.MatchString(name) {
		return nil, invalidArg("topic name must be 3 to 20 letters, digits or underscores")
	}
	var t Topic
	err := c.do(ctx, request{method: http.MethodPost, path: c.api("/create-topic"), body: createTopicWire{Name: name, Description: description}}, &t)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// DeleteTopic removes a topic by ID. Devices stay subscribed to their other
// topics, and DefaultTopic cannot be deleted.
func (c *Client) DeleteTopic(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return invalidArg("topic id is required")
	}
	return c.do(ctx, request{method: http.MethodDelete, path: c.api("/delete-topic/" + url.PathEscape(id))}, nil)
}
