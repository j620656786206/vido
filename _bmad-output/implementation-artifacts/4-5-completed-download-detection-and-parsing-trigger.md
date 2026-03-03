# Story 4.5: Completed Download Detection and Parsing Trigger

Status: done

## Story

As a **media collector**,
I want **completed downloads to automatically trigger parsing**,
So that **new media appears in my library without manual action**.

## Acceptance Criteria

1. **AC1: Completion Detection**
   - Given qBittorrent reports a torrent as complete
   - When the next polling cycle detects the completion
   - Then the system automatically queues the file for parsing
   - And the download shows status: "Parsing..."

2. **AC2: Successful Parsing Flow**
   - Given parsing completes successfully
   - When metadata is retrieved
   - Then the media appears in "Recently Added"
   - And a success notification is shown

3. **AC3: Failed Parsing Handling**
   - Given parsing fails
   - When errors occur
   - Then the download shows: "Parsing failed - Manual action needed"
   - And links to manual search options

4. **AC4: Non-Blocking Operation**
   - Given a download completes
   - When parsing is triggered
   - Then the user can continue browsing without interruption
   - And multiple files can be parsed concurrently

5. **AC5: Duplicate Detection**
   - Given a torrent completes
   - When the same file was already parsed previously
   - Then the system detects the duplicate
   - And shows: "Already in library" instead of re-parsing

## Tasks / Subtasks

- [x] Task 1: Create Completion Detection Service (AC: 1, 5)
  - [x] 1.1: Create `/apps/api/internal/services/completion_detector.go`
  - [x] 1.2: Define `CompletionDetectorInterface`
  - [x] 1.3: Track previously seen completed torrents (prevent re-trigger)
  - [x] 1.4: Implement `DetectNewCompletions(ctx, currentTorrents) []Torrent`
  - [x] 1.5: Check if file already exists in library (duplicate detection)
  - [x] 1.6: Write service tests

- [x] Task 2: Create Parse Queue Types (AC: 1, 2, 3)
  - [x] 2.1: Create `/apps/api/internal/models/parse_job.go`
  - [x] 2.2: Define `ParseJob` struct (ID, TorrentHash, FilePath, Status, CreatedAt, CompletedAt, Error)
  - [x] 2.3: Define `ParseJobStatus` enum (pending, processing, completed, failed, skipped)
  - [x] 2.4: Write type tests

- [x] Task 3: Create Parse Job Repository (AC: 1, 2, 3)
  - [x] 3.1: Create `/apps/api/internal/repository/parse_job_repository.go`
  - [x] 3.2: Add migration for `parse_jobs` table
  - [x] 3.3: Implement `CreateJob(ctx, job) error`
  - [x] 3.4: Implement `GetJobByTorrentHash(ctx, hash) (*ParseJob, error)`
  - [x] 3.5: Implement `UpdateJobStatus(ctx, id, status, error) error`
  - [x] 3.6: Implement `GetPendingJobs(ctx) ([]ParseJob, error)`
  - [x] 3.7: Write repository tests

- [x] Task 4: Create Parse Queue Service (AC: 1, 2, 3, 4)
  - [x] 4.1: Create `/apps/api/internal/services/parse_queue_service.go`
  - [x] 4.2: Define `ParseQueueServiceInterface`
  - [x] 4.3: Implement `QueueParseJob(ctx, torrent) (*ParseJob, error)`
  - [x] 4.4: Implement `ProcessNextJob(ctx) error` - run parsing pipeline
  - [x] 4.5: Implement `GetJobStatus(ctx, jobId) (*ParseJob, error)`
  - [x] 4.6: Integrate with Epic 3 parsing pipeline
  - [x] 4.7: Write service tests

- [x] Task 5: Create Background Worker (AC: 4)
  - [x] 5.1: Create `/apps/api/internal/workers/parse_worker.go`
  - [x] 5.2: Implement worker pool (3-5 concurrent workers per ARCH-5)
  - [x] 5.3: Poll for new completed downloads
  - [x] 5.4: Process parse queue
  - [x] 5.5: Handle errors with retry logic
  - [x] 5.6: Write worker tests

- [x] Task 6: Create Parse Status API (AC: 1, 2, 3)
  - [x] 6.1: Create `GET /api/v1/downloads/:hash/parse-status` endpoint
  - [x] 6.2: Return parse job status for a torrent
  - [x] 6.3: Create `GET /api/v1/parse-jobs` endpoint for all jobs
  - [x] 6.4: Add Swagger documentation
  - [x] 6.5: Write handler tests

