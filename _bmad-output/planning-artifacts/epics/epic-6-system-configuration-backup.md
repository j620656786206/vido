# Epic 6: System Configuration & Backup

**Phase:** 1.0 (Q2 - June 2026)

**Goal:** Users can configure the system through a setup wizard, manage cache, view logs, and backup/restore their data.

## Story 6.1: Setup Wizard

As a **first-time user**,
I want a **guided setup wizard**,
So that **I can configure Vido quickly without confusion**.

**Acceptance Criteria:**

**Given** the user opens Vido for the first time
**When** no configuration exists
**Then** the setup wizard launches automatically
**And** shows progress: Step 1 of 5

**Given** the wizard is running
**When** completing each step:
1. Welcome & language selection
2. qBittorrent connection (skip option available)
3. Media folder configuration
4. API keys (optional, can skip)
5. Complete
**Then** each step validates before proceeding
**And** back navigation is available

**Given** the user completes the wizard
**When** clicking "Finish"
**Then** settings are saved
**And** the user is taken to the main dashboard

**Technical Notes:**
- Implements FR52: Initial setup via wizard
- Implements NFR-U2: Setup wizard <5 steps
- Implements UX-7: Minimal onboarding

---

## Story 6.2: Cache Management

As a **system administrator**,
I want to **view and manage cached data**,
So that **I can reclaim disk space when needed**.

**Acceptance Criteria:**

**Given** the user opens Settings > Cache
**When** viewing cache information
**Then** they see:
- Image cache: X.X GB
- AI parsing cache: X MB
- Metadata cache: X MB
- Total: X.X GB

**Given** cache information is displayed
**When** the user clicks "Clear cache older than 30 days"
**Then** old cache is removed
**And** space reclaimed is shown

**Given** individual cache types are shown
**When** the user clicks "Clear" on a specific type
**Then** only that cache type is cleared
**And** a confirmation is required

**Technical Notes:**
- Implements FR53: Manage cache
- Implements ARCH-5: Cache Management System
- Shows cache by type (images, AI, metadata)

---

## Story 6.3: System Logs Viewer

As a **system administrator**,
I want to **view system logs**,
So that **I can troubleshoot issues and monitor system health**.

**Acceptance Criteria:**

**Given** the user opens Settings > Logs
**When** logs are displayed
**Then** entries show: timestamp, level (ERROR/WARN/INFO/DEBUG), message
**And** logs are color-coded by level

**Given** many log entries exist
**When** viewing the log list
**Then** pagination or infinite scroll is available
**And** newest logs are shown first

**Given** logs are displayed
**When** the user filters by level (e.g., "ERROR only")
**Then** only matching entries are shown
**And** search by keyword is available

**Given** any log entry
**When** it contains sensitive information
**Then** API keys are masked (NFR-S4)
**And** error hints are provided (NFR-U9)

**Technical Notes:**
- Implements FR54: View system logs
- Implements NFR-M11: Severity-level logging
- Implements NFR-U9: Error logs with troubleshooting hints

---

## Story 6.4: Service Connection Status Dashboard

As a **system administrator**,
I want to **see connection status for all external services**,
So that **I can identify integration issues at a glance**.

**Acceptance Criteria:**

**Given** the user opens Settings > Status
**When** the status page loads
**Then** it shows connection status for:
- qBittorrent: 🟢 Connected / 🔴 Disconnected
- TMDb API: 🟢 Available / 🟡 Rate Limited / 🔴 Error
- AI Service: 🟢 Available / 🔴 Error

**Given** a service shows an error
**When** hovering or clicking on the status
**Then** detailed error message is shown
**And** last successful connection time is displayed

**Given** the status page is open
**When** service status changes
**Then** the status updates in real-time
**And** a notification indicates the change

**Technical Notes:**
- Implements FR55: Display service connection status
- Implements NFR-M13: Health status visible
- Implements ARCH-8: Health Check Scheduler

---

## Story 6.5: Database Backup

As a **system administrator**,
I want to **backup my Vido database and configuration**,
So that **I can restore my data if something goes wrong**.

**Acceptance Criteria:**

**Given** the user opens Settings > Backup
**When** they click "Create Backup Now"
**Then** an atomic backup is created using SQLite .backup (NFR-R7)
**And** backup includes: database, configuration, learned mappings

**Given** a backup is created
**When** the backup completes
**Then** it is saved to `/vido-backups` volume
**And** filename format: `vido-backup-YYYYMMDD-HHMMSS-v{schema}.tar.gz`

**Given** backup is in progress
**When** viewing the progress
**Then** a progress indicator is shown
**And** backup for 10,000 items completes in <5 minutes

**Technical Notes:**
- Implements FR57: Backup database and configuration
- Implements NFR-R7: SQLite atomic backups
- Implements ARCH-4: Background Task Queue

---

