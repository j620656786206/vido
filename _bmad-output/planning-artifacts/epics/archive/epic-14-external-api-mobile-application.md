# Epic 14: External API & Mobile Application

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** The system provides a versioned RESTful API (/api/v1) with OpenAPI/Swagger documentation, supports webhook subscriptions, enables Plex/Jellyfin integration, and users can access via mobile application.

## Story 14.1: Versioned RESTful API

As a **developer**,
I want a **versioned RESTful API**,
So that **I can build integrations that won't break on updates**.

**Acceptance Criteria:**

**Given** the API is accessed
**When** making requests
**Then** endpoints follow pattern: `/api/v1/{resource}`
**And** standard HTTP methods are used (GET, POST, PUT, DELETE)

**Given** API versioning
**When** a breaking change is needed
**Then** new version `/api/v2/` is created
**And** v1 remains available with deprecation notice

**Given** API response format
**When** receiving data
**Then** responses are JSON with consistent structure:
```json
{
  "data": {...},
  "meta": {"page": 1, "total": 100},
  "errors": []
}
```

**Given** pagination is needed
**When** listing resources
**Then** `?page=1&limit=20` parameters are supported
**And** Link headers provide next/prev URLs

**Technical Notes:**
- Implements FR87: RESTful API (/api/v1)
- Implements ARCH-9: API Versioning Strategy
- Implements NFR-I16: Versioned API

---

## Story 14.2: API Token Authentication

As a **developer**,
I want to **authenticate with API tokens**,
So that **my integrations can access the API securely**.

**Acceptance Criteria:**

**Given** the user opens Settings > API Tokens
**When** clicking "Generate Token"
**Then** a new token is created with:
- Name (user-provided)
- Permissions (read-only, read-write)
- Expiration (optional)

**Given** an API token is generated
**When** using it in requests
**Then** Authorization header: `Bearer {token}`
**And** request is authenticated

**Given** token permissions are set
**When** a read-only token attempts write
**Then** request is rejected with 403 Forbidden
**And** error explains insufficient permissions

**Given** a token is compromised
**When** the user revokes it
**Then** token immediately stops working
**And** cannot be reactivated

**Technical Notes:**
- Implements FR88: Authenticate API with tokens
- Implements NFR-S11: API endpoints protected
- Token stored as hash, never plaintext

---

## Story 14.3: OpenAPI/Swagger Documentation

As a **developer**,
I want **interactive API documentation**,
So that **I can explore and test the API easily**.

**Acceptance Criteria:**

**Given** the developer accesses `/api/docs`
**When** the page loads
**Then** Swagger UI displays all endpoints
**And** each endpoint shows: Method, Path, Description, Parameters

**Given** an endpoint is viewed
**When** clicking "Try it out"
**Then** interactive testing is available
**And** responses are shown in real-time

**Given** the OpenAPI spec is needed
**When** accessing `/api/v1/openapi.json`
**Then** full OpenAPI 3.0 spec is returned
**And** can be imported into Postman/Insomnia

**Given** authentication is required
**When** using Swagger UI
**Then** "Authorize" button accepts API token
**And** subsequent requests include the token

**Technical Notes:**
- Implements FR89: OpenAPI/Swagger documentation
- Implements NFR-I17: OpenAPI/Swagger spec
- Auto-generated from code annotations

---

## Story 14.4: Webhook Subscriptions

As a **developer**,
I want to **subscribe to webhooks**,
So that **my systems are notified of events**.

**Acceptance Criteria:**

**Given** the user opens Settings > Webhooks
**When** creating a new webhook
**Then** they configure:
- URL to call
- Events to subscribe (new_media, parse_complete, download_complete)
- Secret for signature verification

**Given** a subscribed event occurs
**When** the system sends the webhook
**Then** POST request to configured URL
**And** payload includes event type and data
**And** `X-Vido-Signature` header for verification

**Given** webhook delivery fails
**When** target URL returns error
**Then** retry with exponential backoff (3 attempts)
**And** failure is logged in webhook history