- [x] Task 7: Extend Download Response with Parse Status (AC: 1, 2, 3)
  - [x] 7.1: Add `parseStatus` field to download response
  - [x] 7.2: Include parse job info (status, error message)
  - [x] 7.3: Update Swagger documentation

- [x] Task 8: Create Parse Status UI Components (AC: 1, 2, 3)
  - [x] 8.1: Create `/apps/web/src/components/downloads/DownloadParseStatusBadge.tsx`
  - [x] 8.2: Show "Parsing...", "Parsed", "Failed", "In Library" states
  - [x] 8.3: Create `/apps/web/src/components/downloads/ParseFailedActions.tsx`
  - [x] 8.4: Show retry and manual search buttons on failure
  - [x] 8.5: Write component tests

- [x] Task 9: Update Download Item with Parse Status (AC: 1, 2, 3)
  - [x] 9.1: Integrate ParseStatusBadge in DownloadItem
  - [x] 9.2: Show manual action button when parsing failed
  - [x] 9.3: Link to manual search (from Epic 3)

- [x] Task 10: Create Parse Completion Notification (AC: 2)
  - [x] 10.1: Create `/apps/web/src/components/notifications/ParseCompleteToast.tsx`
  - [x] 10.2: Show notification when parsing completes
  - [x] 10.3: Include media poster and title
  - [x] 10.4: Write component tests

- [x] Task 11: E2E Tests (AC: all)
  - [x] 11.1: Create `/tests/e2e/parse-trigger.spec.ts`
  - [x] 11.2: Test completion detection flow
  - [x] 11.3: Test successful parsing notification
  - [x] 11.4: Test failed parsing with manual action
  - [x] 11.5: Test duplicate detection

## Dev Notes

### Architecture Requirements

**FR32: Detect completed downloads and trigger parsing**
- Automatic detection of completion
- Queue for parsing
- Non-blocking operation

### System Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                                                                     │
│  qBittorrent                                                        │
│  ┌─────────────┐                                                    │
│  │ Torrent     │──────── Polling (5s) ─────────▶                   │
│  │ Completed   │                                 │                  │
│  └─────────────┘                                 ▼                  │
│                                          ┌──────────────┐           │
│                                          │ Completion   │           │
│                                          │ Detector     │           │
│                                          └──────┬───────┘           │
│                                                 │                   │
│                        ┌────────────────────────┼───────────────┐   │
│                        │                        │               │   │
│                        ▼                        ▼               ▼   │
│                  ┌──────────┐            ┌──────────┐    ┌──────────┐
│                  │ Already  │            │ New      │    │ Already  │
│                  │ Parsed   │            │ Complete │    │ In       │
│                  │ (skip)   │            │          │    │ Library  │
│                  └──────────┘            └────┬─────┘    └──────────┘
│                                               │                     │
│                                               ▼                     │
│                                        ┌──────────────┐             │
│                                        │ Parse Queue  │             │
│                                        │ Service      │             │
│                                        └──────┬───────┘             │
│                                               │                     │
│                                               ▼                     │
│                                        ┌──────────────┐             │
│                                        │ Parse Worker │             │
│                                        │ Pool (3-5)   │             │
│                                        └──────┬───────┘             │
│                                               │                     │
│                      ┌────────────────────────┼────────────────┐    │
│                      │                        │                │    │
│                      ▼                        ▼                ▼    │
│               ┌──────────┐            ┌──────────┐      ┌──────────┐
│               │ Success  │            │ Fallback │      │ Failed   │
│               │ TMDb     │            │ Chain    │      │ Manual   │
│               └────┬─────┘            └────┬─────┘      │ Required │
│                    │                       │            └──────────┘
│                    ▼                       ▼                        │
│               ┌─────────────────────────────────┐                   │
│               │       Media Library             │                   │
│               │       (Recently Added)          │                   │
│               └─────────────────────────────────┘                   │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### Database Schema

```sql
-- Migration: create_parse_jobs_table.sql
CREATE TABLE IF NOT EXISTS parse_jobs (
    id TEXT PRIMARY KEY,
    torrent_hash TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- pending, processing, completed, failed, skipped
    media_id TEXT, -- Reference to created media (if successful)
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    UNIQUE(torrent_hash)
);

CREATE INDEX idx_parse_jobs_status ON parse_jobs(status);
CREATE INDEX idx_parse_jobs_torrent_hash ON parse_jobs(torrent_hash);
```

