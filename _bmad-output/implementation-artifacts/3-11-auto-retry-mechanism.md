# Story 3.11: Auto-Retry Mechanism

Status: done

## Story

As a **media collector**,
I want the **system to automatically retry when sources are temporarily unavailable**,
So that **temporary failures don't require my intervention**.

## Acceptance Criteria

1. **AC1: Automatic Retry on Temporary Errors**
   - Given a metadata source returns a temporary error (timeout, rate limit, network error)
   - When the error is detected
   - Then the system automatically queues a retry
   - And uses exponential backoff: 1s → 2s → 4s → 8s (NFR-R5)

2. **AC2: Maximum Retry Attempts**
   - Given all retries fail
   - When the maximum attempts (4) are reached
   - Then the file is marked for manual intervention
   - And the user is notified via the activity monitor

3. **AC3: Automatic Recovery**
   - Given the source recovers
   - When the retry succeeds
   - Then the metadata is applied
   - And the file status updates automatically
   - And a success notification is shown

4. **AC4: Retry Queue Visibility**
   - Given files are queued for retry
   - When viewing the activity monitor
   - Then the user sees pending retries with next attempt time
   - And can manually trigger immediate retry or cancel

## Tasks / Subtasks

- [x] Task 1: Create Retry Queue System (AC: 1, 2)
  - [x] 1.1: Create `/apps/api/internal/retry/queue.go`
  - [x] 1.2: Define `RetryItem` struct (task ID, attempt count, next attempt time)
  - [x] 1.3: Implement priority queue with exponential backoff scheduling
  - [x] 1.4: Add max retry limit (4 attempts)
  - [x] 1.5: Write queue tests

- [x] Task 2: Create Retry Scheduler (AC: 1, 3)
  - [x] 2.1: Create `/apps/api/internal/retry/scheduler.go`
  - [x] 2.2: Implement background goroutine for retry execution
  - [x] 2.3: Handle retry success/failure callbacks
  - [x] 2.4: Integrate with parse service
  - [x] 2.5: Write scheduler tests

- [x] Task 3: Implement Exponential Backoff (AC: 1)
  - [x] 3.1: Create `/apps/api/internal/retry/backoff.go`
  - [x] 3.2: Implement `CalculateBackoff()` with 1s → 2s → 4s → 8s pattern
  - [x] 3.3: Add jitter to prevent thundering herd
  - [x] 3.4: Write backoff tests

- [x] Task 4: Create Retry Repository (AC: 1, 2, 4)
  - [x] 4.1: Create database migration for `retry_queue` table
  - [x] 4.2: Create `/apps/api/internal/repository/retry_repository.go`
  - [x] 4.3: Implement `Add()`, `GetPending()`, `Update()`, `Delete()`
  - [x] 4.4: Add index on `next_attempt_at`
  - [x] 4.5: Write repository tests

- [x] Task 5: Create Retry Service (AC: 1, 2, 3, 4)
  - [x] 5.1: Create `/apps/api/internal/services/retry_service.go`
  - [x] 5.2: Define `RetryServiceInterface`
  - [x] 5.3: Implement `QueueRetry()` method
  - [x] 5.4: Implement `CancelRetry()` method
  - [x] 5.5: Implement `TriggerImmediate()` method
  - [x] 5.6: Implement `GetPendingRetries()` method
  - [x] 5.7: Write service tests

- [x] Task 6: Integrate with Fallback Orchestrator (AC: 1, 3)
  - [x] 6.1: Update orchestrator to detect retryable errors
  - [x] 6.2: Queue retry on temporary errors
  - [x] 6.3: Skip retry on permanent errors (not found, invalid)
  - [x] 6.4: Emit retry events for UI updates
  - [x] 6.5: Write integration tests

- [x] Task 7: Create Retry API Endpoints (AC: 4)
  - [x] 7.1: Create `GET /api/v1/retry/pending` endpoint
  - [x] 7.2: Create `POST /api/v1/retry/{id}/trigger` endpoint
  - [x] 7.3: Create `DELETE /api/v1/retry/{id}` endpoint
  - [x] 7.4: Write handler tests

