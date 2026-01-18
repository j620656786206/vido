# Story 3.10: Parse Status Indicators

Status: ready-for-dev

## Story

As a **media collector**,
I want to **see clear status indicators for parsing progress**,
So that **I know what's happening with each file**.

## Acceptance Criteria

1. **AC1: Status Icons in File List**
   - Given a file is being parsed
   - When viewing the file list
   - Then status icons indicate:
     - ⏳ Parsing in progress
     - ✅ Successfully parsed
     - ⚠️ Parsed with warnings (manual selection needed)
     - ❌ Parse failed

2. **AC2: Step Progress Indicators (UX-3)**
   - Given parsing is in progress
   - When viewing the progress
   - Then step indicators show: "解析檔名中..." → "搜尋 TMDb..." → "下載海報..."
   - And the current step is highlighted
   - And progress percentage is displayed (33% → 66% → 100%)

3. **AC3: Failure Reason Display (UX-4)**
   - Given parsing fails
   - When viewing the error
   - Then the failure reason is explained
   - And shows which sources were tried (TMDb ❌ → Douban ❌ → Wikipedia ❌)
   - And clear next steps are provided (手動搜尋、編輯檔名、跳過)

4. **AC4: Non-Blocking Progress Card**
   - Given parsing is running
   - When the user continues browsing
   - Then a floating progress card appears in the bottom-right corner
   - And parsing runs in the background
   - And the card auto-dismisses 3 seconds after success

## Tasks / Subtasks

- [ ] Task 1: Create Parse Status Types (AC: 1, 2)
  - [ ] 1.1: Create `/apps/api/internal/models/parse_status.go`
  - [ ] 1.2: Define `ParseStatus` enum (pending, parsing, success, warning, failed)
  - [ ] 1.3: Define `ParseStep` struct (step name, status, timestamp)
  - [ ] 1.4: Define `ParseProgress` struct (steps, current step, percentage)
  - [ ] 1.5: Write model tests

- [ ] Task 2: Create Parse Events for Real-Time Updates (AC: 2, 4)
  - [ ] 2.1: Create `/apps/api/internal/events/parse_events.go`
  - [ ] 2.2: Define event types (ParseStarted, StepCompleted, ParseCompleted, ParseFailed)
  - [ ] 2.3: Implement event emitter interface
  - [ ] 2.4: Wire up with SSE (Server-Sent Events) handler
  - [ ] 2.5: Write event tests

- [ ] Task 3: Create Parse Progress SSE Endpoint (AC: 2, 4)
  - [ ] 3.1: Create `GET /api/v1/parse/progress/{taskId}` SSE endpoint
  - [ ] 3.2: Stream real-time parse events
  - [ ] 3.3: Handle connection cleanup on client disconnect
  - [ ] 3.4: Support multiple concurrent listeners
  - [ ] 3.5: Write handler tests

- [ ] Task 4: Create Floating Parse Progress Card Component (AC: 2, 4)
  - [ ] 4.1: Create `FloatingParseProgressCard.tsx` component
  - [ ] 4.2: Implement SSE connection for real-time updates
  - [ ] 4.3: Display current step with spinner animation
  - [ ] 4.4: Show layered progress (TMDb → Douban → Wikipedia → AI)
  - [ ] 4.5: Add minimize/close buttons
  - [ ] 4.6: Auto-dismiss 3 seconds after success
  - [ ] 4.7: Write component tests

- [ ] Task 5: Create Layered Progress Indicator Component (AC: 2)
  - [ ] 5.1: Create `LayeredProgressIndicator.tsx` component
  - [ ] 5.2: Display 4 layers with status icons (✓, ⏳, ❌, ·)
  - [ ] 5.3: Show progress percentage per layer
  - [ ] 5.4: Animate transitions between layers
  - [ ] 5.5: Write component tests

- [ ] Task 6: Create Parse Status Badge Component (AC: 1)
  - [ ] 6.1: Create `ParseStatusBadge.tsx` component
  - [ ] 6.2: Implement status icons (⏳, ✅, ⚠️, ❌)
  - [ ] 6.3: Add color coding (blue, green, yellow, red)
  - [ ] 6.4: Include tooltip with status details
  - [ ] 6.5: Write component tests