### Backend Implementation

```go
// /apps/api/internal/models/parse_job.go
package models

import "time"

type ParseJobStatus string

const (
    ParseJobPending    ParseJobStatus = "pending"
    ParseJobProcessing ParseJobStatus = "processing"
    ParseJobCompleted  ParseJobStatus = "completed"
    ParseJobFailed     ParseJobStatus = "failed"
    ParseJobSkipped    ParseJobStatus = "skipped" // Duplicate or already in library
)

type ParseJob struct {
    ID           string         `json:"id"`
    TorrentHash  string         `json:"torrentHash"`
    FilePath     string         `json:"filePath"`
    FileName     string         `json:"fileName"`
    Status       ParseJobStatus `json:"status"`
    MediaID      *string        `json:"mediaId,omitempty"` // Set when completed
    ErrorMessage *string        `json:"errorMessage,omitempty"`
    RetryCount   int            `json:"retryCount"`
    CreatedAt    time.Time      `json:"createdAt"`
    UpdatedAt    time.Time      `json:"updatedAt"`
    CompletedAt  *time.Time     `json:"completedAt,omitempty"`
}
```

```go
// /apps/api/internal/services/completion_detector.go
package services

import (
    "context"
    "log/slog"
    "sync"

    "vido/apps/api/internal/qbittorrent"
    "vido/apps/api/internal/repository"
)

type CompletionDetectorInterface interface {
    DetectNewCompletions(ctx context.Context, torrents []qbittorrent.Torrent) []qbittorrent.Torrent
}

type CompletionDetector struct {
    parseJobRepo repository.ParseJobRepositoryInterface
    mediaRepo    repository.MediaRepositoryInterface
    logger       *slog.Logger
    seenHashes   map[string]bool
    mu           sync.RWMutex
}

func NewCompletionDetector(
    parseJobRepo repository.ParseJobRepositoryInterface,
    mediaRepo repository.MediaRepositoryInterface,
    logger *slog.Logger,
) *CompletionDetector {
    return &CompletionDetector{
        parseJobRepo: parseJobRepo,
        mediaRepo:    mediaRepo,
        logger:       logger,
        seenHashes:   make(map[string]bool),
    }
}

func (d *CompletionDetector) DetectNewCompletions(ctx context.Context, torrents []qbittorrent.Torrent) []qbittorrent.Torrent {
    var newCompletions []qbittorrent.Torrent

    for _, t := range torrents {
        // Only look at completed torrents
        if t.Status != qbittorrent.StatusCompleted {
            continue
        }

        // Check if we've already processed this hash
        d.mu.RLock()
        seen := d.seenHashes[t.Hash]
        d.mu.RUnlock()

        if seen {
            continue
        }

        // Check if already has a parse job
        existingJob, _ := d.parseJobRepo.GetByTorrentHash(ctx, t.Hash)
        if existingJob != nil {
            d.mu.Lock()
            d.seenHashes[t.Hash] = true
            d.mu.Unlock()
            continue
        }

        // Check if file already in library (duplicate detection)
        existingMedia, _ := d.mediaRepo.FindByFilePath(ctx, t.SavePath)
        if existingMedia != nil {
            d.logger.Info("File already in library, skipping",
                "hash", t.Hash,
                "path", t.SavePath,
            )
            d.mu.Lock()
            d.seenHashes[t.Hash] = true
            d.mu.Unlock()
            continue
        }

        // New completion detected
        newCompletions = append(newCompletions, t)

        d.mu.Lock()
        d.seenHashes[t.Hash] = true
        d.mu.Unlock()
    }

    return newCompletions
}
```