- [x] Task 8: Create Retry Queue UI Component (AC: 4)
  - [x] 8.1: Create `RetryQueuePanel.tsx` component
  - [x] 8.2: Display pending retries with countdown timer
  - [x] 8.3: Add "Retry Now" button for each item
  - [x] 8.4: Add "Cancel" button for each item
  - [x] 8.5: Integrate with Activity Monitor
  - [x] 8.6: Write component tests

- [x] Task 9: Create Retry Notifications (AC: 2, 3)
  - [x] 9.1: Show toast when retry succeeds
  - [x] 9.2: Show toast when all retries exhausted
  - [x] 9.3: Add to Activity Monitor log
  - [x] 9.4: Write notification tests

## Dev Notes

### Architecture Requirements

**FR25: Auto-retry when sources unavailable**
- Automatic retry on transient errors
- No user intervention needed for temporary issues

**NFR-R5: Exponential backoff retry**
- Pattern: 1s → 2s → 4s → 8s
- Maximum 4 attempts
- Jitter to avoid thundering herd

### Retry Queue Types

```go
// RetryItem represents a task queued for retry
type RetryItem struct {
    ID            string         `json:"id" db:"id"`
    TaskID        string         `json:"taskId" db:"task_id"`
    TaskType      string         `json:"taskType" db:"task_type"`      // "parse", "metadata_fetch"
    Payload       json.RawMessage `json:"payload" db:"payload"`        // Task-specific data
    AttemptCount  int            `json:"attemptCount" db:"attempt_count"`
    MaxAttempts   int            `json:"maxAttempts" db:"max_attempts"`
    LastError     string         `json:"lastError,omitempty" db:"last_error"`
    NextAttemptAt time.Time      `json:"nextAttemptAt" db:"next_attempt_at"`
    CreatedAt     time.Time      `json:"createdAt" db:"created_at"`
    UpdatedAt     time.Time      `json:"updatedAt" db:"updated_at"`
}

// RetryableError represents an error that can be retried
type RetryableError struct {
    Code       string
    Message    string
    Retryable  bool
    StatusCode int
}

// Common retryable error types
var (
    ErrTimeout     = &RetryableError{Code: "TIMEOUT", Retryable: true}
    ErrRateLimit   = &RetryableError{Code: "RATE_LIMIT", Retryable: true}
    ErrNetworkError = &RetryableError{Code: "NETWORK_ERROR", Retryable: true}
    ErrServiceDown = &RetryableError{Code: "SERVICE_DOWN", Retryable: true}

    // Non-retryable
    ErrNotFound    = &RetryableError{Code: "NOT_FOUND", Retryable: false}
    ErrInvalidInput = &RetryableError{Code: "INVALID_INPUT", Retryable: false}
)
```

### Database Schema

```sql
-- Migration: XXX_create_retry_queue_table.sql
CREATE TABLE IF NOT EXISTS retry_queue (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    task_type TEXT NOT NULL,
    payload TEXT NOT NULL,              -- JSON payload
    attempt_count INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 4,
    last_error TEXT,
    next_attempt_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_task_id UNIQUE (task_id)
);

CREATE INDEX idx_retry_queue_next_attempt ON retry_queue(next_attempt_at);
CREATE INDEX idx_retry_queue_task_type ON retry_queue(task_type);
```

### Exponential Backoff Implementation