- [ ] Task 7: Create Error Details Panel Component (AC: 3)
  - [ ] 7.1: Create `ErrorDetailsPanel.tsx` component
  - [ ] 7.2: Display failure reasons for each source
  - [ ] 7.3: Show source attempt chain with status
  - [ ] 7.4: Add action buttons (手動搜尋、編輯檔名、跳過)
  - [ ] 7.5: Write component tests

- [ ] Task 8: Integrate Status Indicators with Media List (AC: 1)
  - [ ] 8.1: Add ParseStatusBadge to media card/row components
  - [ ] 8.2: Update media list to show parse status column
  - [ ] 8.3: Add status filter option
  - [ ] 8.4: Write integration tests

- [ ] Task 9: Create SSE Hook for React (AC: 2, 4)
  - [ ] 9.1: Create `useParseProgress` hook
  - [ ] 9.2: Handle SSE connection lifecycle
  - [ ] 9.3: Expose progress state and events
  - [ ] 9.4: Handle reconnection on disconnect
  - [ ] 9.5: Write hook tests

## Dev Notes

### Architecture Requirements

**FR22: View parse status indicators**
- Real-time progress updates via SSE
- Non-blocking UI design

**UX-3: Wait experience optimization**
- Step indicators show progress
- Current step highlighted
- Percentage display

**UX-4: Failure handling friendliness**
- Always show next step
- Explain reasons
- Multi-layer fallback visible

### Parse Status Types

```go
// ParseStatus represents the overall status of a parse operation
type ParseStatus string

const (
    ParseStatusPending   ParseStatus = "pending"
    ParseStatusParsing   ParseStatus = "parsing"
    ParseStatusSuccess   ParseStatus = "success"
    ParseStatusWarning   ParseStatus = "warning"   // Partial success
    ParseStatusFailed    ParseStatus = "failed"
)

// ParseStep represents a single step in the parse process
type ParseStep struct {
    Name      string      `json:"name"`       // e.g., "filename_extract", "tmdb_search"
    Label     string      `json:"label"`      // e.g., "解析檔名", "搜尋 TMDb"
    Status    StepStatus  `json:"status"`     // pending, in_progress, success, failed, skipped
    StartedAt *time.Time  `json:"startedAt,omitempty"`
    EndedAt   *time.Time  `json:"endedAt,omitempty"`
    Error     string      `json:"error,omitempty"`
}

type StepStatus string

const (
    StepPending    StepStatus = "pending"
    StepInProgress StepStatus = "in_progress"
    StepSuccess    StepStatus = "success"
    StepFailed     StepStatus = "failed"
    StepSkipped    StepStatus = "skipped"
)

// ParseProgress holds the full progress state
type ParseProgress struct {
    TaskID       string       `json:"taskId"`
    Filename     string       `json:"filename"`
    Status       ParseStatus  `json:"status"`
    Steps        []ParseStep  `json:"steps"`
    CurrentStep  int          `json:"currentStep"`
    Percentage   int          `json:"percentage"`
    Message      string       `json:"message,omitempty"`
    Result       *ParseResult `json:"result,omitempty"`
    StartedAt    time.Time    `json:"startedAt"`
    CompletedAt  *time.Time   `json:"completedAt,omitempty"`
}

// Standard parse steps
var StandardParseSteps = []ParseStep{
    {Name: "filename_extract", Label: "解析檔名"},
    {Name: "tmdb_search", Label: "搜尋 TMDb"},
    {Name: "douban_search", Label: "搜尋豆瓣"},
    {Name: "wikipedia_search", Label: "搜尋 Wikipedia"},
    {Name: "ai_retry", Label: "AI 重試"},
    {Name: "download_poster", Label: "下載海報"},
}
```

### Parse Events

```go
// ParseEvent represents a real-time parse event
type ParseEvent struct {
    Type      ParseEventType `json:"type"`
    TaskID    string         `json:"taskId"`
    Timestamp time.Time      `json:"timestamp"`
    Data      interface{}    `json:"data"`
}

type ParseEventType string

const (
    EventParseStarted    ParseEventType = "parse_started"
    EventStepStarted     ParseEventType = "step_started"
    EventStepCompleted   ParseEventType = "step_completed"
    EventStepFailed      ParseEventType = "step_failed"
    EventParseCompleted  ParseEventType = "parse_completed"
    EventParseFailed     ParseEventType = "parse_failed"
    EventProgressUpdate  ParseEventType = "progress_update"
)

// EventEmitter interface for broadcasting events
type EventEmitter interface {
    Emit(event ParseEvent)
    Subscribe(taskID string) <-chan ParseEvent
    Unsubscribe(taskID string, ch <-chan ParseEvent)
}
```

