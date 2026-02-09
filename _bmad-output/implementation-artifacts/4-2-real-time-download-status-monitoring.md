# Story 4.2: Real-Time Download Status Monitoring

Status: ready-for-dev

## Story

As a **media collector**,
I want to **see real-time download status**,
So that **I can monitor progress without opening qBittorrent**.

## Acceptance Criteria

1. **AC1: Torrent List Display**
   - Given qBittorrent is connected
   - When viewing the downloads page
   - Then all torrents are displayed with:
     - Name
     - Progress percentage (with progress bar)
     - Download/upload speed
     - ETA
     - Status (downloading, paused, seeding, completed, stalled, error)

2. **AC2: Real-Time Updates**
   - Given a torrent is active
   - When 5 seconds pass (NFR-P8)
   - Then the status updates automatically
   - And the UI updates without full page refresh (optimistic updates)

3. **AC3: Polling Management**
   - Given polling is active
   - When the user navigates away from the downloads page
   - Then polling stops to conserve resources
   - And resumes when they return

4. **AC4: Download Details**
   - Given a torrent is displayed
   - When the user clicks on it
   - Then expanded details show:
     - Total size / Downloaded size
     - Seeds / Peers count
     - Save path
     - Added date
     - Completion date (if completed)

5. **AC5: Sort Options**
   - Given torrents are displayed
   - When the user selects a sort option
   - Then torrents are sorted by: Added date, Name, Progress, Status

## Tasks / Subtasks

- [ ] Task 1: Extend qBittorrent Client for Torrent Operations (AC: 1, 4)
  - [ ] 1.1: Add `GetTorrents(ctx) ([]Torrent, error)` to client
  - [ ] 1.2: Add `GetTorrentDetails(ctx, hash) (*TorrentDetails, error)`
  - [ ] 1.3: Define `Torrent` struct with all fields
  - [ ] 1.4: Define `TorrentDetails` struct for expanded view
  - [ ] 1.5: Map qBittorrent status codes to enum
  - [ ] 1.6: Write client tests

- [ ] Task 2: Create Download Types (AC: 1, 4, 5)
  - [ ] 2.1: Create `/apps/api/internal/qbittorrent/torrent.go`
  - [ ] 2.2: Define `TorrentStatus` enum (downloading, paused, seeding, completed, stalled, error, queued)
  - [ ] 2.3: Define `TorrentSortField` enum
  - [ ] 2.4: Write type tests

- [ ] Task 3: Create Download Service (AC: 1, 4, 5)
  - [ ] 3.1: Create `/apps/api/internal/services/download_service.go`
  - [ ] 3.2: Define `DownloadServiceInterface`
  - [ ] 3.3: Implement `GetAllDownloads(ctx, sort, order) ([]Download, error)`
  - [ ] 3.4: Implement `GetDownloadDetails(ctx, hash) (*DownloadDetails, error)`
  - [ ] 3.5: Handle qBittorrent not configured error gracefully
  - [ ] 3.6: Write service tests

- [ ] Task 4: Create Download Handler (AC: 1, 4, 5)
  - [ ] 4.1: Create `/apps/api/internal/handlers/download_handler.go`
  - [ ] 4.2: Implement `GET /api/v1/downloads` - list all torrents
  - [ ] 4.3: Implement `GET /api/v1/downloads/:hash` - get torrent details
  - [ ] 4.4: Add query params for sort (field, order)
  - [ ] 4.5: Add Swagger documentation
  - [ ] 4.6: Write handler tests

- [ ] Task 5: Register Routes (AC: all)
  - [ ] 5.1: Add download routes to router setup
  - [ ] 5.2: Wire DownloadService dependencies in main.go

- [ ] Task 6: Create Download List Component (AC: 1, 2, 5)
  - [ ] 6.1: Create `/apps/web/src/components/downloads/DownloadList.tsx`
  - [ ] 6.2: Create `/apps/web/src/components/downloads/DownloadItem.tsx`
  - [ ] 6.3: Display progress bar with percentage
  - [ ] 6.4: Display speed with proper formatting (KB/s, MB/s)
  - [ ] 6.5: Display ETA with proper formatting
  - [ ] 6.6: Add sort dropdown
  - [ ] 6.7: Write component tests

