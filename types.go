package notitiago

import "time"

// Target selects which platforms receive a notification.
type Target string

const (
	TargetAll     Target = "ALL"
	TargetWeb     Target = "WEB"
	TargetAndroid Target = "ANDROID"
)

// Urgency is the delivery priority. High, the default, wakes the device.
type Urgency string

const (
	UrgencyHigh    Urgency = "high"
	UrgencyNormal  Urgency = "normal"
	UrgencyLow     Urgency = "low"
	UrgencyVeryLow Urgency = "very-low"
)

// Source is how a notification or campaign was created.
type Source string

const (
	SourceAPI       Source = "api"
	SourceBroadcast Source = "broadcast"
	SourceDashboard Source = "dashboard"
)

// DefaultTopic holds every device, so sending to it reaches everyone.
const DefaultTopic = "default"

// NotificationStatus is the state of one notification.
type NotificationStatus string

const (
	NotificationQueued    NotificationStatus = "queued"
	NotificationSending   NotificationStatus = "sending"
	NotificationSent      NotificationStatus = "sent"
	NotificationFailed    NotificationStatus = "failed"
	NotificationCancelled NotificationStatus = "cancelled"
)

// CampaignStatus is the state of a campaign.
type CampaignStatus string

const (
	CampaignDraft     CampaignStatus = "draft"
	CampaignScheduled CampaignStatus = "scheduled"
	CampaignQueued    CampaignStatus = "queued"
	CampaignSending   CampaignStatus = "sending"
	CampaignCompleted CampaignStatus = "completed"
	CampaignFailed    CampaignStatus = "failed"
	CampaignCancelled CampaignStatus = "cancelled"
)

// Finished reports that a campaign has stopped for good.
func (s CampaignStatus) Finished() bool {
	return s == CampaignCompleted || s == CampaignFailed || s == CampaignCancelled
}

// DeliveryErrorCode explains why a notification failed.
type DeliveryErrorCode string

const (
	ErrCodeConfig               DeliveryErrorCode = "config_error"
	ErrCodeNoSubscription       DeliveryErrorCode = "no_subscription"
	ErrCodeUnauthenticated      DeliveryErrorCode = "unauthenticated"
	ErrCodeMismatchedCredential DeliveryErrorCode = "mismatched_credential"
	ErrCodeSenderIDMismatch     DeliveryErrorCode = "sender_id_mismatch"
	ErrCodeVAPIDMismatch        DeliveryErrorCode = "vapid_mismatch"
	ErrCodeUnregistered         DeliveryErrorCode = "unregistered"
	ErrCodeGone                 DeliveryErrorCode = "gone"
	ErrCodeInvalidSubscription  DeliveryErrorCode = "invalid_subscription"
	ErrCodeThirdPartyAuth       DeliveryErrorCode = "third_party_auth_error"
	ErrCodeInvalidArgument      DeliveryErrorCode = "invalid_argument"
	ErrCodePayloadTooLarge      DeliveryErrorCode = "payload_too_large"
	ErrCodeInvalidPayload       DeliveryErrorCode = "invalid_payload"
	ErrCodeRejected             DeliveryErrorCode = "rejected"
	ErrCodeQuotaExceeded        DeliveryErrorCode = "quota_exceeded"
	ErrCodeRateLimited          DeliveryErrorCode = "rate_limited"
	ErrCodeUnavailable          DeliveryErrorCode = "unavailable"
	ErrCodeInternal             DeliveryErrorCode = "internal"
	ErrCodeNetwork              DeliveryErrorCode = "network"
	ErrCodeProviderError        DeliveryErrorCode = "provider_error"
	ErrCodeUnknown              DeliveryErrorCode = "unknown"
)

// Temporary reports that the service retries this failure on its own.
func (c DeliveryErrorCode) Temporary() bool {
	switch c {
	case ErrCodeQuotaExceeded, ErrCodeRateLimited, ErrCodeUnavailable, ErrCodeInternal, ErrCodeNetwork, ErrCodeProviderError, ErrCodeUnknown:
		return true
	}
	return false
}

// RemovesDevice reports that the failure marks the device invalid, so it
// is skipped from then on.
func (c DeliveryErrorCode) RemovesDevice() bool {
	switch c {
	case ErrCodeMismatchedCredential, ErrCodeSenderIDMismatch, ErrCodeVAPIDMismatch, ErrCodeUnregistered, ErrCodeGone, ErrCodeInvalidSubscription:
		return true
	}
	return false
}