```go
// /apps/api/internal/services/parse_queue_service.go
package services

import (
    "context"
    "fmt"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "vido/apps/api/internal/models"
    "vido/apps/api/internal/qbittorrent"
    "vido/apps/api/internal/repository"
)

type ParseQueueServiceInterface interface {
    QueueParseJob(ctx context.Context, torrent *qbittorrent.Torrent) (*models.ParseJob, error)
    ProcessNextJob(ctx context.Context) error
    GetJobStatus(ctx context.Context, torrentHash string) (*models.ParseJob, error)
    RetryJob(ctx context.Context, jobID string) error
}

type ParseQueueService struct {
    parseJobRepo repository.ParseJobRepositoryInterface
    parserService ParserServiceInterface
    metadataService MetadataServiceInterface
    mediaRepo repository.MediaRepositoryInterface
    logger *slog.Logger
}

func NewParseQueueService(
    parseJobRepo repository.ParseJobRepositoryInterface,
    parserService ParserServiceInterface,
    metadataService MetadataServiceInterface,
    mediaRepo repository.MediaRepositoryInterface,
    logger *slog.Logger,
) *ParseQueueService {
    return &ParseQueueService{
        parseJobRepo:    parseJobRepo,
        parserService:   parserService,
        metadataService: metadataService,
        mediaRepo:       mediaRepo,
        logger:          logger,
    }
}

func (s *ParseQueueService) QueueParseJob(ctx context.Context, torrent *qbittorrent.Torrent) (*models.ParseJob, error) {
    job := &models.ParseJob{
        ID:          uuid.New().String(),
        TorrentHash: torrent.Hash,
        FilePath:    torrent.SavePath,
        FileName:    torrent.Name,
        Status:      models.ParseJobPending,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }

    if err := s.parseJobRepo.Create(ctx, job); err != nil {
        return nil, fmt.Errorf("create parse job: %w", err)
    }

    s.logger.Info("Parse job queued",
        "job_id", job.ID,
        "torrent_hash", torrent.Hash,
        "filename", torrent.Name,
    )

    return job, nil
}

func (s *ParseQueueService) ProcessNextJob(ctx context.Context) error {
    // Get next pending job
    jobs, err := s.parseJobRepo.GetPending(ctx, 1)
    if err != nil {
        return err
    }
    if len(jobs) == 0 {
        return nil // No pending jobs
    }

    job := jobs[0]

    // Mark as processing
    if err := s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobProcessing, ""); err != nil {
        return err
    }

    s.logger.Info("Processing parse job",
        "job_id", job.ID,
        "filename", job.FileName,
    )

    // Run parsing pipeline (from Epic 3)
    parseResult, err := s.parserService.Parse(ctx, job.FileName)
    if err != nil {
        s.logger.Error("Parsing failed",
            "job_id", job.ID,
            "error", err,
        )
        errMsg := err.Error()
        s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobFailed, errMsg)
        return nil // Continue processing other jobs
    }

    // Fetch metadata
    metadata, err := s.metadataService.FetchMetadata(ctx, parseResult)
    if err != nil {
        s.logger.Error("Metadata fetch failed",
            "job_id", job.ID,
            "error", err,
        )
        errMsg := err.Error()
        s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobFailed, errMsg)
        return nil
    }

    // Create media entry
    media, err := s.mediaRepo.Create(ctx, &models.Media{
        Title:     metadata.Title,
        Year:      metadata.Year,
        TMDbID:    metadata.TMDbID,
        PosterURL: metadata.PosterURL,
        FilePath:  job.FilePath,
        FileName:  job.FileName,
        MediaType: metadata.MediaType,
    })
    if err != nil {
        s.logger.Error("Media creation failed",
            "job_id", job.ID,
            "error", err,
        )
        errMsg := err.Error()
        s.parseJobRepo.UpdateStatus(ctx, job.ID, models.ParseJobFailed, errMsg)
        return nil
    }

    // Mark as completed
    mediaID := media.ID
    job.MediaID = &mediaID
    job.Status = models.ParseJobCompleted
    now := time.Now()
    job.CompletedAt = &now

    if err := s.parseJobRepo.Update(ctx, job); err != nil {
        return err
    }

    s.logger.Info("Parse job completed successfully",
        "job_id", job.ID,
        "media_id", media.ID,
        "title", metadata.Title,
    )

    return nil
}

func (s *ParseQueueService) GetJobStatus(ctx context.Context, torrentHash string) (*models.ParseJob, error) {
    return s.parseJobRepo.GetByTorrentHash(ctx, torrentHash)
}

func (s *ParseQueueService) RetryJob(ctx context.Context, jobID string) error {
    job, err := s.parseJobRepo.GetByID(ctx, jobID)
    if err != nil {
        return err
    }

    if job.Status != models.ParseJobFailed {
        return fmt.Errorf("can only retry failed jobs")
    }

    job.Status = models.ParseJobPending
    job.RetryCount++
    job.ErrorMessage = nil
    job.UpdatedAt = time.Now()

    return s.parseJobRepo.Update(ctx, job)
}
```