- [ ] Task 7: Create Download Details Component (AC: 4)
  - [ ] 7.1: Create `/apps/web/src/components/downloads/DownloadDetails.tsx`
  - [ ] 7.2: Show expanded info on click/expand
  - [ ] 7.3: Display all detail fields
  - [ ] 7.4: Write component tests

- [ ] Task 8: Create Downloads Page with Polling (AC: 2, 3)
  - [ ] 8.1: Create `/apps/web/src/routes/downloads.tsx`
  - [ ] 8.2: Use TanStack Query with `refetchInterval: 5000` (NFR-P8)
  - [ ] 8.3: Use `refetchOnWindowFocus: true`
  - [ ] 8.4: Stop polling when page not visible (use document.visibilityState)
  - [ ] 8.5: Add route to TanStack Router configuration

- [ ] Task 9: Create Download API Service (AC: 1, 4)
  - [ ] 9.1: Create `/apps/web/src/services/downloadService.ts`
  - [ ] 9.2: Implement `getDownloads(sort?): Promise<Download[]>`
  - [ ] 9.3: Implement `getDownloadDetails(hash): Promise<DownloadDetails>`
  - [ ] 9.4: Add TanStack Query hooks with polling config

- [ ] Task 10: E2E Tests (AC: all)
  - [ ] 10.1: Create `/e2e/downloads.spec.ts`
  - [ ] 10.2: Test download list display
  - [ ] 10.3: Test sort functionality
  - [ ] 10.4: Test detail expansion
  - [ ] 10.5: Test polling behavior (mock time)

## Dev Notes

### Architecture Requirements

**FR29: Real-time download status**
- Progress, speed, ETA, status
- All active torrents

**NFR-P8: qBittorrent status updates <5 seconds**
- Use 5-second polling interval
- TanStack Query refetchInterval

### qBittorrent API for Torrents

```
GET /api/v2/torrents/info
Query params:
  - filter: all|downloading|seeding|completed|paused|active|inactive|resumed|stalled|stalled_uploading|stalled_downloading|errored
  - sort: name|size|progress|dlspeed|upspeed|priority|eta|added_on|completion_on
  - reverse: true|false

Response (array):
{
  "hash": "8c212779b4abde7c6bc608571a69eb3a9ec4c28c",
  "name": "[SubGroup] Movie Name (2024) [1080p]",
  "size": 4294967296,
  "progress": 0.85,
  "dlspeed": 10485760,
  "upspeed": 524288,
  "eta": 600,
  "state": "downloading",
  "added_on": 1704067200,
  "completion_on": 0,
  "num_seeds": 10,
  "num_leechs": 5,
  "save_path": "/downloads/movies",
  "downloaded": 3650722201,
  "uploaded": 104857600,
  "ratio": 0.03
}

GET /api/v2/torrents/properties?hash={hash}
Response:
{
  "save_path": "/downloads/movies",
  "creation_date": 1704067200,
  "piece_size": 4194304,
  "comment": "...",
  "total_wasted": 0,
  "total_uploaded": 104857600,
  "total_downloaded": 3650722201,
  "peers": 15,
  "seeds": 10,
  "addition_date": 1704067200,
  "completion_date": -1,
  "created_by": "qBittorrent",
  "dl_speed_avg": 8388608,
  "up_speed_avg": 262144,
  "time_elapsed": 3600,
  "seeding_time": 0
}
```

### Backend Implementation

