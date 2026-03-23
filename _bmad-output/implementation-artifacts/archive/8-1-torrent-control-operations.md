# Story 8.1: Torrent Control Operations

Status: ready-for-dev

## Story

As a **media collector**,
I want to **control my torrents directly from Vido**,
So that **I don't need to switch to the qBittorrent interface for basic operations**.

## Acceptance Criteria

1. **AC1: Control Buttons on Download List**
   - Given the user views the download list
   - When they select a torrent
   - Then control buttons are available: Pause, Resume, Delete

2. **AC2: Pause Torrent**
   - Given a running torrent
   - When the user clicks "Pause"
   - Then the torrent pauses immediately via qBittorrent API
   - And status updates within 2 seconds

3. **AC3: Resume Torrent**
   - Given a paused torrent
   - When the user clicks "Resume"
   - Then the torrent resumes
   - And status shows "Downloading"

4. **AC4: Delete Torrent with Confirmation**
   - Given the user clicks "Delete"
   - When confirming the action
   - Then a dialog asks: "Delete torrent only" or "Delete with files"
   - And the selected action is executed
   - And confirmation shows success

5. **AC5: Bulk Operations**
   - Given multiple torrents are selected
   - When using bulk actions
   - Then Pause All / Resume All / Delete Selected are available
   - And operations apply to all selected torrents

## Tasks / Subtasks

- [ ] Task 1: Extend qBittorrent Client with Torrent Control Methods (AC: 1, 2, 3, 4, 5)
  - [ ] 1.1: Add `PauseTorrents(ctx, hashes []string) error` to `/apps/api/internal/qbittorrent/client.go`
  - [ ] 1.2: Add `ResumeTorrents(ctx, hashes []string) error`
  - [ ] 1.3: Add `DeleteTorrents(ctx, hashes []string, deleteFiles bool) error`
  - [ ] 1.4: Handle qBittorrent v5.0 API changes (pause→stop, resume→start) with version detection
  - [ ] 1.5: Write unit tests with mock HTTP server (`client_test.go`)

- [ ] Task 2: Create Torrent Control Types (AC: 1, 4)
  - [ ] 2.1: Add torrent action types to `/apps/api/internal/qbittorrent/types.go`
  - [ ] 2.2: Define `TorrentAction` enum (Pause, Resume, Delete)
  - [ ] 2.3: Define `DeleteOptions` struct (deleteFiles bool)
  - [ ] 2.4: Write type tests

- [ ] Task 3: Extend Download Service with Control Methods (AC: 2, 3, 4, 5)
  - [ ] 3.1: Add `PauseTorrents(ctx, hashes []string) error` to `/apps/api/internal/services/download_service.go`
  - [ ] 3.2: Add `ResumeTorrents(ctx, hashes []string) error`
  - [ ] 3.3: Add `DeleteTorrents(ctx, hashes []string, deleteFiles bool) error`
  - [ ] 3.4: Add validation (check hashes exist before operation)
  - [ ] 3.5: Invalidate cached torrent list after mutations
  - [ ] 3.6: Write service tests with mock qBittorrent client

- [ ] Task 4: Create Download Control Handler Endpoints (AC: 1, 2, 3, 4, 5)
  - [ ] 4.1: Add `POST /api/v1/downloads/{hash}/pause` to `/apps/api/internal/handlers/download_handler.go`
  - [ ] 4.2: Add `POST /api/v1/downloads/{hash}/resume`
  - [ ] 4.3: Add `POST /api/v1/downloads/{hash}/delete` with `delete_files` body param
  - [ ] 4.4: Add `POST /api/v1/downloads/bulk/pause` (bulk action)
  - [ ] 4.5: Add `POST /api/v1/downloads/bulk/resume` (bulk action)
  - [ ] 4.6: Add `POST /api/v1/downloads/bulk/delete` (bulk action)
  - [ ] 4.7: Add Swagger documentation for all endpoints
  - [ ] 4.8: Write handler tests

- [ ] Task 5: Register Routes and Wire Dependencies (AC: all)
  - [ ] 5.1: Register new routes in router setup in `main.go`
  - [ ] 5.2: Verify integration with existing DownloadService

- [ ] Task 6: Create Torrent Control UI Components (AC: 1, 2, 3, 4)
  - [ ] 6.1: Create `/apps/web/src/components/downloads/TorrentControlBar.tsx`
  - [ ] 6.2: Add Pause/Resume/Delete buttons with icons
  - [ ] 6.3: Show contextual buttons based on torrent state (paused → show Resume, downloading → show Pause)
  - [ ] 6.4: Add loading states during API calls
  - [ ] 6.5: Write component tests

