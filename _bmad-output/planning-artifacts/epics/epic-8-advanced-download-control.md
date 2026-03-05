# Epic 8: Advanced Download Control

**Phase:** Growth (Q3+ - September 2026+)

**Goal:** Users can control qBittorrent directly from Vido, including pause/resume/delete torrents, adjust download priority, manage bandwidth settings, and schedule downloads.

## Story 8.1: Torrent Control Operations

As a **media collector**,
I want to **control my torrents directly from Vido**,
So that **I don't need to switch to the qBittorrent interface for basic operations**.

**Acceptance Criteria:**

**Given** the user views the download list
**When** they select a torrent
**Then** control buttons are available: Pause, Resume, Delete

**Given** a running torrent
**When** the user clicks "Pause"
**Then** the torrent pauses immediately via qBittorrent API
**And** status updates within 2 seconds

**Given** a paused torrent
**When** the user clicks "Resume"
**Then** the torrent resumes
**And** status shows "Downloading"

**Given** the user clicks "Delete"
**When** confirming the action
**Then** a dialog asks: "Delete torrent only" or "Delete with files"
**And** the selected action is executed
**And** confirmation shows success

**Technical Notes:**
- Implements FR34: Control qBittorrent directly
- Uses qBittorrent Web API v2.x
- Requires confirmed connection from Epic 4

---

## Story 8.2: Download Priority Management

As a **media collector**,
I want to **adjust download priority**,
So that **important downloads complete first**.

**Acceptance Criteria:**

**Given** multiple torrents are downloading
**When** the user views the download list
**Then** each torrent shows its current priority level

**Given** a torrent is selected
**When** the user clicks "Set Priority"
**Then** options are available: High, Normal, Low
**And** the change is applied immediately

**Given** priority is changed
**When** qBittorrent processes the change
**Then** bandwidth allocation adjusts accordingly
**And** higher priority torrents get more bandwidth

**Given** file priority within a torrent
**When** the user expands torrent details
**Then** individual files can be set to High/Normal/Low/Skip
**And** Skip means the file won't download

**Technical Notes:**
- Implements FR35: Adjust download priority
- Maps to qBittorrent's priority system (0-7)
- File-level priority for selective downloading

---

## Story 8.3: Bandwidth Settings Control

As a **NAS user**,
I want to **manage bandwidth settings**,
So that **downloads don't saturate my network**.

**Acceptance Criteria:**

**Given** the user opens Settings > Downloads
**When** viewing bandwidth settings
**Then** they see:
- Global download limit (KB/s)
- Global upload limit (KB/s)
- Alternative speed limits (for scheduled mode)

**Given** bandwidth limits are set
**When** the user saves changes
**Then** qBittorrent applies the limits immediately
**And** current speeds adjust within 5 seconds

**Given** alternative speed mode
**When** the user toggles "Alternative Speed"
**Then** the preset slower limits are applied
**And** status bar shows alternative mode is active

**Given** per-torrent limits are needed
**When** the user selects a specific torrent
**Then** individual download/upload limits can be set
**And** these override global settings

**Technical Notes:**
- Implements FR36: Manage bandwidth settings
- Maps to qBittorrent preferences API
- Alternative speed mode for peak hours

---

## Story 8.4: Download Scheduling

As a **NAS user**,
I want to **schedule downloads for specific times**,
So that **downloads run during off-peak hours**.

**Acceptance Criteria:**

**Given** the user opens Settings > Schedule
**When** configuring the schedule
**Then** a weekly time grid is available (7 days × 24 hours)
**And** users can select time blocks for each mode

**Given** schedule modes are:
- Full Speed: No limits
- Alternative Speed: Use alternative limits
- Pause All: No downloading
**When** the user selects time blocks
**Then** each block is assigned a mode
**And** visual color coding shows the schedule

**Given** a schedule is configured
**When** the scheduled time arrives
**Then** qBittorrent automatically switches modes
**And** Vido displays current schedule status

**Given** the user wants a simple schedule
**When** using "Quick Schedule"
**Then** presets are available: "Night Only (00:00-06:00)", "Off-Peak (20:00-08:00)"
**And** one-click applies the schedule

**Technical Notes:**
- Implements FR37: Schedule downloads
- Uses qBittorrent's built-in scheduler
- Visual weekly grid interface

---