// Attempt is one try at handing a notification to Firebase or a browser push service.
type Attempt struct {
	ID                string            `json:"id"`
	AttemptNo         int               `json:"attempt_no"`
	Provider          string            `json:"provider"`
	Outcome           string            `json:"outcome"`
	ErrorCode         DeliveryErrorCode `json:"error_code"`
	ErrorMessage      string            `json:"error_message"`
	HTTPStatus        int               `json:"http_status"`
	ProviderMessageID string            `json:"provider_message_id"`
	LatencyMs         int               `json:"latency_ms"`
	CreatedAt         time.Time         `json:"created_at"`
}

// Event is a delivery report sent back by an app or service worker.
type Event struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Platform  string    `json:"platform"`
	UserAgent string    `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
}

// Notification is one notification sent to one device, with its delivery history.
// Text fields are empty and time fields nil when the service reports null.
type Notification struct {
	ID            string             `json:"id"`
	Index         int64              `json:"index"`
	Topic         string             `json:"topic"`
	Target        Target             `json:"target"`
	Project       string             `json:"project"`
	Title         string             `json:"title"`
	Body          string             `json:"body"`
	Image         string             `json:"image"`
	Icon          string             `json:"icon"`
	URL           string             `json:"url"`
	Route         string             `json:"route"`
	Tag           string             `json:"tag"`
	Data          map[string]string  `json:"data"`
	CampaignID    string             `json:"campaign_id"`
	CampaignName  string             `json:"campaign_name"`
	Identifier    string             `json:"identifier"`
	Platform      string             `json:"platform"`
	Source        Source             `json:"source"`
	Status        NotificationStatus `json:"status"`
	AttemptCount  int                `json:"attempt_count"`
	TTLSeconds    int                `json:"ttl_seconds"`
	Urgency       Urgency            `json:"urgency"`
	ErrorCode     DeliveryErrorCode  `json:"error_code"`
	ErrorMessage  string             `json:"error_message"`
	SentAt        *time.Time         `json:"sent_at"`
	DeliveredAt   *time.Time         `json:"delivered_at"`
	ClickedAt     *time.Time         `json:"clicked_at"`
	NextAttemptAt *time.Time         `json:"next_attempt_at"`
	FailedAt      *time.Time         `json:"failed_at"`
	CreatedAt     time.Time          `json:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at"`
	Attempts      []Attempt          `json:"attempts"`
	Events        []Event            `json:"events"`
}

// TopicRef names a topic a campaign was sent to.
type TopicRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CampaignStats are the live counters of a campaign. Sent means a push
// service accepted the notification; Delivered and Clicked are reported by apps.
type CampaignStats struct {
	Total        int64   `json:"total"`
	Sent         int64   `json:"sent"`
	Failed       int64   `json:"failed"`
	Delivered    int64   `json:"delivered"`
	Clicked      int64   `json:"clicked"`
	Pending      int64   `json:"pending"`
	Retryable    int64   `json:"retryable"`
	DeliveryRate float64 `json:"delivery_rate"`
	ClickRate    float64 `json:"click_rate"`
	FailureRate  float64 `json:"failure_rate"`
}

// Campaign is a notification sent to the people in one or more topics.
type Campaign struct {
	ID            string            `json:"id"`
	Index         int64             `json:"index"`
	ProjectID     string            `json:"project_id"`
	Name          string            `json:"name"`
	Source        Source            `json:"source"`
	Status        CampaignStatus    `json:"status"`
	Target        Target            `json:"target"`
	Title         string            `json:"title"`
	Body          string            `json:"body"`
	Image         string            `json:"image"`
	Icon          string            `json:"icon"`
	URL           string            `json:"url"`
	Route         string            `json:"route"`
	Tag           string            `json:"tag"`
	Data          map[string]string `json:"data"`
	TTLSeconds    int               `json:"ttl_seconds"`
	Urgency       Urgency           `json:"urgency"`
	Tracked       bool              `json:"tracked"`
	FailureReason string            `json:"failure_reason"`
	Topics        []TopicRef        `json:"topics"`
	Stats         CampaignStats     `json:"stats"`
	ScheduledAt   *time.Time        `json:"scheduled_at"`
	StartedAt     *time.Time        `json:"started_at"`
	CompletedAt   *time.Time        `json:"completed_at"`
	CancelledAt   *time.Time        `json:"cancelled_at"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
}

// Topic groups devices so they can be broadcast to.
type Topic struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Project         string    `json:"project"`
	Description     string    `json:"description"`
	AndroidCount    int64     `json:"android_count"`
	WebCount        int64     `json:"web_count"`
	SubscriberCount int64     `json:"subscriber_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
