# Story 8.3: Bandwidth Settings Control

Status: ready-for-dev

## Story

As a **NAS user**,
I want to **manage bandwidth settings**,
So that **downloads don't saturate my network**.

## Acceptance Criteria

1. **AC1: Bandwidth Settings Display**
   - Given the user opens Settings > Downloads
   - When viewing bandwidth settings
   - Then they see:
     - Global download limit (KB/s)
     - Global upload limit (KB/s)
     - Alternative speed limits (for scheduled mode)

2. **AC2: Save Bandwidth Limits**
   - Given bandwidth limits are set
   - When the user saves changes
   - Then qBittorrent applies the limits immediately
   - And current speeds adjust within 5 seconds

3. **AC3: Alternative Speed Mode Toggle**
   - Given the alternative speed mode exists
   - When the user toggles "Alternative Speed"
   - Then the preset slower limits are applied
   - And status bar shows alternative mode is active

4. **AC4: Per-Torrent Limits**
   - Given per-torrent limits are needed
   - When the user selects a specific torrent
   - Then individual download/upload limits can be set
   - And these override global settings

5. **AC5: Real-time Speed Display**
   - Given limits are active
   - When viewing the download dashboard
   - Then current global download/upload speed is displayed
   - And speed respects the configured limits

## Tasks / Subtasks

- [ ] Task 1: Extend qBittorrent Client with Bandwidth Methods (AC: 1, 2, 3, 4, 5)
  - [ ] 1.1: Add `GetGlobalDownloadLimit(ctx) (int64, error)` to client.go
  - [ ] 1.2: Add `SetGlobalDownloadLimit(ctx, limit int64) error`
  - [ ] 1.3: Add `GetGlobalUploadLimit(ctx) (int64, error)`
  - [ ] 1.4: Add `SetGlobalUploadLimit(ctx, limit int64) error`
  - [ ] 1.5: Add `GetSpeedLimitsMode(ctx) (bool, error)` (alternative speed active?)
  - [ ] 1.6: Add `ToggleSpeedLimitsMode(ctx) error`
  - [ ] 1.7: Add `GetTorrentDownloadLimit(ctx, hashes []string) (map[string]int64, error)`
  - [ ] 1.8: Add `SetTorrentDownloadLimit(ctx, hashes []string, limit int64) error`
  - [ ] 1.9: Add `GetTorrentUploadLimit(ctx, hashes []string) (map[string]int64, error)`
  - [ ] 1.10: Add `SetTorrentUploadLimit(ctx, hashes []string, limit int64) error`
  - [ ] 1.11: Add `GetTransferInfo(ctx) (*TransferInfo, error)` for real-time speeds
  - [ ] 1.12: Write unit tests with mock HTTP server

- [ ] Task 2: Add Bandwidth Types (AC: 1, 4, 5)
  - [ ] 2.1: Add to types.go:
    - `TransferInfo` struct (dl_info_speed, up_info_speed, dl_info_data, up_info_data)
    - `BandwidthSettings` struct (global dl/up limits, alt dl/up limits)
  - [ ] 2.2: Write type tests

- [ ] Task 3: Create Bandwidth Service (AC: 1, 2, 3, 4, 5)
  - [ ] 3.1: Add `GetBandwidthSettings(ctx) (*BandwidthSettings, error)` to download_service.go
  - [ ] 3.2: Add `SetGlobalLimits(ctx, dlLimit, upLimit int64) error`
  - [ ] 3.3: Add `ToggleAlternativeSpeed(ctx) (bool, error)` - returns new state
  - [ ] 3.4: Add `SetTorrentLimits(ctx, hash string, dlLimit, upLimit int64) error`
  - [ ] 3.5: Add `GetTransferInfo(ctx) (*TransferInfo, error)`
  - [ ] 3.6: Convert between KB/s (UI) and bytes/s (qBittorrent API)
  - [ ] 3.7: Write service tests

- [ ] Task 4: Create Bandwidth Handler Endpoints (AC: 1, 2, 3, 4, 5)
  - [ ] 4.1: Add `GET /api/v1/downloads/bandwidth` - get current limits and speeds
  - [ ] 4.2: Add `PUT /api/v1/downloads/bandwidth` - set global limits
  - [ ] 4.3: Add `POST /api/v1/downloads/bandwidth/toggle-alt` - toggle alternative speed
  - [ ] 4.4: Add `PUT /api/v1/downloads/{hash}/limits` - set per-torrent limits
  - [ ] 4.5: Add `GET /api/v1/downloads/transfer` - get real-time transfer info
  - [ ] 4.6: Add Swagger documentation
  - [ ] 4.7: Write handler tests

