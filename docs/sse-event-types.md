# SSE Event Types Reference

> **Endpoint:** `GET /api/v1/events`
> **Protocol:** Server-Sent Events (text/event-stream)
> **Last Updated:** 2026-03-26

## Overview

Vido uses Server-Sent Events for real-time progress updates across three features: media library scanning, subtitle search/download, and batch subtitle processing. The SSE hub uses a fan-out pattern with non-blocking broadcast and per-client buffered channels (capacity 100).

## Connection

Connect to the SSE endpoint using the browser `EventSource` API:

```typescript
const es = new EventSource('/api/v1/events');

es.addEventListener('scan_progress', (e: MessageEvent) => {
  const event = JSON.parse(e.data);
  console.log(event.data);
});
```

On connection, the server sends a `connected` event:

```json
{
  "type": "connected",
  "data": {
    "clientId": "uuid",
    "message": "Connected to event stream"
  }
}
```

A `ping` keepalive is sent every 30 seconds:

```json
{
  "type": "ping",
  "data": {
    "timestamp": 1711468800
  }
}
```

## Event Types

### scan_progress

Media library scan progress updates. Broadcast every 10 files during a scan and once on completion.

**Publisher:** `ScannerService.broadcastProgress()`

| Field          | Type     | Description                      |
| -------------- | -------- | -------------------------------- |
| `filesFound`   | `int`    | Total files discovered so far    |
| `filesCreated` | `int`    | New media entries created        |
| `filesUpdated` | `int`    | Existing entries updated         |
| `filesSkipped` | `int`    | Files skipped (already indexed)  |
| `filesRemoved` | `int`    | Orphaned entries removed         |
| `errorCount`   | `int`    | Errors encountered               |
| `currentFile`  | `string` | File currently being processed   |
| `percentDone`  | `int`    | Progress percentage (0-100)      |
| `isActive`     | `bool`   | Whether scan is still running    |
| `startedAt`    | `string` | ISO 8601 timestamp of scan start |

**Example payload:**

```json
{
  "id": "uuid",
  "type": "scan_progress",
  "data": {
    "filesFound": 142,
    "filesCreated": 38,
    "filesUpdated": 2,
    "filesSkipped": 100,
    "filesRemoved": 0,
    "errorCount": 2,
    "currentFile": "/media/movies/Inception.2010.BluRay.mkv",
    "percentDone": 67,
    "isActive": true,
    "startedAt": "2026-03-26T14:30:00Z"
  }
}
```

---

### scan_complete

Sent once when a media library scan finishes successfully. Contains final counts.

**Publisher:** `ScannerService.broadcastScanComplete()`

| Field           | Type     | Description                 |
| --------------- | -------- | --------------------------- |
| `files_found`   | `int`    | Total files discovered      |
| `files_created` | `int`    | New media entries created   |
| `files_updated` | `int`    | Existing entries updated    |
| `files_skipped` | `int`    | Files skipped               |
| `files_removed` | `int`    | Orphaned entries removed    |
| `error_count`   | `int`    | Errors encountered          |
| `duration`      | `string` | Scan duration (e.g. "2.5s") |

**Example payload:**

```json
{
  "id": "uuid",
  "type": "scan_complete",
  "data": {
    "files_found": 142,
    "files_created": 38,
    "files_updated": 2,
    "files_skipped": 100,
    "files_removed": 2,
    "error_count": 0,
    "duration": "4.231s"
  }
}
```

---

### scan_cancelled

Sent when a scan is cancelled by the user. Contains partial counts at the time of cancellation.

**Publisher:** `ScannerService.broadcastScanCancelled()`

| Field         | Type  | Description                          |
| ------------- | ----- | ------------------------------------ |
| `files_found` | `int` | Files discovered before cancellation |
| `error_count` | `int` | Errors encountered before cancel     |

**Example payload:**

```json
{
  "id": "uuid",
  "type": "scan_cancelled",
  "data": {
    "files_found": 42,
    "error_count": 1
  }
}
```

---

### subtitle_progress

Subtitle pipeline progress for individual media items. Broadcast at each pipeline stage during search and download.

**Publishers:** `Engine.broadcastStatus()`, `SubtitleHandler.broadcastStatus()`

