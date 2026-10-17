package notitiago_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/theexcelrobin/notitiago"
)

func newClient() *notitiago.Client {
	return notitiago.New(notitiago.WithAPIKey(os.Getenv("NOTITIA_API_KEY")))
}

func ExampleClient_Subscribe() {
	identifier, err := newClient().Subscribe(context.Background(), notitiago.SubscribeRequest{
		Target:     notitiago.TargetWeb,
		Identifier: "user-1042",
		Topics:     []string{"youth_church", "choir"},
		Web: &notitiago.WebDevice{
			Endpoint: "https://fcm.googleapis.com/fcm/send/...",
			Keys:     notitiago.WebKeys{P256DH: "BNcRdre...", Auth: "tBHItJI..."},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("subscribed", identifier)
}

func ExampleClient_Send() {
	client := newClient()
	ctx := context.Background()

	index, err := client.Send(ctx, notitiago.SendRequest{
		Identifier: "user-1042",
		Content: notitiago.Content{
			Title: "Your order has shipped",
			Body:  "It should arrive on Thursday.",
			Route: "/orders/1042",
			Tag:   "order-1042",
			Data:  map[string]string{"order_id": "1042"},
			TTL:   notitiago.Ptr(24 * time.Hour),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	n, err := client.GetNotification(ctx, notitiago.IndexRef(index))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n.Status, n.ErrorCode)
}

func ExampleClient_Broadcast() {
	when := time.Now().Add(time.Hour)

	index, err := newClient().Broadcast(context.Background(), notitiago.BroadcastRequest{
		Topics:      []string{"youth_church"},
		ScheduledAt: &when,
		Content:     notitiago.Content{Title: "Service starts in 30 minutes", URL: "https://example.org/live"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("campaign", index)
}

func ExampleClient_CreateCampaign() {
	client := newClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmp, err := client.CreateCampaign(ctx, notitiago.CampaignRequest{
		Name:    "Choir rehearsal moved",
		Topics:  []string{"choir"},
		Content: notitiago.Content{Title: "Rehearsal moved to Saturday", Body: "Same time, main auditorium."},
	})
	if err != nil {
		log.Fatal(err)
	}

	done, err := client.WaitForCampaign(ctx, notitiago.IndexRef(cmp.Index), 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s: %d sent, %d failed\n", done.Status, done.Stats.Sent, done.Stats.Failed)
}

func ExampleClient_EachCampaign() {
	err := newClient().EachCampaign(context.Background(),
		notitiago.ListCampaignsParams{Status: []notitiago.CampaignStatus{notitiago.CampaignCompleted}, Limit: 50},
		func(c notitiago.Campaign) error {
			fmt.Println(c.Name, c.Stats.ClickRate)
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_UploadFile() {
	client := newClient()
	ctx := context.Background()

	f, err := os.Open("cover.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	url, err := client.UploadFile(ctx, "cover.png", f)
	if err != nil {
		log.Fatal(err)
	}
	_, err = client.Broadcast(ctx, notitiago.BroadcastRequest{Content: notitiago.Content{Title: "New album", Image: url}})
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleClient_Track() {
	recorded, err := newClient().Track(context.Background(), notitiago.TrackClicked, "lwn_id from the payload", "lwn_sig from the payload")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("first report:", recorded)
}

func ExampleClient_ClickURL() {
	client := notitiago.New(notitiago.WithBaseURL("https://api.lwnotifications.com"))
	fmt.Println(client.ClickURL("01a0fc69", "q8KpX2c1"))
	// Output: https://api.lwnotifications.com/t/01a0fc69/q8KpX2c1
}

func ExampleAPIError() {
	_, err := newClient().Send(context.Background(), notitiago.SendRequest{
		Identifier: "user-1042",
		Content:    notitiago.Content{Title: "Hello"},
	})

	var apiErr *notitiago.APIError
	switch {
	case err == nil:
	case errors.Is(err, notitiago.ErrUnauthorized):
		fmt.Println("check your API key")
	case errors.Is(err, notitiago.ErrRateLimited):
		errors.As(err, &apiErr)
		fmt.Println("retry in", apiErr.RetryAfter)
	case errors.As(err, &apiErr):
		fmt.Println(apiErr.Status, apiErr.Message)
	default:
		log.Fatal(err)
	}
}

func ExampleNew() {
	_ = notitiago.New(
		notitiago.WithAPIKey(os.Getenv("NOTITIA_API_KEY")),
		notitiago.WithTimeout(10*time.Second),
		notitiago.WithMaxRetries(2),
	)
}
