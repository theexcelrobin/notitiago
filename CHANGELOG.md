# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and the project uses
[Semantic Versioning](https://semver.org/). Until v1.0.0 the API can change
between minor versions.

## [0.1.0] - 2026-10-17

### Added
- Devices: `Subscribe`, `Unsubscribe`, `WebKey`.
- Sending: `Send`, `Broadcast`, `GetNotification`, `UpdateNotification`, `DeleteNotification`.
- Campaigns: `CreateCampaign`, `GetCampaign`, `CancelCampaign`, `WaitForCampaign`,
  `ListCampaigns`, `EachCampaign`.
- Topics: `CreateTopic`, `DeleteTopic`.
- Files: `UploadFile`.
- Tracking: `Track`, `ClickURL`.
- Typed `*APIError` with `errors.Is` sentinels, reading both the standard error
  envelope and the rate limit body.
- Client options for API key, base URL, HTTP client, timeout and 429 retries.