### SSE Endpoint

```go
// GET /api/v1/parse/progress/{taskId}
// Content-Type: text/event-stream

func (h *ParseHandler) StreamProgress(c *gin.Context) {
    taskID := c.Param("taskId")

    c.Header("Content-Type", "text/event-stream")
    c.Header("Cache-Control", "no-cache")
    c.Header("Connection", "keep-alive")

    eventChan := h.emitter.Subscribe(taskID)
    defer h.emitter.Unsubscribe(taskID, eventChan)

    c.Stream(func(w io.Writer) bool {
        select {
        case event := <-eventChan:
            data, _ := json.Marshal(event)
            fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
            return true
        case <-c.Request.Context().Done():
            return false
        }
    })
}
```

### Frontend Components

**FloatingParseProgressCard.tsx:**
```tsx
interface FloatingParseProgressCardProps {
  taskId: string;
  onClose: () => void;
  onComplete: (result: ParseResult) => void;
}

const FloatingParseProgressCard: React.FC<FloatingParseProgressCardProps> = ({
  taskId,
  onClose,
  onComplete,
}) => {
  const { progress, status } = useParseProgress(taskId);
  const [isMinimized, setIsMinimized] = useState(false);

  // Auto-dismiss after success
  useEffect(() => {
    if (status === 'success') {
      const timer = setTimeout(() => {
        onComplete(progress.result);
        onClose();
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [status]);

  return (
    <div className={cn(
      "fixed bottom-6 right-6 w-[420px] bg-card rounded-xl shadow-xl",
      "border border-border animate-in slide-in-from-right-5",
      isMinimized && "h-12"
    )}>
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b">
        <div className="flex items-center gap-2">
          {status === 'parsing' && <Loader2 className="h-4 w-4 animate-spin" />}
          {status === 'success' && <CheckCircle className="h-4 w-4 text-green-500" />}
          {status === 'failed' && <XCircle className="h-4 w-4 text-red-500" />}
          <span className="font-medium">
            {status === 'parsing' && '正在解析...'}
            {status === 'success' && '✅ 解析完成！'}
            {status === 'failed' && '❌ 解析失敗'}
          </span>
        </div>
        <div className="flex items-center gap-1">
          <Button variant="ghost" size="icon" onClick={() => setIsMinimized(!isMinimized)}>
            {isMinimized ? <ChevronUp /> : <ChevronDown />}
          </Button>
          <Button variant="ghost" size="icon" onClick={onClose}>
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Progress Content */}
      {!isMinimized && (
        <div className="p-4 space-y-4">
          {/* Overall Progress Bar */}
          <div className="space-y-1">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">進度</span>
              <span>{progress.percentage}%</span>
            </div>
            <Progress value={progress.percentage} />
          </div>

          {/* Layered Steps */}
          <LayeredProgressIndicator steps={progress.steps} currentStep={progress.currentStep} />

          {/* Filename */}
          <div className="text-sm text-muted-foreground truncate">
            檔案：{progress.filename}
          </div>

          {/* Actions on failure */}
          {status === 'failed' && (
            <ErrorDetailsPanel
              steps={progress.steps}
              filename={progress.filename}
            />
          )}
        </div>
      )}
    </div>
  );
};
```