```go
// /apps/api/internal/qbittorrent/torrent.go
package qbittorrent

import "time"

type TorrentStatus string

const (
    StatusDownloading TorrentStatus = "downloading"
    StatusPaused      TorrentStatus = "paused"
    StatusSeeding     TorrentStatus = "seeding"
    StatusCompleted   TorrentStatus = "completed"
    StatusStalled     TorrentStatus = "stalled"
    StatusError       TorrentStatus = "error"
    StatusQueued      TorrentStatus = "queued"
    StatusChecking    TorrentStatus = "checking"
)

type Torrent struct {
    Hash         string        `json:"hash"`
    Name         string        `json:"name"`
    Size         int64         `json:"size"`
    Progress     float64       `json:"progress"`
    DownloadSpeed int64        `json:"downloadSpeed"`
    UploadSpeed  int64         `json:"uploadSpeed"`
    ETA          int64         `json:"eta"` // seconds, -1 if unknown
    Status       TorrentStatus `json:"status"`
    AddedOn      time.Time     `json:"addedOn"`
    CompletedOn  *time.Time    `json:"completedOn,omitempty"`
    Seeds        int           `json:"seeds"`
    Peers        int           `json:"peers"`
    Downloaded   int64         `json:"downloaded"`
    Uploaded     int64         `json:"uploaded"`
    Ratio        float64       `json:"ratio"`
    SavePath     string        `json:"savePath"`
}

type TorrentDetails struct {
    Torrent
    PieceSize     int64     `json:"pieceSize"`
    Comment       string    `json:"comment,omitempty"`
    CreatedBy     string    `json:"createdBy,omitempty"`
    CreationDate  time.Time `json:"creationDate"`
    TotalWasted   int64     `json:"totalWasted"`
    TimeElapsed   int64     `json:"timeElapsed"` // seconds
    SeedingTime   int64     `json:"seedingTime"` // seconds
    AvgDownSpeed  int64     `json:"avgDownSpeed"`
    AvgUpSpeed    int64     `json:"avgUpSpeed"`
}

// MapQBState maps qBittorrent state strings to our enum
func MapQBState(state string) TorrentStatus {
    switch state {
    case "downloading", "forcedDL", "metaDL":
        return StatusDownloading
    case "pausedDL", "pausedUP":
        return StatusPaused
    case "uploading", "forcedUP":
        return StatusSeeding
    case "stalledUP":
        return StatusCompleted
    case "stalledDL":
        return StatusStalled
    case "queuedDL", "queuedUP":
        return StatusQueued
    case "checkingDL", "checkingUP", "checkingResumeData":
        return StatusChecking
    case "error", "missingFiles":
        return StatusError
    default:
        return StatusDownloading
    }
}
```

```go
// Client extension - /apps/api/internal/qbittorrent/client.go

type TorrentsFilter string

const (
    FilterAll         TorrentsFilter = "all"
    FilterDownloading TorrentsFilter = "downloading"
    FilterCompleted   TorrentsFilter = "completed"
    FilterPaused      TorrentsFilter = "paused"
    FilterSeeding     TorrentsFilter = "seeding"
    FilterActive      TorrentsFilter = "active"
)

type TorrentsSort string

const (
    SortAddedOn    TorrentsSort = "added_on"
    SortName       TorrentsSort = "name"
    SortProgress   TorrentsSort = "progress"
    SortSize       TorrentsSort = "size"
)

type ListTorrentsOptions struct {
    Filter  TorrentsFilter
    Sort    TorrentsSort
    Reverse bool
}

func (c *Client) GetTorrents(ctx context.Context, opts *ListTorrentsOptions) ([]Torrent, error) {
    if err := c.ensureAuthenticated(ctx); err != nil {
        return nil, err
    }

    url := c.buildURL("/torrents/info")

    // Build query params
    query := url.Query()
    if opts != nil {
        if opts.Filter != "" {
            query.Set("filter", string(opts.Filter))
        }
        if opts.Sort != "" {
            query.Set("sort", string(opts.Sort))
        }
        if opts.Reverse {
            query.Set("reverse", "true")
        }
    }
    url.RawQuery = query.Encode()

    req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("get torrents failed: %w", err)
    }
    defer resp.Body.Close()

    var qbTorrents []qbTorrent
    if err := json.NewDecoder(resp.Body).Decode(&qbTorrents); err != nil {
        return nil, fmt.Errorf("decode torrents: %w", err)
    }

    torrents := make([]Torrent, len(qbTorrents))
    for i, qbt := range qbTorrents {
        torrents[i] = mapQBTorrent(qbt)
    }

    return torrents, nil
}

func (c *Client) GetTorrentDetails(ctx context.Context, hash string) (*TorrentDetails, error) {
    if err := c.ensureAuthenticated(ctx); err != nil {
        return nil, err
    }

    // Get basic torrent info
    torrents, err := c.GetTorrents(ctx, nil)
    if err != nil {
        return nil, err
    }

    var torrent *Torrent
    for _, t := range torrents {
        if t.Hash == hash {
            torrent = &t
            break
        }
    }
    if torrent == nil {
        return nil, fmt.Errorf("torrent not found: %s", hash)
    }

    // Get detailed properties
    url := c.buildURL("/torrents/properties")
    url.RawQuery = fmt.Sprintf("hash=%s", hash)

    req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
    if err != nil {
        return nil, err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("get torrent properties failed: %w", err)
    }
    defer resp.Body.Close()

    var props qbTorrentProperties
    if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
        return nil, fmt.Errorf("decode properties: %w", err)
    }

    return mapTorrentDetails(torrent, props), nil
}
```