```go
// BackoffCalculator calculates retry delays with exponential backoff
type BackoffCalculator struct {
    BaseDelay  time.Duration // 1 second
    MaxDelay   time.Duration // 8 seconds
    Multiplier float64       // 2.0
    JitterMax  float64       // 0.1 (10% jitter)
}

func NewBackoffCalculator() *BackoffCalculator {
    return &BackoffCalculator{
        BaseDelay:  1 * time.Second,
        MaxDelay:   8 * time.Second,
        Multiplier: 2.0,
        JitterMax:  0.1,
    }
}

func (b *BackoffCalculator) Calculate(attempt int) time.Duration {
    if attempt <= 0 {
        return b.BaseDelay
    }

    // Calculate exponential delay: base * multiplier^attempt
    delay := float64(b.BaseDelay) * math.Pow(b.Multiplier, float64(attempt))

    // Cap at max delay
    if delay > float64(b.MaxDelay) {
        delay = float64(b.MaxDelay)
    }

    // Add jitter: ±10%
    jitter := delay * b.JitterMax * (2*rand.Float64() - 1)
    delay += jitter

    return time.Duration(delay)
}

// Example delays:
// Attempt 0: 1s (base)
// Attempt 1: 2s (1s * 2^1)
// Attempt 2: 4s (1s * 2^2)
// Attempt 3: 8s (1s * 2^3) - max
```

### Retry Scheduler

```go
// RetryScheduler manages the retry background process
type RetryScheduler struct {
    repo        repository.RetryRepositoryInterface
    backoff     *BackoffCalculator
    executor    TaskExecutor
    logger      *slog.Logger
    stopCh      chan struct{}
    wg          sync.WaitGroup
}

type TaskExecutor interface {
    Execute(ctx context.Context, item *RetryItem) error
}

func (s *RetryScheduler) Start(ctx context.Context) {
    s.wg.Add(1)
    go func() {
        defer s.wg.Done()
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-s.stopCh:
                return
            case <-ticker.C:
                s.processPendingRetries(ctx)
            }
        }
    }()
}

func (s *RetryScheduler) processPendingRetries(ctx context.Context) {
    // Get items ready for retry (next_attempt_at <= now)
    items, err := s.repo.GetPendingRetries(ctx, time.Now())
    if err != nil {
        s.logger.Error("Failed to get pending retries", "error", err)
        return
    }

    for _, item := range items {
        go s.executeRetry(ctx, item)
    }
}

func (s *RetryScheduler) executeRetry(ctx context.Context, item *RetryItem) {
    item.AttemptCount++
    item.UpdatedAt = time.Now()

    err := s.executor.Execute(ctx, item)
    if err != nil {
        s.handleRetryFailure(ctx, item, err)
    } else {
        s.handleRetrySuccess(ctx, item)
    }
}

func (s *RetryScheduler) handleRetryFailure(ctx context.Context, item *RetryItem, err error) {
    item.LastError = err.Error()

    if item.AttemptCount >= item.MaxAttempts {
        // Max attempts reached - mark for manual intervention
        s.logger.Warn("Retry exhausted, marking for manual intervention",
            "task_id", item.TaskID,
            "attempts", item.AttemptCount,
        )
        s.repo.Delete(ctx, item.ID)
        s.emitMaxRetriesExhausted(item)
        return
    }

    // Schedule next retry
    delay := s.backoff.Calculate(item.AttemptCount)
    item.NextAttemptAt = time.Now().Add(delay)
    s.repo.Update(ctx, item)

    s.logger.Info("Retry scheduled",
        "task_id", item.TaskID,
        "attempt", item.AttemptCount,
        "next_attempt", item.NextAttemptAt,
    )
}

func (s *RetryScheduler) handleRetrySuccess(ctx context.Context, item *RetryItem) {
    s.logger.Info("Retry succeeded",
        "task_id", item.TaskID,
        "attempts", item.AttemptCount,
    )
    s.repo.Delete(ctx, item.ID)
    s.emitRetrySuccess(item)
}
```

### Retry Service