```go
// /apps/api/internal/workers/parse_worker.go
package workers

import (
    "context"
    "log/slog"
    "sync"
    "time"

    "vido/apps/api/internal/services"
)

type ParseWorker struct {
    downloadService    services.DownloadServiceInterface
    completionDetector services.CompletionDetectorInterface
    parseQueueService  services.ParseQueueServiceInterface
    logger             *slog.Logger
    workerCount        int
    pollInterval       time.Duration
    wg                 sync.WaitGroup
    stop               chan struct{}
}

func NewParseWorker(
    downloadService services.DownloadServiceInterface,
    completionDetector services.CompletionDetectorInterface,
    parseQueueService services.ParseQueueServiceInterface,
    logger *slog.Logger,
) *ParseWorker {
    return &ParseWorker{
        downloadService:    downloadService,
        completionDetector: completionDetector,
        parseQueueService:  parseQueueService,
        logger:             logger,
        workerCount:        3, // ARCH-5: 3-5 goroutines
        pollInterval:       5 * time.Second,
        stop:               make(chan struct{}),
    }
}

func (w *ParseWorker) Start(ctx context.Context) {
    // Start completion detector (polls qBittorrent)
    w.wg.Add(1)
    go w.runCompletionDetector(ctx)

    // Start parse workers
    for i := 0; i < w.workerCount; i++ {
        w.wg.Add(1)
        go w.runParseWorker(ctx, i)
    }

    w.logger.Info("Parse workers started",
        "worker_count", w.workerCount,
        "poll_interval", w.pollInterval,
    )
}

func (w *ParseWorker) Stop() {
    close(w.stop)
    w.wg.Wait()
    w.logger.Info("Parse workers stopped")
}

func (w *ParseWorker) runCompletionDetector(ctx context.Context) {
    defer w.wg.Done()

    ticker := time.NewTicker(w.pollInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-w.stop:
            return
        case <-ticker.C:
            w.checkForCompletions(ctx)
        }
    }
}

func (w *ParseWorker) checkForCompletions(ctx context.Context) {
    // Get current downloads
    downloads, err := w.downloadService.GetAllDownloads(ctx, "completed", "", "")
    if err != nil {
        w.logger.Debug("Failed to get downloads for completion check", "error", err)
        return
    }

    // Detect new completions
    newCompletions := w.completionDetector.DetectNewCompletions(ctx, downloads)

    // Queue parse jobs
    for _, torrent := range newCompletions {
        _, err := w.parseQueueService.QueueParseJob(ctx, &torrent)
        if err != nil {
            w.logger.Error("Failed to queue parse job",
                "hash", torrent.Hash,
                "error", err,
            )
        }
    }
}

func (w *ParseWorker) runParseWorker(ctx context.Context, workerID int) {
    defer w.wg.Done()

    ticker := time.NewTicker(time.Second) // Check for jobs every second
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-w.stop:
            return
        case <-ticker.C:
            if err := w.parseQueueService.ProcessNextJob(ctx); err != nil {
                w.logger.Error("Parse worker error",
                    "worker_id", workerID,
                    "error", err,
                )
            }
        }
    }
}
```

### API Response Format

**Download with Parse Status:**
```
GET /api/v1/downloads
```
Response:
```json
{
  "success": true,
  "data": [
    {
      "hash": "abc123",
      "name": "[SubGroup] Movie (2024).mkv",
      "status": "completed",
      "progress": 1.0,
      "parseStatus": {
        "status": "processing",
        "message": "正在解析中..."
      }
    },
    {
      "hash": "def456",
      "name": "[Another] Show S01E01.mkv",
      "status": "completed",
      "progress": 1.0,
      "parseStatus": {
        "status": "completed",
        "mediaId": "media-789",
        "message": "已加入媒體庫"
      }
    },
    {
      "hash": "ghi789",
      "name": "Unknown.File.mkv",
      "status": "completed",
      "progress": 1.0,
      "parseStatus": {
        "status": "failed",
        "message": "無法識別檔案名稱",
        "canRetry": true
      }
    }
  ]
}
```

### Frontend Implementation