- [ ] Task 5: Register Routes (AC: all)
  - [ ] 5.1: Register bandwidth routes in `main.go`

- [ ] Task 6: Create Bandwidth Settings UI (AC: 1, 2)
  - [ ] 6.1: Create `/apps/web/src/components/downloads/BandwidthSettings.tsx`
  - [ ] 6.2: Input fields for global download/upload limits (KB/s)
  - [ ] 6.3: Input fields for alternative speed limits
  - [ ] 6.4: Save button with validation (non-negative integers, 0 = unlimited)
  - [ ] 6.5: Show "0 = unlimited" hint
  - [ ] 6.6: Write component tests

- [ ] Task 7: Create Alternative Speed Toggle (AC: 3)
  - [ ] 7.1: Create `/apps/web/src/components/downloads/AltSpeedToggle.tsx`
  - [ ] 7.2: Toggle button/switch with icon (turtle icon for alt speed)
  - [ ] 7.3: Visual indicator in download dashboard header
  - [ ] 7.4: Show current mode (Normal / Alternative)
  - [ ] 7.5: Write component tests

- [ ] Task 8: Create Per-Torrent Limit Dialog (AC: 4)
  - [ ] 8.1: Create `/apps/web/src/components/downloads/TorrentLimitDialog.tsx`
  - [ ] 8.2: Dialog with download/upload limit inputs
  - [ ] 8.3: Show current limits, 0 = use global
  - [ ] 8.4: Write component tests

- [ ] Task 9: Create Speed Display Widget (AC: 5)
  - [ ] 9.1: Create `/apps/web/src/components/downloads/SpeedIndicator.tsx`
  - [ ] 9.2: Show real-time download/upload speed in header area
  - [ ] 9.3: Format speed (B/s, KB/s, MB/s) automatically
  - [ ] 9.4: Poll transfer info every 2 seconds (or use existing polling from Epic 4)
  - [ ] 9.5: Write component tests

- [ ] Task 10: Create Frontend API Methods (AC: all)
  - [ ] 10.1: Add `getBandwidthSettings()` to downloadService.ts
  - [ ] 10.2: Add `setGlobalLimits(dlLimit, upLimit)`
  - [ ] 10.3: Add `toggleAlternativeSpeed()`
  - [ ] 10.4: Add `setTorrentLimits(hash, dlLimit, upLimit)`
  - [ ] 10.5: Add `getTransferInfo()`
  - [ ] 10.6: Add TanStack Query hooks with appropriate staleTime

- [ ] Task 11: Integrate into Downloads Page (AC: all)
  - [ ] 11.1: Add speed indicator to downloads page header
  - [ ] 11.2: Add alt speed toggle to downloads page header
  - [ ] 11.3: Add per-torrent limit option to DownloadItem context menu
  - [ ] 11.4: Add bandwidth settings to Settings > Downloads section

- [ ] Task 12: Write Tests (AC: all)
  - [ ] 12.1: Backend unit tests - coverage ≥80%
  - [ ] 12.2: Frontend component tests - coverage ≥70%
  - [ ] 12.3: E2E test: `/e2e/bandwidth-settings.spec.ts`

## Dev Notes

### Architecture Requirements

**FR36: Manage bandwidth settings**
- Maps to qBittorrent preferences API
- Alternative speed mode for peak hours
- Per-torrent overrides for fine-grained control

### qBittorrent Web API Reference

```
Global Speed Limits:
GET  /api/v2/transfer/downloadLimit        → int (bytes/s, 0=unlimited)
POST /api/v2/transfer/setDownloadLimit     → Body: limit={bytes/s}
GET  /api/v2/transfer/uploadLimit          → int (bytes/s, 0=unlimited)
POST /api/v2/transfer/setUploadLimit       → Body: limit={bytes/s}

Alternative Speed Mode:
GET  /api/v2/transfer/speedLimitsMode      → 1 (enabled) or 0 (disabled)
POST /api/v2/transfer/toggleSpeedLimitsMode → toggles, HTTP 200

Per-Torrent Limits:
POST /api/v2/torrents/downloadLimit        → Body: hashes={h1}|{h2}  → JSON { "hash": limit }
POST /api/v2/torrents/setDownloadLimit     → Body: hashes={h1}|{h2}&limit={bytes/s}
POST /api/v2/torrents/uploadLimit          → Body: hashes={h1}|{h2}  → JSON { "hash": limit }
POST /api/v2/torrents/setUploadLimit       → Body: hashes={h1}|{h2}&limit={bytes/s}

Transfer Info (real-time speeds):
GET  /api/v2/transfer/info → JSON { dl_info_speed, up_info_speed, dl_info_data, up_info_data, ... }

Alternative Speed Settings (via preferences):
GET  /api/v2/app/preferences → includes alt_dl_limit, alt_up_limit (KiB/s)
POST /api/v2/app/setPreferences → Body: json={"alt_dl_limit": 500, "alt_up_limit": 100}
```