**Given** webhook history is needed
**When** viewing webhook details
**Then** last 100 deliveries are shown
**And** each shows: timestamp, status, response

**Technical Notes:**
- Implements FR90: Webhook subscriptions
- Implements NFR-I18: Webhook support for events
- HMAC-SHA256 signature verification

---

## Story 14.5: Plex/Jellyfin Metadata Export

As a **media center user**,
I want to **export metadata to Plex/Jellyfin**,
So that **my media center shows correct information**.

**Acceptance Criteria:**

**Given** the user opens Settings > Integrations
**When** configuring Plex/Jellyfin export
**Then** they select export format: NFO (Kodi/Plex), Jellyfin API

**Given** NFO export is configured
**When** exporting media
**Then** NFO files are created alongside media files
**And** format is Kodi-compatible XML

**Given** the user clicks "Export All"
**When** export runs
**Then** all library items get NFO files
**And** existing NFO files can be overwritten or skipped

**Given** auto-export is enabled
**When** new media is added
**Then** NFO is automatically created
**And** Plex/Jellyfin can scan and import

**Technical Notes:**
- Implements FR91: Export metadata to Plex/Jellyfin
- Implements FR62: Export as NFO files
- NFO format: tvshow.nfo, movie.nfo

---

## Story 14.6: Watch Status Sync with Plex/Jellyfin

As a **media center user**,
I want **watch status synced with Plex/Jellyfin**,
So that **progress is consistent across platforms**.

**Acceptance Criteria:**

**Given** the user configures Plex integration
**When** entering Plex server details
**Then** connection is tested
**And** library matching is configured

**Given** sync is enabled
**When** user marks watched in Vido
**Then** Plex/Jellyfin is updated (if online)
**And** vice versa: Plex changes sync to Vido

**Given** conflict occurs
**When** both systems changed
**Then** user is prompted to resolve
**And** options: "Use Vido", "Use Plex", "Keep both"

**Given** sync fails
**When** Plex/Jellyfin is unreachable
**Then** changes are queued
**And** synced when connection restored

**Technical Notes:**
- Implements FR92: Sync watch status with Plex/Jellyfin
- Plex API and Jellyfin API clients
- Conflict resolution UI

---

## Story 14.7: Mobile Application Core

As a **mobile user**,
I want to **access Vido from my phone**,
So that **I can manage my library on the go**.

**Acceptance Criteria:**

**Given** the mobile app is installed
**When** the user opens the app
**Then** they connect to their Vido server URL
**And** authenticate with password/PIN

**Given** the user is authenticated
**When** using the mobile app
**Then** they can:
- Browse library
- View media details
- See download status
- Search for new content

**Given** mobile-optimized UI
**When** viewing content
**Then** interface adapts to phone screen
**And** touch gestures are supported

**Given** the server is unreachable
**When** the app loses connection
**Then** offline cached data is available
**And** message indicates limited mode

**Technical Notes:**
- Implements FR93: Mobile application access
- Implements UX-2: Mobile simplified monitoring
- React Native or Flutter (PWA as MVP)

---

## Story 14.8: Remote Download Control from Mobile

As a **mobile user**,
I want to **control downloads remotely**,
So that **I can manage my NAS when away from home**.

**Acceptance Criteria:**

**Given** the mobile app is connected
**When** viewing Downloads section
**Then** current download status is shown
**And** refreshes every 10 seconds

**Given** downloads are listed
**When** tapping a download
**Then** control options appear: Pause, Resume, Delete
**And** actions are sent to server

**Given** the user wants to add downloads
**When** searching from mobile
**Then** they can initiate new downloads
**And** downloads start on the NAS

**Given** notifications are enabled
**When** a download completes
**Then** push notification is sent
**And** tapping opens the completed item

**Technical Notes:**
- Implements FR94: Remote download control from mobile
- Push notifications via Firebase/APNs
- Requires VPN or exposed server for remote access
