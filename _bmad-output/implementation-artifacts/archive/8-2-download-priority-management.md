# Story 8.2: Download Priority Management

Status: ready-for-dev

## Story

As a **media collector**,
I want to **adjust download priority**,
So that **important downloads complete first**.

## Acceptance Criteria

1. **AC1: Display Current Priority**
   - Given multiple torrents are downloading
   - When the user views the download list
   - Then each torrent shows its current priority level

2. **AC2: Set Torrent Priority**
   - Given a torrent is selected
   - When the user clicks "Set Priority"
   - Then options are available: High, Normal, Low
   - And the change is applied immediately

3. **AC3: Bandwidth Allocation**
   - Given priority is changed
   - When qBittorrent processes the change
   - Then bandwidth allocation adjusts accordingly
   - And higher priority torrents get more bandwidth

4. **AC4: File-Level Priority**
   - Given the user expands torrent details
   - When viewing individual files
   - Then individual files can be set to High/Normal/Low/Skip
   - And Skip means the file won't download

5. **AC5: Queuing Disabled Handling**
   - Given qBittorrent has torrent queuing disabled
   - When the user tries to change priority
   - Then a helpful message explains queuing must be enabled
   - And links to qBittorrent settings

## Tasks / Subtasks

- [ ] Task 1: Extend qBittorrent Client with Priority Methods (AC: 1, 2, 3, 4, 5)
  - [ ] 1.1: Add `IncreasePriority(ctx, hashes []string) error` to `/apps/api/internal/qbittorrent/client.go`
  - [ ] 1.2: Add `DecreasePriority(ctx, hashes []string) error`
  - [ ] 1.3: Add `TopPriority(ctx, hashes []string) error`
  - [ ] 1.4: Add `BottomPriority(ctx, hashes []string) error`
  - [ ] 1.5: Add `SetFilePriority(ctx, hash string, fileIDs []int, priority FilePriority) error`
  - [ ] 1.6: Add `GetTorrentFiles(ctx, hash string) ([]TorrentFile, error)`
  - [ ] 1.7: Handle HTTP 409 (Conflict) when queuing is disabled
  - [ ] 1.8: Write unit tests with mock HTTP server

- [ ] Task 2: Add Priority Types (AC: 1, 2, 4)
  - [ ] 2.1: Add to `/apps/api/internal/qbittorrent/types.go`:
    - `FilePriority` type (0=Skip, 1=Normal, 6=High, 7=Maximal)
    - `TorrentFile` struct (name, size, progress, priority, file index)
  - [ ] 2.2: Write type tests

- [ ] Task 3: Extend Download Service with Priority Methods (AC: 2, 3, 4, 5)
  - [ ] 3.1: Add `SetTorrentPriority(ctx, hash string, level PriorityLevel) error` to download_service.go
  - [ ] 3.2: Add `GetTorrentFiles(ctx, hash string) ([]TorrentFile, error)`
  - [ ] 3.3: Add `SetFilePriority(ctx, hash string, fileIDs []int, priority FilePriority) error`
  - [ ] 3.4: Map UI priority levels (High/Normal/Low) to qBittorrent API calls (topPrio/no-change/bottomPrio)
  - [ ] 3.5: Handle 409 Conflict gracefully with descriptive AppError
  - [ ] 3.6: Write service tests

- [ ] Task 4: Create Priority Handler Endpoints (AC: 1, 2, 4, 5)
  - [ ] 4.1: Add `POST /api/v1/downloads/{hash}/priority` to download_handler.go
  - [ ] 4.2: Add `GET /api/v1/downloads/{hash}/files` to list files in torrent
  - [ ] 4.3: Add `POST /api/v1/downloads/{hash}/files/priority` to set file priority
  - [ ] 4.4: Add Swagger documentation
  - [ ] 4.5: Write handler tests

- [ ] Task 5: Register Routes (AC: all)
  - [ ] 5.1: Register new priority routes in `main.go`

- [ ] Task 6: Create Priority UI Components (AC: 1, 2)
  - [ ] 6.1: Create `/apps/web/src/components/downloads/PrioritySelector.tsx`
  - [ ] 6.2: Dropdown or button group with High/Normal/Low options
  - [ ] 6.3: Visual indicator (icon/color) for current priority on each download item
  - [ ] 6.4: Loading state during priority change
  - [ ] 6.5: Write component tests

- [ ] Task 7: Create File Priority Panel (AC: 4)
  - [ ] 7.1: Create `/apps/web/src/components/downloads/TorrentFilesPanel.tsx`
  - [ ] 7.2: Expandable panel showing files within a torrent
  - [ ] 7.3: Each file shows: name, size, progress, priority dropdown
  - [ ] 7.4: Priority options: High, Normal, Low, Skip (with strikethrough styling for Skip)
  - [ ] 7.5: Write component tests

- [ ] Task 8: Create Frontend API Methods (AC: 1, 2, 4)
  - [ ] 8.1: Add `setTorrentPriority(hash, level)` to downloadService.ts
  - [ ] 8.2: Add `getTorrentFiles(hash)`
  - [ ] 8.3: Add `setFilePriority(hash, fileIds, priority)`
  - [ ] 8.4: Add TanStack Query hooks for files list and priority mutations

- [ ] Task 9: Integrate into Download Dashboard (AC: 1, 2, 4)
  - [ ] 9.1: Add priority indicator to DownloadItem.tsx
  - [ ] 9.2: Add expandable file list to DownloadItem.tsx
  - [ ] 9.3: Invalidate queries on priority change
  - [ ] 9.4: Toast notifications for success/failure

