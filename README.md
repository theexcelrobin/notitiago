# notitiago

[![Go Reference](https://pkg.go.dev/badge/github.com/theexcelrobin/notitiago.svg)](https://pkg.go.dev/github.com/theexcelrobin/notitiago)
[![CI](https://github.com/theexcelrobin/notitiago/actions/workflows/ci.yml/badge.svg)](https://github.com/theexcelrobin/notitiago/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

A Go client for Loveworld Notifications (notitia), the push notification
service at [api.lwnotifications.com](https://api.lwnotifications.com).
Standard library only, Go 1.22 or newer.

| Area | Methods |
|---|---|
| Devices | `Subscribe`, `Unsubscribe`, `WebKey` |
| Sending | `Send`, `Broadcast`, `GetNotification`, `UpdateNotification`, `DeleteNotification` |
| Campaigns | `CreateCampaign`, `GetCampaign`, `CancelCampaign`, `WaitForCampaign`, `ListCampaigns`, `EachCampaign` |
| Topics | `CreateTopic`, `DeleteTopic` |
| Files | `UploadFile` |
| Tracking | `Track`, `ClickURL` |

## Install

```sh
go get github.com/theexcelrobin/notitiago
```

## Quick start

```go
client := notitiago.New(notitiago.WithAPIKey(os.Getenv("NOTITIA_API_KEY")))

index, err := client.Send(ctx, notitiago.SendRequest{
	Identifier: "user-1042",
	Content: notitiago.Content{
		Title: "Your order has shipped",
		Body:  "It should arrive on Thursday.",
		Route: "/orders/1042",
	},
})
```

**Keep your API key on your server.** It can send to every subscriber of the
project and subscribe devices as any of your users. Never put it in a browser or
a mobile app. Your backend subscribes devices on behalf of signed-in users and
hands the web push key to the browser with `WebKey`.

## Devices and topics

```go
identifier, err := client.Subscribe(ctx, notitiago.SubscribeRequest{
	Target:     notitiago.TargetWeb,
	Identifier: user.ID, // from your session, not from the request
	Topics:     []string{"youth_church"},
	Web:        &notitiago.WebDevice{Endpoint: sub.Endpoint, Keys: notitiago.WebKeys{P256DH: sub.P256DH, Auth: sub.Auth}},
})
```

`WebDevice` decodes straight from the JSON of the browser's `PushSubscription`.
Every device is also added to `notitiago.DefaultTopic`, which is what "everyone"
means. In each topic a person has one device per platform, so a newer phone or
browser replaces the older one. Use `Recreate: true` to create topics on the
fly, or `CreateTopic` to add them ahead of time.

## Sending

`Send` reaches one person's devices. `Broadcast` and `CreateCampaign` reach
everyone in one or more topics, and people in several topics get it once.
`Content` is shared by all three: title, body, image, icon, URL, route, tag, data,
TTL and urgency.

```go
index, err := client.Broadcast(ctx, notitiago.BroadcastRequest{
	Topics:  []string{"youth_church"},
	Content: notitiago.Content{Title: "Service starts in 30 minutes", URL: "https://example.org/live"},
})
```

Sends are queued and delivered in the background with retries, so a successful
call means accepted, not shown. Follow the result:

```go
n, err := client.GetNotification(ctx, notitiago.IndexRef(index))
// n.Status, n.ErrorCode, n.Attempts, n.Events

cmp, err := client.WaitForCampaign(ctx, notitiago.IndexRef(campaignIndex), 5*time.Second)
// cmp.Stats.Sent, cmp.Stats.Failed, cmp.Stats.Delivered, cmp.Stats.Clicked
```

`Send` still returns success when no device matches. The notification is then
stored as failed with `ErrCodeNoSubscription`, so check the result when it matters.

Use `Ptr` for fields where zero is meaningful: `TTL: notitiago.Ptr(time.Hour)`,
or `UpdateRequest{Title: notitiago.Ptr("New title")}`. `TTL` can be up to 28 days.
Data keys starting with `lwn_` are reserved by the service.

Delivery failures carry a `DeliveryErrorCode`. `Temporary()` says the service
will retry it, and `RemovesDevice()` says the device was marked invalid.

## Files

```go
url, err := client.UploadFile(ctx, "cover.png", file) // up to 20 MB
```

The URL is public and can be used as a notification's `Image` or `Icon`.
Uploading the same file twice returns the same URL.

## Tracking

Every notification delivered to a device carries `lwn_id` and `lwn_sig` in its
data. Report back when it is shown or opened:

```go
client.Track(ctx, notitiago.TrackDelivered, id, sig)
client.Track(ctx, notitiago.TrackClicked, id, sig)
```

No API key is needed. `ClickURL(id, sig)` gives a link that records a click and
redirects to the notification's URL, for email or SMS.

## Errors

```go
_, err := client.Send(ctx, req)

if errors.Is(err, notitiago.ErrUnauthorized) { ... }

var apiErr *notitiago.APIError
if errors.As(err, &apiErr) {
	log.Println(apiErr.Status, apiErr.Message, apiErr.RetryAfter)
}
```

The service answers 429 above 6,000 requests a minute per project, with a
different body than its other errors. The client reads both. `WithMaxRetries(n)`
retries 429 responses only, waiting as long as asked. Sends are never retried
on other failures, so a notification cannot go out twice. Prefer `Broadcast` over
looping `Send`: one request reaches any number of devices.

## Notes

- `GetNotification` and `GetCampaign` take an ID or an index. `UpdateNotification`
  and `DeleteNotification` take the ID.
- Text fields are empty and time fields are nil where the service sends null.
- There are no webhooks. The service reports deliveries through the tracking calls.

The package is `v0.x`: the API can change between minor versions.
See the [changelog](CHANGELOG.md).

## License

[MIT](LICENSE)