## Story 6.6: Backup Integrity Verification

As a **system administrator**,
I want **backup integrity to be verified**,
So that **I know my backups are reliable**.

**Acceptance Criteria:**

**Given** a backup is created
**When** the backup completes
**Then** a SHA-256 checksum is calculated
**And** stored alongside the backup file

**Given** a backup file exists
**When** the user clicks "Verify Backup"
**Then** the checksum is recalculated
**And** compared against the stored checksum

**Given** verification fails
**When** the checksum doesn't match
**Then** the backup is marked as "Potentially Corrupted"
**And** the user is warned before attempting restore

**Technical Notes:**
- Implements FR59: Verify backup integrity
- Implements NFR-R8: Backup integrity verification
- Checksum stored in separate .sha256 file

---

## Story 6.7: Data Restore

As a **system administrator**,
I want to **restore data from a backup**,
So that **I can recover from data loss or migration**.

**Acceptance Criteria:**

**Given** backup files exist
**When** the user opens Settings > Backup > Restore
**Then** available backups are listed with date, size, and version

**Given** the user selects a backup
**When** they click "Restore"
**Then** a confirmation dialog warns: "This will replace current data"
**And** an auto-snapshot of current state is created first (NFR-R9)

**Given** restore is confirmed
**When** the restore process runs
**Then** progress is shown
**And** the application restarts with restored data

**Given** restore fails
**When** an error occurs
**Then** the auto-snapshot is used to recover
**And** an error message explains what happened

**Technical Notes:**
- Implements FR58: Restore data from backup
- Implements NFR-R9: Auto-snapshot before restore
- Schema version compatibility checked

---

## Story 6.8: Scheduled Backups

As a **system administrator**,
I want to **schedule automatic backups**,
So that **I don't have to remember to backup manually**.

**Acceptance Criteria:**

**Given** the user opens backup settings
**When** configuring schedule
**Then** options include: Daily, Weekly, or Disabled
**And** time of day can be selected

**Given** scheduled backup is enabled
**When** the scheduled time arrives
**Then** backup runs automatically
**And** runs in background without UI disruption

**Given** backups accumulate
**When** retention policy is active
**Then** keeps last 7 daily + last 4 weekly backups
**And** older backups are automatically deleted (FR64)

**Technical Notes:**
- Implements FR63: Configure backup schedule
- Implements FR64: Auto-cleanup old backups
- Uses ARCH-4: Background Task Queue

---

## Story 6.9: Metadata Export (JSON/YAML/NFO)

As a **power user**,
I want to **export my library metadata in various formats**,
So that **I can use it with other tools or for backup purposes**.

**Acceptance Criteria:**

**Given** the user opens Export options
**When** selecting export format
**Then** options include: JSON, YAML, NFO (Kodi/Plex compatible)

**Given** JSON/YAML export is selected
**When** export completes
**Then** a single file contains all library metadata
**And** the format is human-readable and documented

**Given** NFO export is selected
**When** export completes
**Then** .nfo files are created alongside each media file
**And** format is compatible with Kodi/Plex/Jellyfin

**Given** export is in progress
**When** processing large library
**Then** progress is shown
**And** can be run in background

**Technical Notes:**
- Implements FR60, FR62: Export to JSON/YAML/NFO
- NFO follows Kodi standard format
- Export runs asynchronously

---

## Story 6.10: Metadata Import

As a **power user**,
I want to **import metadata from JSON/YAML files**,
So that **I can restore or migrate my library data**.

**Acceptance Criteria:**

**Given** the user has a JSON/YAML export file
**When** they select "Import Metadata"
**Then** they can upload or provide path to the file

**Given** an import file is provided
**When** import runs
**Then** metadata is merged with existing library
**And** conflicts are handled: Skip / Overwrite / Ask

**Given** import completes
**When** viewing results
**Then** summary shows: Added X, Updated Y, Skipped Z items

**Technical Notes:**
- Implements FR61: Import metadata from JSON/YAML
- Supports incremental import (merge)
- Validates file format before processing

---

## Story 6.11: Performance Metrics Dashboard

As a **system administrator**,
I want to **view performance metrics**,
So that **I can monitor system health and identify issues**.

**Acceptance Criteria:**

**Given** the user opens Settings > Performance
**When** metrics are displayed
**Then** they see:
- Query latency (p50, p95)
- Cache hit rate
- API response times
- Library item count

**Given** metrics show concerning values
**When** p95 latency > 500ms or items > 8,000
**Then** a warning is displayed (NFR-SC2)
**And** recommendation: "Consider PostgreSQL migration"

**Given** the metrics page is open
**When** viewing trends
**Then** charts show 24-hour and 7-day trends

**Technical Notes:**
- Implements FR65: Display performance metrics
- Implements FR66, NFR-SC2: Scalability warnings
- Implements NFR-M12: Performance metrics queryable

---