| Field        | Type     | Description                        |
| ------------ | -------- | ---------------------------------- |
| `media_id`   | `string` | Media item UUID                    |
| `media_type` | `string` | `"movie"` or `"series"`            |
| `stage`      | `string` | Current pipeline stage (see below) |
| `message`    | `string` | Human-readable status message      |

**Pipeline stages (in order):**

| Stage         | Description                                                |
| ------------- | ---------------------------------------------------------- |
| `searching`   | Querying subtitle providers (Assrt, Zimuku, OpenSubtitles) |
| `scoring`     | Ranking results by language, resolution, source trust      |
| `downloading` | Fetching the subtitle file                                 |
| `converting`  | OpenCC language conversion (Simplified → Traditional)      |
| `placing`     | Writing subtitle file to disk alongside media              |
| `complete`    | Subtitle successfully placed                               |
| `failed`      | Error occurred at any stage                                |

**Example payload:**

```json
{
  "type": "subtitle_progress",
  "data": {
    "media_id": "550e8400-e29b-41d4-a716-446655440000",
    "media_type": "movie",
    "stage": "converting",
    "message": "Converting simplified to traditional Chinese..."
  }
}
```

---

### subtitle_batch_progress

Batch subtitle processing progress. Broadcast after each item completes and when the batch finishes.

**Publisher:** `BatchProcessor.broadcastProgress()`

| Field           | Type     | Description                  |
| --------------- | -------- | ---------------------------- |
| `batch_id`      | `string` | Batch operation UUID         |
| `total_items`   | `int`    | Total items in batch         |
| `current_index` | `int`    | Current item index (0-based) |
| `current_item`  | `string` | Name of item being processed |
| `success_count` | `int`    | Successfully processed items |
| `fail_count`    | `int`    | Failed items                 |
| `status`        | `string` | Batch status (see below)     |

**Status values:**

| Status      | Description                     |
| ----------- | ------------------------------- |
| `running`   | Batch is actively processing    |
| `complete`  | All items have been processed   |
| `cancelled` | User cancelled the batch        |
| `error`     | Critical error halted the batch |

**Example payload:**

```json
{
  "type": "subtitle_batch_progress",
  "data": {
    "batch_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "total_items": 42,
    "current_index": 15,
    "current_item": "Inception (2010)",
    "success_count": 12,
    "fail_count": 3,
    "status": "running"
  }
}
```

---

### notification

General-purpose notifications. Defined but not actively used in current implementation.

| Field     | Type     | Description                         |
| --------- | -------- | ----------------------------------- |
| `level`   | `string` | `"error"`, `"warning"`, or `"info"` |
| `title`   | `string` | Notification title                  |
| `message` | `string` | Human-readable message              |

---

## Implementation Notes

### Hub Architecture

- **Broadcast channel:** 256-event buffer, non-blocking send (drops if full)
- **Per-client channel:** 100-event buffer
- **Keepalive:** 30-second ping interval
- **Gzip exclusion:** SSE endpoint is excluded from gzip middleware to preserve streaming

### Frontend Connection Patterns

**Lazy connection** (recommended): Only connect when real-time updates are needed. Disconnect when idle to avoid blocking Playwright `networkidle` detection in E2E tests.

```typescript
useEffect(() => {
  if (!isActive) {
    eventSource?.close();
    return;
  }
  const es = new EventSource('/api/v1/events');
  // ... listen for events
  return () => es.close();
}, [isActive]);
```

**Polling fallback:** If SSE connection fails, fall back to polling the relevant REST endpoint.

### Source Files

| Component                  | Path                                             |
| -------------------------- | ------------------------------------------------ |
| Hub + Event types          | `apps/api/internal/sse/hub.go`                   |
| SSE handler                | `apps/api/internal/sse/handler.go`               |
| Hub tests                  | `apps/api/internal/sse/hub_test.go`              |
| Scanner publisher          | `apps/api/internal/services/scanner_service.go`  |
| Subtitle publisher         | `apps/api/internal/subtitle/engine.go`           |
| Batch publisher            | `apps/api/internal/subtitle/batch.go`            |
| Subtitle handler publisher | `apps/api/internal/handlers/subtitle_handler.go` |
| Scan progress hook         | `apps/web/src/hooks/useScanProgress.ts`          |
| Subtitle search hook       | `apps/web/src/hooks/useSubtitleSearch.ts`        |