```go
// RetryService manages retry operations
type RetryService struct {
    repo      repository.RetryRepositoryInterface
    scheduler *RetryScheduler
    backoff   *BackoffCalculator
    logger    *slog.Logger
}

type RetryServiceInterface interface {
    QueueRetry(ctx context.Context, taskID, taskType string, payload interface{}, err error) error
    CancelRetry(ctx context.Context, itemID string) error
    TriggerImmediate(ctx context.Context, itemID string) error
    GetPendingRetries(ctx context.Context) ([]*RetryItem, error)
    GetRetryStats(ctx context.Context) (*RetryStats, error)
}

type RetryStats struct {
    TotalPending   int `json:"totalPending"`
    TotalSucceeded int `json:"totalSucceeded"`
    TotalFailed    int `json:"totalFailed"`
}

func (s *RetryService) QueueRetry(ctx context.Context, taskID, taskType string, payload interface{}, origErr error) error {
    // Check if error is retryable
    if !isRetryableError(origErr) {
        return fmt.Errorf("error is not retryable: %w", origErr)
    }

    // Check if already queued
    existing, _ := s.repo.FindByTaskID(ctx, taskID)
    if existing != nil {
        return nil // Already queued
    }

    payloadJSON, err := json.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to marshal payload: %w", err)
    }

    item := &RetryItem{
        ID:            uuid.NewString(),
        TaskID:        taskID,
        TaskType:      taskType,
        Payload:       payloadJSON,
        AttemptCount:  0,
        MaxAttempts:   4,
        LastError:     origErr.Error(),
        NextAttemptAt: time.Now().Add(s.backoff.Calculate(0)),
        CreatedAt:     time.Now(),
        UpdatedAt:     time.Now(),
    }

    if err := s.repo.Add(ctx, item); err != nil {
        return fmt.Errorf("failed to queue retry: %w", err)
    }

    s.logger.Info("Retry queued",
        "task_id", taskID,
        "task_type", taskType,
        "next_attempt", item.NextAttemptAt,
    )

    return nil
}

func (s *RetryService) TriggerImmediate(ctx context.Context, itemID string) error {
    item, err := s.repo.FindByID(ctx, itemID)
    if err != nil {
        return fmt.Errorf("retry item not found: %w", err)
    }

    item.NextAttemptAt = time.Now()
    if err := s.repo.Update(ctx, item); err != nil {
        return fmt.Errorf("failed to update retry: %w", err)
    }

    return nil
}

func isRetryableError(err error) bool {
    var retryErr *RetryableError
    if errors.As(err, &retryErr) {
        return retryErr.Retryable
    }

    // Check common error patterns
    errStr := err.Error()
    retryablePatterns := []string{
        "timeout", "rate limit", "network", "connection",
        "temporary", "unavailable", "503", "502", "504",
    }
    for _, pattern := range retryablePatterns {
        if strings.Contains(strings.ToLower(errStr), pattern) {
            return true
        }
    }

    return false
}
```

### API Endpoints

**Get Pending Retries:**
```
GET /api/v1/retry/pending
```

Response:
```json
{
  "success": true,
  "data": {
    "items": [
      {
        "id": "retry-123",
        "taskId": "parse-456",
        "taskType": "parse",
        "attemptCount": 2,
        "maxAttempts": 4,
        "lastError": "TMDb API timeout",
        "nextAttemptAt": "2026-01-18T12:00:04Z",
        "timeUntilRetry": "3s"
      }
    ],
    "stats": {
      "totalPending": 3,
      "totalSucceeded": 15,
      "totalFailed": 2
    }
  }
}
```

**Trigger Immediate Retry:**
```
POST /api/v1/retry/{id}/trigger
```

**Cancel Retry:**
```
DELETE /api/v1/retry/{id}
```

### Frontend Components

