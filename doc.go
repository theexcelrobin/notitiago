// Package notitiago is a Go client for the Loveworld Notifications (notitia)
// developer API.
//
// It manages devices and topics, sends push notifications to Android and web
// devices, runs campaigns, uploads images and reports deliveries and clicks. It
// uses only the standard library.
//
//	client := notitiago.New(notitiago.WithAPIKey(os.Getenv("NOTITIA_API_KEY")))
//
//	index, err := client.Send(ctx, notitiago.SendRequest{
//		Identifier: "user-1042",
//		Content:    notitiago.Content{Title: "Your order has shipped", Route: "/orders/1042"},
//	})
//
// Sends are queued and delivered in the background, so a successful call means
// the notification was accepted, not shown. Look it up with GetNotification, or
// follow a campaign with WaitForCampaign.
//
// Your API key can send to every subscriber of the project. Use this package
// from your server only, never from a browser or mobile app.
//
// Failed calls return an *APIError. Use errors.Is with the Err* sentinels, or
// errors.As to read the status code and message.
package notitiago
