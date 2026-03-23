> **STATUS: COMPLETED (v3 PRD)**
> This epic was completed under the v3 PRD structure. Its work is fully integrated
> into the v4 codebase. See [Completed Work Registry](./completed-work-registry.md)
> for the mapping to v4 feature IDs.

# Epic 4: qBittorrent Download Monitoring

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can connect to their qBittorrent instance and monitor downloads in real-time from a unified dashboard.

## Story 4.1: qBittorrent Connection Configuration

As a **NAS user**,
I want to **connect Vido to my qBittorrent instance**,
So that **I can monitor downloads from within Vido**.

**Acceptance Criteria:**

**Given** the user navigates to qBittorrent settings
**When** they enter:
- Host URL (e.g., `http://192.168.1.100:8080`)
- Username
- Password
**Then** credentials are encrypted before storage (FR51)
**And** credentials never appear in logs (NFR-I4)

**Given** credentials are entered
**When** the user clicks "Test Connection"
**Then** the system verifies connectivity within 10 seconds (NFR-I2)
**And** shows success or detailed error message

**Given** qBittorrent is behind a reverse proxy
**When** configuring the connection
**Then** custom base paths are supported (NFR-I3)
**And** HTTPS connections work properly

**Technical Notes:**
- Implements FR27, FR28: Connect and test qBittorrent
- Implements NFR-I1, NFR-I2, NFR-I3, NFR-I4
- Uses qBittorrent Web API v2.x

---

## Story 4.2: Real-Time Download Status Monitoring

As a **media collector**,
I want to **see real-time download status**,
So that **I can monitor progress without opening qBittorrent**.

**Acceptance Criteria:**

**Given** qBittorrent is connected
**When** viewing the downloads dashboard
**Then** all torrents are displayed with:
- Name
- Progress percentage
- Download/upload speed
- ETA
- Status (downloading, paused, seeding, completed)

**Given** a torrent is active
**When** 5 seconds pass (NFR-P8)
**Then** the status updates automatically
**And** the UI updates without full page refresh

**Given** polling is active
**When** the user navigates away from the downloads page
**Then** polling stops to conserve resources
**And** resumes when they return

**Technical Notes:**
- Implements FR29: Real-time download status
- Implements NFR-P8: Updates within 5 seconds
- Uses TanStack Query polling with refetchInterval

---

## Story 4.3: Unified Download Dashboard

As a **media collector**,
I want a **unified dashboard showing downloads and recent media**,
So that **I can see my complete workflow in one place**.

**Acceptance Criteria:**

**Given** the user opens the homepage
**When** the dashboard loads
**Then** they see:
- Left panel: qBittorrent download list
- Right panel: Recently added media
- Bottom: Quick search bar

**Given** downloads and media are displayed
**When** a download completes
**Then** the completed item moves from "Downloads" to "Recent Media" after parsing
**And** a notification indicates successful addition

**Given** qBittorrent is disconnected
**When** viewing the dashboard
**Then** the download panel shows connection status
**And** other panels remain functional (NFR-R12)

**Technical Notes:**
- Implements FR30: View download list in unified dashboard
- Implements NFR-R12: Partial functionality when disconnected
- Desktop multi-column layout (UX-1)

---

## Story 4.4: Download Status Filtering

As a **media collector**,
I want to **filter downloads by status**,
So that **I can focus on specific download states**.

**Acceptance Criteria:**

**Given** the download list is displayed
**When** filter buttons are shown
**Then** options include: All, Downloading, Paused, Completed, Seeding

**Given** the user selects "Downloading" filter
**When** the filter is applied
**Then** only actively downloading torrents are shown
**And** the count updates in the filter button

**Given** filters are applied
**When** the list updates (polling)
**Then** new items matching the filter appear
**And** items no longer matching disappear

**Technical Notes:**
- Implements FR31: Filter downloads by status
- Client-side filtering for responsiveness
- Filter state persisted in URL for bookmarking

---

## Story 4.5: Completed Download Detection and Parsing Trigger

As a **media collector**,
I want **completed downloads to automatically trigger parsing**,
So that **new media appears in my library without manual action**.

**Acceptance Criteria:**

**Given** qBittorrent reports a torrent as complete
**When** the next polling cycle detects the completion
**Then** the system automatically queues the file for parsing
**And** the download shows status: "Parsing..."

**Given** parsing completes successfully
**When** metadata is retrieved
**Then** the media appears in "Recently Added"
**And** a success notification is shown

**Given** parsing fails
**When** errors occur
**Then** the download shows: "Parsing failed - Manual action needed"
**And** links to manual search options

**Technical Notes:**
- Implements FR32: Detect completed downloads and trigger parsing
- Integrates with Epic 3 parsing pipeline
- Non-blocking: user can continue browsing

---

## Story 4.6: Connection Health Monitoring

As a **system administrator**,
I want to **see qBittorrent connection health status**,
So that **I know immediately when there are connectivity issues**.

**Acceptance Criteria:**

**Given** qBittorrent is connected
**When** viewing the dashboard header
**Then** a status indicator shows: 🟢 Connected

**Given** qBittorrent becomes unreachable
**When** the health check fails
**Then** the indicator changes to: 🔴 Disconnected
**And** shows: "Last success: 2 minutes ago"

**Given** connection is lost
**When** automatic recovery is attempted
**Then** the system retries every 30 seconds (NFR-R6)
**And** reconnects automatically when available

**Given** the user clicks on the connection status
**When** viewing details
**Then** they see connection history and error logs

**Technical Notes:**
- Implements FR33: Display connection health status
- Implements NFR-R6: Auto-recover (30s reconnection)
- Implements ARCH-8: Health Check Scheduler

---