### Service Layer

```go
// /apps/api/internal/services/download_service.go
package services

type DownloadServiceInterface interface {
    GetAllDownloads(ctx context.Context, sort string, order string) ([]qbittorrent.Torrent, error)
    GetDownloadDetails(ctx context.Context, hash string) (*qbittorrent.TorrentDetails, error)
}

type DownloadService struct {
    qbService QBittorrentServiceInterface
    logger    *slog.Logger
}

func NewDownloadService(qbService QBittorrentServiceInterface, logger *slog.Logger) *DownloadService {
    return &DownloadService{
        qbService: qbService,
        logger:    logger,
    }
}

func (s *DownloadService) GetAllDownloads(ctx context.Context, sort string, order string) ([]qbittorrent.Torrent, error) {
    config, err := s.qbService.GetConfig(ctx)
    if err != nil {
        return nil, err
    }

    if config.Host == "" {
        return nil, fmt.Errorf("qBittorrent not configured")
    }

    client := qbittorrent.NewClient(config, s.logger)

    opts := &qbittorrent.ListTorrentsOptions{
        Sort:    qbittorrent.TorrentsSort(sort),
        Reverse: order == "desc",
    }

    return client.GetTorrents(ctx, opts)
}

func (s *DownloadService) GetDownloadDetails(ctx context.Context, hash string) (*qbittorrent.TorrentDetails, error) {
    config, err := s.qbService.GetConfig(ctx)
    if err != nil {
        return nil, err
    }

    if config.Host == "" {
        return nil, fmt.Errorf("qBittorrent not configured")
    }

    client := qbittorrent.NewClient(config, s.logger)
    return client.GetTorrentDetails(ctx, hash)
}
```

### API Response Format

**List Downloads:**
```
GET /api/v1/downloads?sort=added_on&order=desc
```
Response:
```json
{
  "success": true,
  "data": [
    {
      "hash": "8c212779b4abde7c6bc608571a69eb3a9ec4c28c",
      "name": "[SubGroup] Movie Name (2024) [1080p].mkv",
      "size": 4294967296,
      "progress": 0.85,
      "downloadSpeed": 10485760,
      "uploadSpeed": 524288,
      "eta": 600,
      "status": "downloading",
      "addedOn": "2026-01-15T10:00:00Z",
      "seeds": 10,
      "peers": 5,
      "downloaded": 3650722201,
      "uploaded": 104857600,
      "ratio": 0.03,
      "savePath": "/downloads/movies"
    }
  ]
}
```

### Frontend Implementation