```tsx
// /apps/web/src/components/downloads/ParseStatusBadge.tsx
interface ParseStatusBadgeProps {
  parseStatus?: ParseStatus;
}

export function ParseStatusBadge({ parseStatus }: ParseStatusBadgeProps) {
  if (!parseStatus) return null;

  const configs: Record<ParseStatus['status'], {
    label: string;
    icon: React.ComponentType<{ className?: string }>;
    variant: 'default' | 'secondary' | 'destructive' | 'outline';
  }> = {
    pending: {
      label: '等待解析',
      icon: Clock,
      variant: 'outline',
    },
    processing: {
      label: '解析中...',
      icon: Loader2,
      variant: 'secondary',
    },
    completed: {
      label: '已入庫',
      icon: Check,
      variant: 'default',
    },
    failed: {
      label: '解析失敗',
      icon: AlertCircle,
      variant: 'destructive',
    },
    skipped: {
      label: '已在庫中',
      icon: Library,
      variant: 'outline',
    },
  };

  const config = configs[parseStatus.status];
  const Icon = config.icon;

  return (
    <Badge variant={config.variant} className="gap-1">
      <Icon className={cn(
        "h-3 w-3",
        parseStatus.status === 'processing' && "animate-spin"
      )} />
      {config.label}
    </Badge>
  );
}
```

```tsx
// /apps/web/src/components/downloads/ParseFailedActions.tsx
interface ParseFailedActionsProps {
  torrentHash: string;
  fileName: string;
}

export function ParseFailedActions({ torrentHash, fileName }: ParseFailedActionsProps) {
  const queryClient = useQueryClient();

  const retryMutation = useMutation({
    mutationFn: () => downloadService.retryParse(torrentHash),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['downloads'] });
      toast.success('已重新加入解析佇列');
    },
    onError: () => {
      toast.error('重試失敗');
    },
  });

  return (
    <div className="flex gap-2 mt-2">
      <Button
        size="sm"
        variant="outline"
        onClick={() => retryMutation.mutate()}
        disabled={retryMutation.isPending}
      >
        <RefreshCw className={cn(
          "h-4 w-4 mr-1",
          retryMutation.isPending && "animate-spin"
        )} />
        重試
      </Button>
      <Link to="/search" search={{ filename: fileName }}>
        <Button size="sm" variant="outline">
          <Search className="h-4 w-4 mr-1" />
          手動搜尋
        </Button>
      </Link>
    </div>
  );
}
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/models/
└── parse_job.go

/apps/api/internal/repository/
├── parse_job_repository.go
└── parse_job_repository_test.go

/apps/api/internal/services/
├── completion_detector.go
├── completion_detector_test.go
├── parse_queue_service.go
└── parse_queue_service_test.go

/apps/api/internal/workers/
├── parse_worker.go
└── parse_worker_test.go

/apps/api/internal/handlers/
└── download_handler.go (extend with parse status)

/apps/api/migrations/
└── XXX_create_parse_jobs_table.sql
```

**Frontend Files to Create:**
```
/apps/web/src/components/downloads/
├── ParseStatusBadge.tsx
├── ParseStatusBadge.spec.tsx
├── ParseFailedActions.tsx
├── ParseFailedActions.spec.tsx
└── index.ts (update exports)

/apps/web/src/components/notifications/
├── ParseCompleteToast.tsx
└── ParseCompleteToast.spec.tsx
```

### Testing Strategy

**Backend Tests:**
1. Completion detector tests (new detection, duplicate skip)
2. Parse queue service tests (queue, process, retry)
3. Parse worker tests (mock services)
4. Repository tests

**Frontend Tests:**
1. ParseStatusBadge render tests
2. ParseFailedActions interaction tests

**E2E Tests:**
1. Full completion → parse → library flow (with mock qBittorrent)
2. Failed parse → retry flow
3. Duplicate detection

**Coverage Targets:**
- Backend workers: ≥80%
- Backend services: ≥80%
- Frontend components: ≥70%

### Error Codes

- `PARSE_JOB_NOT_FOUND` - Parse job not found
- `PARSE_QUEUE_FULL` - Too many pending jobs
- `PARSE_RETRY_FAILED` - Cannot retry (not in failed state)

### Dependencies

