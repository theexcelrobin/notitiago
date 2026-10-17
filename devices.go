package notitiago

import (
	"context"
	"net/http"
	"strings"
)

// AndroidDevice is an Android registration token from FirebaseMessaging.getToken().
type AndroidDevice struct {
	FCMToken string `json:"fcm_token"`
}

// WebKeys are the encryption keys of a browser push subscription.
type WebKeys struct {
	P256DH string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// WebDevice is a browser push subscription. It decodes straight from the JSON
// of PushSubscription.toJSON().
type WebDevice struct {
	Endpoint string  `json:"endpoint"`
	Keys     WebKeys `json:"keys"`
}

// SubscribeRequest registers a device under an identifier.
type SubscribeRequest struct {
	// Target is TargetAndroid or TargetWeb.
	Target Target
	// Identifier is who the device belongs to, usually your user ID. Leave it
	// empty to get a random one back, and store it.
	Identifier string
	// Topics to join, up to 50. DefaultTopic is always added.
	Topics []string
	// Android is required when Target is TargetAndroid.
	Android *AndroidDevice
	// Web is required when Target is TargetWeb.
	Web *WebDevice
	// Recreate creates topics that do not exist yet instead of failing.
	Recreate bool
}

type subscribeWire struct {
	Target     Target         `json:"target"`
	Identifier string         `json:"identifier,omitempty"`
	Topics     []string       `json:"topics,omitempty"`
	Android    *AndroidDevice `json:"android,omitempty"`
	Web        *WebDevice     `json:"web,omitempty"`
	Recreate   *bool          `json:"recreate,omitempty"`
}

// Subscribe registers or updates a device and returns its identifier.
// Calling it again for the same device updates it. In each topic a person has
// one device per platform, so a newer device replaces the older one.
func (c *Client) Subscribe(ctx context.Context, r SubscribeRequest) (string, error) {
	if err := validDeviceTarget(r.Target); err != nil {
		return "", err
	}
	if err := validTopicList(r.Topics); err != nil {
		return "", err
	}
	switch r.Target {
	case TargetAndroid:
		if r.Android == nil || strings.TrimSpace(r.Android.FCMToken) == "" {
			return "", invalidArg("android fcm token is required")
		}
	case TargetWeb:
		if r.Web == nil || strings.TrimSpace(r.Web.Endpoint) == "" || r.Web.Keys.P256DH == "" || r.Web.Keys.Auth == "" {
			return "", invalidArg("web endpoint and keys are required")
		}
	}

	wire := subscribeWire{Target: r.Target, Identifier: r.Identifier, Topics: r.Topics}
	if r.Target == TargetAndroid {
		wire.Android = r.Android
	} else {
		wire.Web = r.Web
	}
	if r.Recreate {
		wire.Recreate = Ptr(true)
	}

	var identifier string
	err := c.do(ctx, request{method: http.MethodPost, path: c.api("/subscribe"), body: wire}, &identifier)
	if err != nil {
		return "", err
	}
	return identifier, nil
}

// UnsubscribeRequest removes a person's device from topics.
type UnsubscribeRequest struct {
	Target     Target
	Identifier string
	// Topics to leave, up to 50. Empty leaves DefaultTopic only, which takes
	// the device out of everyone campaigns. A device left in no topics stops
	// receiving anything.
	Topics []string
}

type unsubscribeWire struct {
	Target     Target   `json:"target"`
	Identifier string   `json:"identifier"`
	Topics     []string `json:"topics,omitempty"`
}

// Unsubscribe removes a device from topics and returns its identifier.
func (c *Client) Unsubscribe(ctx context.Context, r UnsubscribeRequest) (string, error) {
	if err := validDeviceTarget(r.Target); err != nil {
		return "", err
	}
	if strings.TrimSpace(r.Identifier) == "" {
		return "", invalidArg("identifier is required")
	}
	if err := validTopicList(r.Topics); err != nil {
		return "", err
	}

	var identifier string
	wire := unsubscribeWire(r)
	err := c.do(ctx, request{method: http.MethodPost, path: c.api("/unsubscribe"), body: wire}, &identifier)
	if err != nil {
		return "", err
	}
	return identifier, nil
}

// WebKey returns the project's public web push key, which browsers pass to
// pushManager.subscribe() as applicationServerKey. Fetch it on your server and
// hand it to the browser.
func (c *Client) WebKey(ctx context.Context) (string, error) {
	var key string
	if err := c.do(ctx, request{method: http.MethodGet, path: c.api("/web-key")}, &key); err != nil {
		return "", err
	}
	return key, nil
}

func validDeviceTarget(t Target) error {
	if t != TargetAndroid && t != TargetWeb {
		return invalidArg("target must be ANDROID or WEB")
	}
	return nil
}