**LayeredProgressIndicator.tsx:**
```tsx
interface LayeredProgressIndicatorProps {
  steps: ParseStep[];
  currentStep: number;
}

const LayeredProgressIndicator: React.FC<LayeredProgressIndicatorProps> = ({
  steps,
  currentStep,
}) => {
  const getStepIcon = (step: ParseStep, index: number) => {
    switch (step.status) {
      case 'success':
        return <CheckCircle className="h-4 w-4 text-green-500" />;
      case 'in_progress':
        return <Loader2 className="h-4 w-4 text-blue-500 animate-spin" />;
      case 'failed':
        return <XCircle className="h-4 w-4 text-red-500" />;
      case 'skipped':
        return <MinusCircle className="h-4 w-4 text-muted-foreground" />;
      default:
        return <Circle className="h-4 w-4 text-muted-foreground" />;
    }
  };

  return (
    <div className="space-y-2">
      {steps.map((step, index) => (
        <div
          key={step.name}
          className={cn(
            "flex items-center gap-3 py-1",
            index === currentStep && "font-medium"
          )}
        >
          {getStepIcon(step, index)}
          <span className={cn(
            step.status === 'pending' && "text-muted-foreground"
          )}>
            {step.label}
          </span>
          {step.status === 'in_progress' && (
            <span className="text-sm text-muted-foreground animate-pulse">
              搜尋中...
            </span>
          )}
          {step.status === 'failed' && step.error && (
            <span className="text-sm text-red-500">{step.error}</span>
          )}
        </div>
      ))}
    </div>
  );
};
```

**ParseStatusBadge.tsx:**
```tsx
interface ParseStatusBadgeProps {
  status: ParseStatus;
  tooltip?: string;
}

const ParseStatusBadge: React.FC<ParseStatusBadgeProps> = ({ status, tooltip }) => {
  const config = {
    pending: { icon: Clock, color: 'text-muted-foreground', label: '等待中' },
    parsing: { icon: Loader2, color: 'text-blue-500', label: '解析中', animate: true },
    success: { icon: CheckCircle, color: 'text-green-500', label: '已完成' },
    warning: { icon: AlertTriangle, color: 'text-yellow-500', label: '需要處理' },
    failed: { icon: XCircle, color: 'text-red-500', label: '失敗' },
  };

  const { icon: Icon, color, label, animate } = config[status];

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger>
          <Icon className={cn("h-4 w-4", color, animate && "animate-spin")} />
        </TooltipTrigger>
        <TooltipContent>
          <p>{tooltip || label}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
};
```

**ErrorDetailsPanel.tsx:**
```tsx
interface ErrorDetailsPanelProps {
  steps: ParseStep[];
  filename: string;
  onManualSearch: () => void;
  onEditFilename: () => void;
  onSkip: () => void;
}

const ErrorDetailsPanel: React.FC<ErrorDetailsPanelProps> = ({
  steps,
  filename,
  onManualSearch,
  onEditFilename,
  onSkip,
}) => {
  const failedSteps = steps.filter(s => s.status === 'failed');

  return (
    <div className="space-y-4">
      {/* Failed Steps Summary */}
      <div className="bg-red-500/10 rounded-lg p-3 space-y-2">
        <h4 className="font-medium text-red-500">失敗原因</h4>
        <ul className="space-y-1 text-sm">
          {failedSteps.map(step => (
            <li key={step.name} className="flex items-center gap-2">
              <XCircle className="h-3 w-3 text-red-500" />
              <span>{step.label}：{step.error || '無回應'}</span>
            </li>
          ))}
        </ul>
      </div>

      {/* Source Chain Visualization */}
      <div className="flex items-center gap-2 text-sm">
        {steps.slice(1, 5).map((step, i) => (
          <React.Fragment key={step.name}>
            <span className={cn(
              step.status === 'success' && 'text-green-500',
              step.status === 'failed' && 'text-red-500',
              step.status === 'skipped' && 'text-muted-foreground'
            )}>
              {step.label.replace('搜尋 ', '')}
              {step.status === 'success' && ' ✓'}
              {step.status === 'failed' && ' ✗'}
            </span>
            {i < 3 && <ArrowRight className="h-3 w-3 text-muted-foreground" />}
          </React.Fragment>
        ))}
      </div>

      {/* Action Buttons */}
      <div className="flex flex-col gap-2">
        <Button onClick={onManualSearch} className="w-full">
          <Search className="h-4 w-4 mr-2" />
          手動搜尋
        </Button>
        <Button variant="outline" onClick={onEditFilename} className="w-full">
          <Edit className="h-4 w-4 mr-2" />
          編輯檔名後重試
        </Button>
        <Button variant="ghost" onClick={onSkip} className="w-full text-muted-foreground">
          跳過此檔案
        </Button>
      </div>
    </div>
  );
};
```

### useParseProgress Hook