```tsx
// /apps/web/src/components/downloads/DownloadItem.tsx
interface DownloadItemProps {
  download: Download;
  expanded?: boolean;
  onToggleExpand: () => void;
}

export function DownloadItem({ download, expanded, onToggleExpand }: DownloadItemProps) {
  return (
    <Card className="mb-2">
      <CardContent className="p-4" onClick={onToggleExpand}>
        <div className="flex items-center gap-4">
          {/* Status Icon */}
          <StatusIcon status={download.status} />

          {/* Name and Progress */}
          <div className="flex-1 min-w-0">
            <p className="font-medium truncate">{download.name}</p>
            <Progress value={download.progress * 100} className="h-2 mt-1" />
            <div className="flex gap-4 text-xs text-muted-foreground mt-1">
              <span>{formatProgress(download.progress)}</span>
              <span>{formatSize(download.downloaded)} / {formatSize(download.size)}</span>
            </div>
          </div>

          {/* Speed and ETA */}
          <div className="text-right text-sm">
            {download.status === 'downloading' && (
              <>
                <p className="text-green-500">↓ {formatSpeed(download.downloadSpeed)}</p>
                <p className="text-muted-foreground">{formatETA(download.eta)}</p>
              </>
            )}
            {download.status === 'seeding' && (
              <p className="text-blue-500">↑ {formatSpeed(download.uploadSpeed)}</p>
            )}
            {download.status === 'completed' && (
              <p className="text-green-500">完成</p>
            )}
          </div>

          {/* Expand indicator */}
          <ChevronDown className={cn("h-4 w-4 transition-transform", expanded && "rotate-180")} />
        </div>

        {/* Expanded Details */}
        {expanded && <DownloadDetails hash={download.hash} />}
      </CardContent>
    </Card>
  );
}

// Helper functions
function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec >= 1048576) {
    return `${(bytesPerSec / 1048576).toFixed(1)} MB/s`;
  }
  return `${(bytesPerSec / 1024).toFixed(1)} KB/s`;
}

function formatETA(seconds: number): string {
  if (seconds < 0 || seconds === 8640000) return '∞';
  if (seconds < 60) return `${seconds}秒`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分鐘`;
  const hours = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  return `${hours}時${mins}分`;
}

function formatProgress(progress: number): string {
  return `${(progress * 100).toFixed(1)}%`;
}
```

```tsx
// /apps/web/src/routes/downloads.tsx
import { useQuery } from '@tanstack/react-query';
import { useState, useEffect } from 'react';
import { downloadService } from '@/services/downloadService';

export function DownloadsPage() {
  const [isVisible, setIsVisible] = useState(true);
  const [sortField, setSortField] = useState<string>('added_on');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  // Handle visibility change for polling
  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsVisible(document.visibilityState === 'visible');
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, []);

  const { data: downloads, isLoading, error } = useQuery({
    queryKey: ['downloads', sortField, sortOrder],
    queryFn: () => downloadService.getDownloads({ sort: sortField, order: sortOrder }),
    refetchInterval: isVisible ? 5000 : false, // NFR-P8: 5 second polling when visible
    refetchOnWindowFocus: true,
  });

  // ... render
}
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/qbittorrent/
├── torrent.go
└── torrent_test.go

/apps/api/internal/services/
├── download_service.go
└── download_service_test.go

/apps/api/internal/handlers/
├── download_handler.go
└── download_handler_test.go
```

**Frontend Files to Create:**
```
/apps/web/src/routes/
└── downloads.tsx

/apps/web/src/services/
└── downloadService.ts

/apps/web/src/components/downloads/
├── DownloadList.tsx
├── DownloadList.spec.tsx
├── DownloadItem.tsx
├── DownloadItem.spec.tsx
├── DownloadDetails.tsx
├── DownloadDetails.spec.tsx
├── StatusIcon.tsx
└── index.ts
```

### Testing Strategy

**Backend Tests:**
1. Torrent status mapping tests
2. Client torrent listing tests
3. Service tests with mock client
4. Handler tests

**Frontend Tests:**
1. DownloadItem render tests
2. Speed/ETA formatting tests
3. Polling behavior tests

**Coverage Targets:**
- Backend qbittorrent package: ≥80%
- Backend services: ≥80%
- Frontend components: ≥70%

### Error Codes

- `QB_NOT_CONFIGURED` - qBittorrent not configured
- `QB_CONNECTION_FAILED` - Cannot connect to qBittorrent
- `QB_TORRENT_NOT_FOUND` - Torrent hash not found

### Dependencies

**Story Dependencies:**
- Story 4-1 (Connection Configuration) - Must be completed first

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.2]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR29]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-P8]
- [Source: project-context.md#Rule-5-TanStack-Query]
- [qBittorrent API - torrents/info](https://github.com/qbittorrent/qBittorrent/wiki/WebUI-API-(qBittorrent-4.1)#get-torrent-list)

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