**Story Dependencies:**
- Story 4-2 (Download Monitoring) - Downloads API
- Story 3-2 (AI Fansub Parsing) - Parsing pipeline
- Story 3-3 (Multi-Source Fallback) - Metadata retrieval
- Story 2-6 (Media Entity) - Media storage

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-4.5]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR32]
- [Source: _bmad-output/planning-artifacts/architecture.md#ARCH-5]
- [Source: project-context.md#Rule-4-Layered-Architecture]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List

**Backend - New Files:**
- `apps/api/internal/models/parse_job.go` — ParseJob model and status enum
- `apps/api/internal/models/parse_job_test.go` — Model tests
- `apps/api/internal/repository/parse_job_repository.go` — ParseJob repository (CRUD)
- `apps/api/internal/repository/parse_job_repository_test.go` — Repository tests
- `apps/api/internal/services/completion_detector.go` — Detects newly completed downloads
- `apps/api/internal/services/completion_detector_test.go` — Detector tests
- `apps/api/internal/services/parse_queue_service.go` — Parse queue management
- `apps/api/internal/services/parse_queue_service_test.go` — Queue service tests
- `apps/api/internal/workers/parse_worker.go` — Background worker pool
- `apps/api/internal/workers/parse_worker_test.go` — Worker tests
- `apps/api/internal/handlers/parse_job_handler.go` — Parse job API endpoints
- `apps/api/internal/handlers/parse_job_handler_test.go` — Handler tests
- `apps/api/internal/database/migrations/013_create_parse_jobs_table.go` — DB migration

**Backend - Modified Files:**
- `apps/api/internal/handlers/download_handler.go` — Added parse status enrichment
- `apps/api/internal/handlers/download_handler_test.go` — Enrichment tests
- `apps/api/internal/repository/interfaces.go` — Added ParseJobRepositoryInterface
- `apps/api/internal/repository/registry.go` — Added ParseJobs to registry
- `apps/api/internal/repository/movie_repository.go` — Added FindByFilePath, fixed Create() columns
- `apps/api/internal/services/movie_service_test.go` — Added FindByFilePath mock

**Frontend - New Files:**
- `apps/web/src/components/downloads/DownloadParseStatusBadge.tsx` — Parse status badge
- `apps/web/src/components/downloads/DownloadParseStatusBadge.spec.tsx` — Badge tests
- `apps/web/src/components/downloads/ParseFailedActions.tsx` — Retry/manual search actions
- `apps/web/src/components/downloads/ParseFailedActions.spec.tsx` — Action tests
- `apps/web/src/components/notifications/ParseCompleteToast.tsx` — Parse completion toast
- `apps/web/src/components/notifications/ParseCompleteToast.spec.tsx` — Toast tests

**Frontend - Modified Files:**
- `apps/web/src/components/downloads/DownloadItem.tsx` — Integrated parse status badge
- `apps/web/src/components/downloads/DownloadItem.spec.tsx` — Added parse status tests
- `apps/web/src/services/downloadService.ts` — Added parse status types, URL encoding

**E2E Tests:**
- `tests/e2e/parse-trigger.spec.ts` — Parse trigger E2E tests

### Change Log

- **2026-03-03** (Code Review): Fixed H1 MovieRepository.Create() missing columns (file_path, parse_status, metadata_source, vote_average). Fixed H3 seenHashes memory leak with maxSeenHashes cap. Fixed H4 swallowed UpdateStatus errors. Fixed H5 SavePath→full file path in CompletionDetector and QueueParseJob. Added M2 sentinel errors (ErrJobNotRetryable, ErrMaxRetriesReached). Added M3 MaxRetryAttempts limit. Fixed M4 weak test assertions (toBeTruthy→toBeInTheDocument). Fixed M5 conditional manual search button render. Fixed M6 URL parameter encoding. Fixed M7 DB errors now skip torrent instead of treating as new.

### Technical Debt (for Epic 4 Retrospective)

The following issues were identified during code review but deferred as out-of-scope:

- **[TD-H2] TV shows stored as Movie** — `parse_queue_service.go:146` always creates `models.Movie` regardless of `mediaType`. When Series entity support is fully built out, `ProcessNextJob` should branch on mediaType to create the correct entity. Low impact now since TV parsing pipeline is not yet active.
- **[TD-M1] N+1 query in download parse status enrichment** — `download_handler.go:81-97` calls `GetJobStatus` individually per completed torrent. Should add `GetByTorrentHashes(ctx, []string{...})` batch method to `ParseJobRepositoryInterface` when download volume justifies the optimization.
- **[TD-M8] E2E missing core polling flow test** — `parse-trigger.spec.ts` only tests static states. No test for the critical scenario: download transitions from `downloading` → `completed` during polling and parse badge appears. Requires mock polling infrastructure or WebSocket support.