**RetryQueuePanel.tsx:**
```tsx
interface RetryQueuePanelProps {
  onRefresh: () => void;
}

const RetryQueuePanel: React.FC<RetryQueuePanelProps> = ({ onRefresh }) => {
  const { data, isLoading } = useQuery({
    queryKey: ['retry', 'pending'],
    queryFn: retryService.getPending,
    refetchInterval: 1000, // Update countdown every second
  });

  const triggerMutation = useMutation({
    mutationFn: retryService.triggerImmediate,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['retry'] });
      toast.success('已觸發立即重試');
    },
  });

  const cancelMutation = useMutation({
    mutationFn: retryService.cancel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['retry'] });
      toast.info('已取消重試');
    },
  });

  if (isLoading) return <Skeleton />;
  if (!data?.items.length) return null;

  return (
    <Card className="p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="font-medium">待重試項目</h3>
        <Badge variant="secondary">{data.items.length}</Badge>
      </div>

      <div className="space-y-3">
        {data.items.map((item) => (
          <div key={item.id} className="flex items-center justify-between p-3 bg-muted rounded-lg">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <RefreshCw className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm font-medium">
                  {item.taskType === 'parse' ? '解析任務' : '元數據獲取'}
                </span>
                <span className="text-xs text-muted-foreground">
                  嘗試 {item.attemptCount}/{item.maxAttempts}
                </span>
              </div>
              <p className="text-xs text-muted-foreground">
                上次錯誤：{item.lastError}
              </p>
              <p className="text-xs text-blue-500">
                <Clock className="h-3 w-3 inline mr-1" />
                <CountdownTimer targetTime={item.nextAttemptAt} /> 後重試
              </p>
            </div>

            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => triggerMutation.mutate(item.id)}
                disabled={triggerMutation.isPending}
              >
                立即重試
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => cancelMutation.mutate(item.id)}
              >
                取消
              </Button>
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
};
```

**CountdownTimer Component:**
```tsx
interface CountdownTimerProps {
  targetTime: string;
}

const CountdownTimer: React.FC<CountdownTimerProps> = ({ targetTime }) => {
  const [remaining, setRemaining] = useState<string>('');

  useEffect(() => {
    const updateRemaining = () => {
      const target = new Date(targetTime).getTime();
      const now = Date.now();
      const diff = Math.max(0, target - now);

      if (diff <= 0) {
        setRemaining('即將重試');
        return;
      }

      const seconds = Math.ceil(diff / 1000);
      setRemaining(`${seconds}s`);
    };

    updateRemaining();
    const interval = setInterval(updateRemaining, 1000);
    return () => clearInterval(interval);
  }, [targetTime]);

  return <span>{remaining}</span>;
};
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/retry/
├── queue.go
├── queue_test.go
├── scheduler.go
├── scheduler_test.go
├── backoff.go
└── backoff_test.go

/apps/api/internal/repository/
└── retry_repository.go

/apps/api/internal/services/
└── retry_service.go

/apps/api/internal/handlers/
└── retry_handler.go

/apps/api/migrations/
└── XXX_create_retry_queue_table.sql
```

**Frontend Files to Create:**
```
/apps/web/src/components/retry/
├── RetryQueuePanel.tsx
├── RetryQueuePanel.spec.tsx
├── CountdownTimer.tsx
└── index.ts
```

### Testing Strategy

**Backend Tests:**
1. Backoff calculation tests (exponential, jitter)
2. Queue operations tests (add, update, delete)
3. Scheduler timing tests
4. Retry service integration tests
5. Error classification tests (retryable vs non-retryable)

**Frontend Tests:**
1. RetryQueuePanel rendering tests
2. Countdown timer tests
3. Trigger/Cancel action tests

**E2E Tests:**
1. Full retry flow (queue → wait → retry → success)
2. Max retries exhausted flow
3. Manual trigger flow

**Coverage Targets:**
- Backend retry package: ≥80%
- Backend service: ≥80%
- Frontend components: ≥70%

### Error Codes

Following project-context.md Rule 7:
- `RETRY_QUEUE_FULL` - Retry queue is at capacity
- `RETRY_NOT_FOUND` - Retry item not found
- `RETRY_NOT_RETRYABLE` - Error is not retryable
- `RETRY_EXHAUSTED` - Maximum retry attempts reached

### Dependencies

**Story Dependencies:**
- Story 3.3 (Fallback Chain) - Provides error context
- Story 3.10 (Status Indicators) - Shows retry status in UI