- [ ] Task 10: Handle Queuing Disabled (AC: 5)
  - [ ] 10.1: Create `/apps/web/src/components/downloads/QueueingDisabledAlert.tsx`
  - [ ] 10.2: Show when 409 Conflict returned from API
  - [ ] 10.3: Message: "請在 qBittorrent 設定中啟用排隊功能以使用優先度管理"

- [ ] Task 11: Write Tests (AC: all)
  - [ ] 11.1: Backend unit tests - coverage ≥80%
  - [ ] 11.2: Frontend component tests - coverage ≥70%
  - [ ] 11.3: E2E test: `/e2e/download-priority.spec.ts`

## Dev Notes

### Architecture Requirements

**FR35: Adjust download priority**
- Maps to qBittorrent's priority system (0-7)
- File-level priority for selective downloading
- Requires qBittorrent's torrent queuing feature enabled

### qBittorrent Web API Reference

```
Increase Priority:
POST /api/v2/torrents/increasePrio
  Body: hashes={hash1}|{hash2}
  Response: HTTP 200 or 409 (queuing disabled)

Decrease Priority:
POST /api/v2/torrents/decreasePrio
  Body: hashes={hash1}|{hash2}
  Response: HTTP 200 or 409 (queuing disabled)

Set Maximum Priority:
POST /api/v2/torrents/topPrio
  Body: hashes={hash1}|{hash2}
  Response: HTTP 200 or 409 (queuing disabled)

Set Minimum Priority:
POST /api/v2/torrents/bottomPrio
  Body: hashes={hash1}|{hash2}
  Response: HTTP 200 or 409 (queuing disabled)

Get Torrent Files:
(included in torrent properties or via /api/v2/torrents/files)
GET /api/v2/torrents/files?hash={hash}
  Response: JSON array of file objects
  Each file: { name, size, progress, priority, piece_range, availability }

Set File Priority:
POST /api/v2/torrents/filePrio
  Body: hash={hash}&id={0}|{1}&priority={0-7}
  Response: HTTP 200, 400 (invalid params), 404 (torrent not found), 409 (not seeded)

File Priority Values:
  0 = Do not download (Skip)
  1 = Normal priority
  6 = High priority
  7 = Maximal priority
```

### Vido API Endpoints

```
POST /api/v1/downloads/{hash}/priority
  Body: { "level": "high" | "normal" | "low" }

GET  /api/v1/downloads/{hash}/files
  Response: { "success": true, "data": [{ "id": 0, "name": "file.mkv", "size": 1073741824, "progress": 0.5, "priority": 1 }] }

POST /api/v1/downloads/{hash}/files/priority
  Body: { "file_ids": [0, 1], "priority": 6 }
```

### Priority Level Mapping

```
Vido UI Level → qBittorrent API Call
─────────────────────────────────────
High          → POST /torrents/topPrio
Normal        → (no action, or reset via decrease from top)
Low           → POST /torrents/bottomPrio
```

### Error Codes

- `QBIT_CONNECTION_FAILED` - qBittorrent not reachable
- `QBIT_TORRENT_NOT_FOUND` - Hash doesn't match any torrent
- `QBIT_QUEUING_DISABLED` - Torrent queuing not enabled (HTTP 409)
- `QBIT_OPERATION_FAILED` - Generic operation failure
- `VALIDATION_OUT_OF_RANGE` - Invalid priority value

### Project Structure Notes

**Backend Files to Modify:**
```
/apps/api/internal/qbittorrent/client.go        → Add priority methods + GetTorrentFiles
/apps/api/internal/qbittorrent/client_test.go    → Add tests
/apps/api/internal/qbittorrent/types.go          → Add FilePriority, TorrentFile types
/apps/api/internal/services/download_service.go  → Add priority service methods
/apps/api/internal/handlers/download_handler.go  → Add priority endpoints
/apps/api/main.go                                → Register new routes
```

**Frontend Files to Create:**
```
/apps/web/src/components/downloads/PrioritySelector.tsx
/apps/web/src/components/downloads/PrioritySelector.spec.tsx
/apps/web/src/components/downloads/TorrentFilesPanel.tsx
/apps/web/src/components/downloads/TorrentFilesPanel.spec.tsx
/apps/web/src/components/downloads/QueueingDisabledAlert.tsx
/apps/web/src/components/downloads/QueueingDisabledAlert.spec.tsx
```

**Frontend Files to Modify:**
```
/apps/web/src/services/downloadService.ts        → Add priority methods
/apps/web/src/components/downloads/DownloadItem.tsx → Add priority indicator + expandable files
```

### Dependencies

**Story Dependencies:**
- Story 8-1 (Torrent Control Operations) - Establishes torrent control patterns
- Epic 4 (qBittorrent connection, monitoring, dashboard)

**Library Dependencies:**
- None (uses existing Go standard library + established qbittorrent package)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.2]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR35]
- [Source: _bmad-output/planning-artifacts/architecture.md#QBIT-Error-Codes]
- [Source: _bmad-output/implementation-artifacts/4-1-qbittorrent-connection-configuration.md]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [qBittorrent Web API v4.1](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))

### Previous Story Intelligence

**From Story 8-1 (Torrent Control Operations):**
- qBittorrent client extended with postForm helper for torrent operations
- Download handler pattern for torrent-specific endpoints established
- Frontend mutation pattern with TanStack Query + toast notifications
- v5.0 compatibility pattern (try new endpoint, fall back to v4.x)

**From Epic 4 Stories:**
- qBittorrent client package structure and authentication flow
- Download service architecture with polling and caching
- Frontend download component hierarchy (DownloadList → DownloadItem)

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