- [ ] Task 7: Create Delete Confirmation Dialog (AC: 4)
  - [ ] 7.1: Create `/apps/web/src/components/downloads/DeleteTorrentDialog.tsx`
  - [ ] 7.2: Two options: "Delete torrent only" / "Delete with files"
  - [ ] 7.3: Show torrent name in dialog for confirmation
  - [ ] 7.4: Add destructive action styling (red for "Delete with files")
  - [ ] 7.5: Write component tests

- [ ] Task 8: Create Frontend API Service Methods (AC: 1, 2, 3, 4, 5)
  - [ ] 8.1: Add `pauseTorrent(hash)` to `/apps/web/src/services/downloadService.ts`
  - [ ] 8.2: Add `resumeTorrent(hash)`
  - [ ] 8.3: Add `deleteTorrent(hash, deleteFiles)`
  - [ ] 8.4: Add bulk operation methods
  - [ ] 8.5: Add TanStack Query mutation hooks with cache invalidation

- [ ] Task 9: Integrate Controls into Download Dashboard (AC: 1, 5)
  - [ ] 9.1: Update `/apps/web/src/components/downloads/DownloadItem.tsx` with inline controls
  - [ ] 9.2: Add selection checkboxes for bulk operations
  - [ ] 9.3: Add bulk action bar that appears when items selected
  - [ ] 9.4: Invalidate `['downloads']` query key after mutations
  - [ ] 9.5: Show toast notifications on success/failure

- [ ] Task 10: Write Tests (AC: all)
  - [ ] 10.1: Backend unit tests (client, service, handler) - coverage ≥80%
  - [ ] 10.2: Frontend component tests - coverage ≥70%
  - [ ] 10.3: E2E test: `/e2e/torrent-control.spec.ts`

## Dev Notes

### Architecture Requirements

**FR34: Control qBittorrent directly**
- Pause, Resume, Delete torrents from Vido UI
- Uses qBittorrent Web API v2.x
- Requires confirmed connection from Epic 4

### qBittorrent Web API Reference

```
Base URL: {host}{basePath}/api/v2

Pause Torrents (v4.x):
POST /torrents/pause
  Body: hashes={hash1}|{hash2}  (pipe-separated, or "all")
  Response: HTTP 200

Stop Torrents (v5.0+ replacement for pause):
POST /torrents/stop
  Body: hashes={hash1}|{hash2}
  Response: HTTP 200

Resume Torrents (v4.x):
POST /torrents/resume
  Body: hashes={hash1}|{hash2}  (pipe-separated, or "all")
  Response: HTTP 200

Start Torrents (v5.0+ replacement for resume):
POST /torrents/start
  Body: hashes={hash1}|{hash2}
  Response: HTTP 200

Delete Torrents:
POST /torrents/delete
  Body: hashes={hash1}|{hash2}&deleteFiles=true|false
  Response: HTTP 200
```

### CRITICAL: qBittorrent v5.0 Breaking Change

In qBittorrent v5.0, `pause`/`resume` were renamed to `stop`/`start`. Implementation MUST handle both versions:

```go
// Strategy: Try v5.0 endpoints first, fall back to v4.x on 404
func (c *Client) PauseTorrents(ctx context.Context, hashes []string) error {
    // Try v5.0: /torrents/stop
    err := c.post(ctx, "/torrents/stop", hashesParam(hashes))
    if isNotFound(err) {
        // Fallback to v4.x: /torrents/pause
        return c.post(ctx, "/torrents/pause", hashesParam(hashes))
    }
    return err
}
```

### Backend Implementation Pattern

Following Story 4-1 established patterns:

```go
// Extend existing client.go - DO NOT create new file
// Add methods to existing Client struct in /apps/api/internal/qbittorrent/client.go

func (c *Client) PauseTorrents(ctx context.Context, hashes []string) error {
    hashStr := strings.Join(hashes, "|")
    return c.postForm(ctx, "/torrents/pause", url.Values{"hashes": {hashStr}})
}

func (c *Client) ResumeTorrents(ctx context.Context, hashes []string) error {
    hashStr := strings.Join(hashes, "|")
    return c.postForm(ctx, "/torrents/resume", url.Values{"hashes": {hashStr}})
}

func (c *Client) DeleteTorrents(ctx context.Context, hashes []string, deleteFiles bool) error {
    hashStr := strings.Join(hashes, "|")
    data := url.Values{
        "hashes":      {hashStr},
        "deleteFiles": {fmt.Sprintf("%t", deleteFiles)},
    }
    return c.postForm(ctx, "/torrents/delete", data)
}
```