**Go Libraries:**
- Standard library only

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.11]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR25]
- [Source: _bmad-output/planning-artifacts/architecture.md#NFR-R5]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.3 (Fallback Chain):**
- Orchestrator already detects errors from each source
- Can classify errors as retryable or permanent

**From Story 3.10 (Status Indicators):**
- Status update events can include retry info
- Progress card can show retry countdown

## Dev Agent Record

### Agent Model Used

Claude Opus 4.5 (claude-opus-4-5-20251101)

### Debug Log References

N/A - Implementation was largely complete, session focused on integration.

### Completion Notes List

1. **Implementation already existed** - Found comprehensive implementation in the codebase:
   - `/apps/api/internal/retry/` package with queue.go, scheduler.go, backoff.go, metadata_integration.go
   - `/apps/api/internal/repository/retry_repository.go`
   - `/apps/api/internal/services/retry_service.go`
   - `/apps/api/internal/handlers/retry_handler.go`
   - `/apps/api/internal/database/migrations/011_create_retry_queue_table.go`
   - `/apps/web/src/components/retry/` with RetryQueuePanel.tsx, CountdownTimer.tsx, RetryNotifications.tsx

2. **Integration work completed**:
   - Added `Retry` field to `Repositories` struct in registry.go
   - Updated `NewRepositories()` and `NewRepositoriesWithCache()` to include retry repository
   - Added retry service initialization in main.go
   - Added retry handler registration in main.go
   - Added scheduler start/stop lifecycle in main.go

3. **All tests passing**:
   - Backend retry package: 100% pass
   - Backend repository tests: 100% pass
   - Backend service tests: 100% pass
   - Backend handler tests: 100% pass
   - Frontend component tests: All retry-specific tests pass

### File List

**Backend Files (already existed):**
- `apps/api/internal/retry/queue.go` - RetryItem, RetryableError types
- `apps/api/internal/retry/queue_test.go` - Queue tests
- `apps/api/internal/retry/scheduler.go` - RetryScheduler background process
- `apps/api/internal/retry/scheduler_test.go` - Scheduler tests
- `apps/api/internal/retry/backoff.go` - BackoffCalculator with 1s→2s→4s→8s pattern
- `apps/api/internal/retry/backoff_test.go` - Backoff tests
- `apps/api/internal/retry/metadata_integration.go` - Error classification utilities
- `apps/api/internal/retry/metadata_integration_test.go` - Integration tests
- `apps/api/internal/repository/retry_repository.go` - SQLite CRUD operations
- `apps/api/internal/repository/retry_repository_test.go` - Repository tests
- `apps/api/internal/services/retry_service.go` - Business logic layer
- `apps/api/internal/services/retry_service_test.go` - Service tests
- `apps/api/internal/handlers/retry_handler.go` - HTTP API endpoints
- `apps/api/internal/handlers/retry_handler_test.go` - Handler tests
- `apps/api/internal/database/migrations/011_create_retry_queue_table.go` - Migration

**Backend Files (modified):**
- `apps/api/internal/repository/registry.go` - Added Retry repository
- `apps/api/cmd/api/main.go` - Added retry service, handler, and scheduler lifecycle

**Frontend Files (already existed):**
- `apps/web/src/components/retry/RetryQueuePanel.tsx` - Main panel component
- `apps/web/src/components/retry/RetryQueuePanel.spec.tsx` - Panel tests
- `apps/web/src/components/retry/CountdownTimer.tsx` - Real-time countdown
- `apps/web/src/components/retry/CountdownTimer.spec.tsx` - Timer tests
- `apps/web/src/components/retry/RetryNotifications.tsx` - Toast notifications
- `apps/web/src/components/retry/RetryNotifications.spec.tsx` - Notification tests
- `apps/web/src/components/retry/RetryQueueWithNotifications.tsx` - Combined component
- `apps/web/src/services/retry.ts` - API service
- `apps/web/src/hooks/useRetry.ts` - React Query hooks