```typescript
// hooks/useParseProgress.ts
export const useParseProgress = (taskId: string) => {
  const [progress, setProgress] = useState<ParseProgress | null>(null);
  const [status, setStatus] = useState<ParseStatus>('pending');
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    const eventSource = new EventSource(`/api/v1/parse/progress/${taskId}`);

    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      setProgress(data);
      setStatus(data.status);
    };

    eventSource.onerror = (err) => {
      setError(new Error('SSE connection failed'));
      eventSource.close();

      // Attempt reconnection after 3 seconds
      setTimeout(() => {
        // Reconnect logic
      }, 3000);
    };

    return () => {
      eventSource.close();
    };
  }, [taskId]);

  return { progress, status, error };
};
```

### Project Structure Notes

**Backend Files to Create:**
```
/apps/api/internal/models/
└── parse_status.go

/apps/api/internal/events/
├── parse_events.go
└── emitter.go

/apps/api/internal/handlers/
└── parse_progress_handler.go
```

**Frontend Files to Create:**
```
/apps/web/src/components/parse/
├── FloatingParseProgressCard.tsx
├── FloatingParseProgressCard.spec.tsx
├── LayeredProgressIndicator.tsx
├── LayeredProgressIndicator.spec.tsx
├── ParseStatusBadge.tsx
├── ParseStatusBadge.spec.tsx
├── ErrorDetailsPanel.tsx
├── ErrorDetailsPanel.spec.tsx
└── index.ts

/apps/web/src/hooks/
├── useParseProgress.ts
└── useParseProgress.spec.ts
```

### Animation Specifications

| Animation | Duration | Easing | Trigger |
|-----------|----------|--------|---------|
| Card slide-in | 300ms | ease-out | On mount |
| Progress bar | 500ms | ease-out | On update |
| Success checkmark | 500ms | bounce | On success |
| Spinner | 1s | linear | During parsing |
| Auto-dismiss | 200ms | ease-out | After 3s success |

### Accessibility Requirements

**ARIA Attributes:**
- Progress card: `role="status"`, `aria-live="polite"`
- Progress bar: `role="progressbar"`, `aria-valuenow`, `aria-valuemin`, `aria-valuemax`
- Step list: `aria-label="解析進度: 66%, 目前搜尋豆瓣電影"`
- Close button: `aria-label="關閉進度卡片"`

**Keyboard Navigation:**
- ESC: Close progress card
- Tab: Navigate between buttons
- Enter/Space: Activate buttons

### Testing Strategy

**Backend Tests:**
1. Parse status model tests
2. Event emitter tests
3. SSE endpoint tests (connection, streaming, cleanup)
4. Integration tests with parse service

**Frontend Tests:**
1. FloatingParseProgressCard rendering tests
2. LayeredProgressIndicator state tests
3. ParseStatusBadge variant tests
4. ErrorDetailsPanel action tests
5. useParseProgress hook tests (mock SSE)

**E2E Tests:**
1. Full parse progress flow
2. SSE connection handling
3. Auto-dismiss behavior
4. Error recovery flow

**Coverage Targets:**
- Backend models: ≥80%
- Backend handlers: ≥70%
- Frontend components: ≥70%
- Hooks: ≥80%

### Error Codes

Following project-context.md Rule 7:
- `PARSE_TASK_NOT_FOUND` - Parse task ID not found
- `PARSE_SSE_CONNECTION_FAILED` - SSE connection failed
- `PARSE_EVENT_EMIT_FAILED` - Failed to emit parse event

### Dependencies

**Go Libraries:**
- Standard library SSE (net/http)

**Frontend Libraries:**
- `lucide-react` - Icons (already installed)
- Native EventSource API - SSE support

**Story Dependencies:**
- Story 3.2 (AI Fansub Parsing) - Triggers parse events
- Story 3.3 (Fallback Chain) - Provides multi-source status

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-3.10]
- [Source: _bmad-output/planning-artifacts/architecture.md#FR22]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#UX-3-UX-4]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Component-Strategy]
- [Source: project-context.md#Rule-4-Layered-Architecture]

### Previous Story Intelligence

**From Story 3.3 (Fallback Chain):**
- Orchestrator provides status updates for each source
- Can emit events during fallback progression

**From Story 3.2 (AI Parsing):**
- AI parsing step can be tracked
- Returns structured parse result

**From UX Design Specification:**
- Floating Parse Progress Card specifications
- Layered Progress Indicator design
- Error Card component design
- Animation and timing specifications

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