### API Endpoints

```
POST /api/v1/downloads/{hash}/pause     → DownloadHandler.PauseTorrent
POST /api/v1/downloads/{hash}/resume    → DownloadHandler.ResumeTorrent
POST /api/v1/downloads/{hash}/delete    → DownloadHandler.DeleteTorrent
POST /api/v1/downloads/bulk/pause       → DownloadHandler.BulkPause
POST /api/v1/downloads/bulk/resume      → DownloadHandler.BulkResume
POST /api/v1/downloads/bulk/delete      → DownloadHandler.BulkDelete
```

### Handler Pattern

```go
type DeleteTorrentRequest struct {
    DeleteFiles bool `json:"delete_files"`
}

type BulkActionRequest struct {
    Hashes      []string `json:"hashes" binding:"required,min=1"`
    DeleteFiles bool     `json:"delete_files"` // only for bulk delete
}
```

### Frontend Mutation Pattern

```typescript
// Use TanStack Query mutations with optimistic updates
const pauseMutation = useMutation({
    mutationFn: (hash: string) => downloadService.pauseTorrent(hash),
    onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['downloads'] });
        toast.success('Torrent 已暫停');
    },
    onError: (error: ApiError) => {
        toast.error(error.message);
    },
});
```

### Error Codes

Following project-context.md Rule 7:
- `QBIT_CONNECTION_FAILED` - qBittorrent not reachable
- `QBIT_AUTH_FAILED` - Session expired, re-authentication needed
- `QBIT_TORRENT_NOT_FOUND` - Hash doesn't match any torrent
- `QBIT_OPERATION_FAILED` - Generic operation failure

### Project Structure Notes

**Backend Files to Modify:**
```
/apps/api/internal/qbittorrent/client.go        → Add PauseTorrents, ResumeTorrents, DeleteTorrents
/apps/api/internal/qbittorrent/client_test.go    → Add tests
/apps/api/internal/qbittorrent/types.go          → Add action types
/apps/api/internal/services/download_service.go  → Add control methods
/apps/api/internal/handlers/download_handler.go  → Add control endpoints
/apps/api/main.go                                → Register new routes
```

**Frontend Files to Create:**
```
/apps/web/src/components/downloads/TorrentControlBar.tsx
/apps/web/src/components/downloads/TorrentControlBar.spec.tsx
/apps/web/src/components/downloads/DeleteTorrentDialog.tsx
/apps/web/src/components/downloads/DeleteTorrentDialog.spec.tsx
```

**Frontend Files to Modify:**
```
/apps/web/src/services/downloadService.ts        → Add control methods
/apps/web/src/components/downloads/DownloadItem.tsx → Add inline controls
```

### Dependencies

**Epic Dependencies:**
- Epic 4 (Stories 4-1 through 4-6) - qBittorrent connection, monitoring, dashboard must be implemented

**Library Dependencies:**
- None (uses existing Go standard library + established qbittorrent package)

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-8.1]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR34]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-I1]
- [Source: _bmad-output/planning-artifacts/architecture.md#QBIT-Error-Codes]
- [Source: _bmad-output/implementation-artifacts/4-1-qbittorrent-connection-configuration.md]
- [Source: project-context.md#Rule-4-Layered-Architecture]
- [qBittorrent Web API v4.1](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1))
- [qBittorrent Web API v5.0](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-5.0))

### Previous Story Intelligence

**From Epic 4 Stories:**
- qBittorrent client package at `/apps/api/internal/qbittorrent/` with `Client` struct, auth, cookie management
- Download service at `/apps/api/internal/services/download_service.go`
- Download handler at `/apps/api/internal/handlers/download_handler.go`
- Frontend download components at `/apps/web/src/components/downloads/`
- Frontend download service at `/apps/web/src/services/downloadService.ts`
- All qBittorrent API calls go through the existing `Client.postForm()` / `Client.get()` helper methods
- Error handling uses AppError with `QBIT_*` error codes
- Re-authentication happens automatically when session expires

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
