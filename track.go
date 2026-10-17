package notitiago

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

// TrackEvent is what happened to a notification on a device.
type TrackEvent string

const (
	TrackDelivered TrackEvent = "delivered"
	TrackClicked   TrackEvent = "clicked"
	TrackDismissed TrackEvent = "dismissed"
)

type trackWire struct {
	ID  string `json:"id"`
	Sig string `json:"sig"`
}

type trackResult struct {
	Recorded bool `json:"recorded"`
}

// Track reports that a notification was delivered, clicked or dismissed. id
// and sig are the lwn_id and lwn_sig values every notification carries in its
// data. No API key is needed, as the signature proves the report is genuine,
// and the endpoint is limited to 1200 requests a minute per IP address.
//
// A click also counts as delivered. It returns false when the event was
// already recorded, which is harmless.
func (c *Client) Track(ctx context.Context, event TrackEvent, id, sig string) (bool, error) {
	switch event {
	case TrackDelivered, TrackClicked, TrackDismissed:
	default:
		return false, invalidArg("event must be delivered, clicked or dismissed")
	}
	if strings.TrimSpace(id) == "" || strings.TrimSpace(sig) == "" {
		return false, invalidArg("id and sig are required")
	}

	var res trackResult
	err := c.do(ctx, request{
		method: http.MethodPost,
		path:   "/public/v1/track/" + url.PathEscape(string(event)),
		body:   trackWire{ID: id, Sig: sig},
		public: true,
	}, &res)
	if err != nil {
		return false, err
	}
	return res.Recorded, nil
}

// ClickURL is the click-through link for a notification. Opening it records a
// click and redirects to the notification's URL, which suits links in email
// or SMS where you cannot run code.
func (c *Client) ClickURL(id, sig string) string {
	return c.baseURL + "/t/" + url.PathEscape(id) + "/" + url.PathEscape(sig)
}