### CRITICAL: Unit Conversion

qBittorrent API uses **bytes/second** for speed limits, but the UI should display **KB/s**:
```go
// KB/s → bytes/s (for API calls)
bytesPerSec := kbPerSec * 1024

// bytes/s → KB/s (for display)
kbPerSec := bytesPerSec / 1024
```

**Exception:** Alternative speed limits in preferences use **KiB/s** directly.

### Vido API Endpoints

```
GET  /api/v1/downloads/bandwidth
  Response: { "success": true, "data": {
    "global_dl_limit": 0, "global_up_limit": 0,
    "alt_dl_limit": 500, "alt_up_limit": 100,
    "alt_speed_enabled": false,
    "current_dl_speed": 5242880, "current_up_speed": 1048576
  }}

PUT  /api/v1/downloads/bandwidth
  Body: { "global_dl_limit": 1024, "global_up_limit": 512 }  (KB/s, 0=unlimited)

POST /api/v1/downloads/bandwidth/toggle-alt
  Response: { "success": true, "data": { "alt_speed_enabled": true } }

PUT  /api/v1/downloads/{hash}/limits
  Body: { "dl_limit": 512, "up_limit": 256 }  (KB/s, 0=use global)

GET  /api/v1/downloads/transfer
  Response: { "success": true, "data": {
    "dl_speed": 5242880, "up_speed": 1048576,
    "dl_total": 107374182400, "up_total": 53687091200
  }}
```

### Error Codes

- `QBIT_CONNECTION_FAILED` - qBittorrent not reachable
- `QBIT_OPERATION_FAILED` - Failed to set limits
- `VALIDATION_OUT_OF_RANGE` - Negative limit value
- `VALIDATION_INVALID_FORMAT` - Non-numeric limit value

### Project Structure Notes

**Backend Files to Modify:**
```
/apps/api/internal/qbittorrent/client.go        → Add bandwidth methods + GetTransferInfo
/apps/api/internal/qbittorrent/client_test.go    → Add tests
/apps/api/internal/qbittorrent/types.go          → Add TransferInfo, BandwidthSettings
/apps/api/internal/services/download_service.go  → Add bandwidth service methods
/apps/api/internal/handlers/download_handler.go  → Add bandwidth endpoints
/apps/api/main.go                                → Register new routes
```

**Frontend Files to Create:**
```
/apps/web/src/components/downloads/BandwidthSettings.tsx
/apps/web/src/components/downloads/BandwidthSettings.spec.tsx
/apps/web/src/components/downloads/AltSpeedToggle.tsx
/apps/web/src/components/downloads/AltSpeedToggle.spec.tsx
/apps/web/src/components/downloads/TorrentLimitDialog.tsx
/apps/web/src/components/downloads/TorrentLimitDialog.spec.tsx
/apps/web/src/components/downloads/SpeedIndicator.tsx
/apps/web/src/components/downloads/SpeedIndicator.spec.tsx
```

**Frontend Files to Modify:**
```
/apps/web/src/services/downloadService.ts        → Add bandwidth methods
/apps/web/src/routes/downloads.tsx               → Add speed indicator + alt toggle to header
/apps/web/src/components/downloads/DownloadItem.tsx → Add per-torrent limit option
```

### Dependencies

**Story Dependencies:**
- Story 8-1 (Torrent Control Operations) - Basic torrent interaction patterns
- Epic 4 (qBittorrent connection, monitoring, polling infrastructure)

**Library Dependencies:**
- None (uses existing Go standard library + established qbittorrent package)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.3]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR36]
- [Source: _bmad-output/planning-artifacts/architecture.md#Caching-Boundaries] (qBittorrent status 5s TTL)
- [Source: _bmad-output/implementation-artifacts/4-1-qbittorrent-connection-configuration.md]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [qBittorrent Web API v4.1](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))

### Previous Story Intelligence

**From Story 8-1 (Torrent Control Operations):**
- qBittorrent client method pattern (postForm with url.Values)
- Download handler endpoint pattern for torrent-specific operations
- TanStack Query mutation + cache invalidation pattern

**From Story 8-2 (Priority Management):**
- Per-torrent operation pattern (hash in URL path)
- Handling qBittorrent 409 responses

**From Epic 4 Stories:**
- Polling infrastructure for real-time status (5-second interval)
- Download dashboard layout and component hierarchy
- TransferInfo data may already be available from Epic 4 polling - check and REUSE

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
