# Story 5.7: Batch Operations

Status: ready-for-dev

## Story

As a **power user**,
I want to **perform batch operations on multiple media items**,
So that **I can efficiently manage large numbers of files**.

## Acceptance Criteria

1. **AC1: Selection Mode**
   - Given the library is displayed
   - When the user enters "selection mode" (via toolbar button or long-press on mobile)
   - Then checkboxes appear on each item (top-left corner overlay)
   - And a selection toolbar appears at top

2. **AC2: Selection Methods**
   - Given selection mode is active
   - When selecting items
   - Then single click toggles selection on one item
   - And Shift+Click selects a range
   - And Ctrl/Cmd+Click selects multiple individual items
   - And "全選" (Select All) button selects all visible items

3. **AC3: Batch Actions**
   - Given multiple items are selected
   - When batch actions are available
   - Then options include:
     - 刪除選取項目 (Delete selected)
     - 重新解析 (Re-parse selected)
     - 匯出元數據 (Export metadata)

4. **AC4: Delete Confirmation**
   - Given the user selects "Delete selected"
   - When confirming the action
   - Then a confirmation dialog shows item count: "確定要刪除 5 個項目嗎？"
   - And upon confirmation, items are removed from library
   - And deletion is permanent (cannot be undone)

5. **AC5: Batch Progress**
   - Given a batch operation is in progress
   - When processing multiple items
   - Then a progress indicator shows: "處理中 5 / 20..."
   - And errors are collected and shown at the end
   - And the user can cancel remaining items

## Tasks / Subtasks

- [ ] Task 1: Create Batch Delete API (AC: 3, 4)
  - [ ] 1.1: Add `DELETE /api/v1/library/batch` endpoint
  - [ ] 1.2: Accept body: `{ "ids": ["id1", "id2"], "type": "movie"|"series" }`
  - [ ] 1.3: Implement batch delete in LibraryService
  - [ ] 1.4: Return success count and error details for failed items
  - [ ] 1.5: Write handler and service tests

- [ ] Task 2: Create Batch Re-parse API (AC: 3)
  - [ ] 2.1: Add `POST /api/v1/library/batch/reparse` endpoint
  - [ ] 2.2: Accept body: `{ "ids": ["id1", "id2"], "type": "movie"|"series" }`
  - [ ] 2.3: Reset parse_status to "pending" for selected items
  - [ ] 2.4: Trigger re-parse for items with file_path
  - [ ] 2.5: Write tests

- [ ] Task 3: Create Batch Export API (AC: 3)
  - [ ] 3.1: Add `POST /api/v1/library/batch/export` endpoint
  - [ ] 3.2: Accept body: `{ "ids": ["id1", "id2"], "format": "json" }`
  - [ ] 3.3: Return downloadable JSON with metadata for selected items
  - [ ] 3.4: Write tests

- [ ] Task 4: Create Selection Mode State (AC: 1, 2)
  - [ ] 4.1: Add selection state to library page (or Zustand store if complex)
  - [ ] 4.2: Track: `isSelectionMode`, `selectedIds: Set<string>`, `selectedType`
  - [ ] 4.3: Implement single click, Shift+Click range, Ctrl+Click multi-select
  - [ ] 4.4: "Select All" selects all currently visible (filtered) items

- [ ] Task 5: Create Selection Toolbar Component (AC: 1, 2, 3)
  - [ ] 5.1: Create `/apps/web/src/components/library/SelectionToolbar.tsx`
  - [ ] 5.2: Show when selection mode active: selected count + action buttons
  - [ ] 5.3: Actions: 🗑️ Delete, 🔄 Re-parse, 📤 Export, ✕ Cancel selection
  - [ ] 5.4: Sticky at top of library content area
  - [ ] 5.5: Write component tests

- [ ] Task 6: Create Batch Confirmation Dialog (AC: 4)
  - [ ] 6.1: Create `/apps/web/src/components/library/BatchConfirmDialog.tsx`
  - [ ] 6.2: For delete: warning message with item count, "此操作無法復原"
  - [ ] 6.3: Confirm button (danger red for delete), Cancel button
  - [ ] 6.4: Write component tests

- [ ] Task 7: Create Batch Progress Component (AC: 5)
  - [ ] 7.1: Create `/apps/web/src/components/library/BatchProgress.tsx`
  - [ ] 7.2: Progress bar with "處理中 N / Total..."
  - [ ] 7.3: Error collection shown after completion
  - [ ] 7.4: Cancel button to stop remaining items
  - [ ] 7.5: Write component tests

- [ ] Task 8: Add Checkbox Overlay to PosterCard (AC: 1)
  - [ ] 8.1: Add optional `selectable` and `selected` props to PosterCard
  - [ ] 8.2: When selectable: show checkbox overlay (top-left, blue border when selected)
  - [ ] 8.3: Selected state: blue border 2px + checkbox ✓ overlay
  - [ ] 8.4: Write updated PosterCard tests

- [ ] Task 9: Batch Hooks & Service (AC: 3, 4, 5)
  - [ ] 9.1: Add `useBatchDelete()`, `useBatchReparse()`, `useBatchExport()` mutation hooks
  - [ ] 9.2: Add batch methods to libraryService.ts
  - [ ] 9.3: Invalidate library queries after batch operations
  - [ ] 9.4: Handle progress tracking for large batches

## Dev Notes

### Architecture Requirements

**FR40:** Batch operations (delete, re-parse)
Confirmation required for destructive operations
Progress tracking for large batches

### Existing Code to Reuse (DO NOT Reinvent)

- `PosterCard` — extend with selectable props, not new component
- `LibraryGrid` / `LibraryTable` — wrap items to add selection behavior
- TanStack Query `useMutation` — for batch operations
- Dialog component patterns from metadata-editor (`MetadataEditorDialog.tsx`)
- `PaginatedResponse` pattern for batch results

### Batch Delete API Pattern

```go
// DELETE /api/v1/library/batch
type BatchDeleteRequest struct {
    IDs  []string `json:"ids" binding:"required,min=1"`
    Type string   `json:"type" binding:"required,oneof=movie series"`
}

type BatchResult struct {
    SuccessCount int            `json:"success_count"`
    FailedCount  int            `json:"failed_count"`
    Errors       []BatchError   `json:"errors,omitempty"`
}

type BatchError struct {
    ID      string `json:"id"`
    Message string `json:"message"`
}
```

### Selection State Pattern

```typescript
// Can use useState in library page or Zustand if needed across components
interface SelectionState {
  isSelectionMode: boolean;
  selectedIds: Set<string>;
  toggleSelection: (id: string) => void;
  selectRange: (startId: string, endId: string, allIds: string[]) => void;
  selectAll: (ids: string[]) => void;
  clearSelection: () => void;
  enterSelectionMode: () => void;
  exitSelectionMode: () => void;
}
```

### Keyboard Shortcuts (Desktop)

- `Shift+Click` — range select
- `Ctrl/Cmd+Click` — multi-select
- `Ctrl/Cmd+A` — select all (when in selection mode)
- `Escape` — exit selection mode

### Project Structure Notes

```
Backend (extend):
/apps/api/internal/services/library_service.go   ← ADD BatchDelete, BatchReparse, BatchExport
/apps/api/internal/handlers/library_handler.go    ← ADD batch endpoints

Frontend (new):
/apps/web/src/components/library/SelectionToolbar.tsx       ← NEW
/apps/web/src/components/library/SelectionToolbar.spec.tsx  ← NEW
/apps/web/src/components/library/BatchConfirmDialog.tsx     ← NEW
/apps/web/src/components/library/BatchConfirmDialog.spec.tsx ← NEW
/apps/web/src/components/library/BatchProgress.tsx          ← NEW
/apps/web/src/components/library/BatchProgress.spec.tsx     ← NEW

Frontend (modify):
/apps/web/src/components/media/PosterCard.tsx    ← ADD selectable/selected props
/apps/web/src/routes/library.tsx                 ← ADD selection state
/apps/web/src/hooks/useLibrary.ts                ← ADD batch mutation hooks
/apps/web/src/services/libraryService.ts         ← ADD batch methods
```

### Dependencies

- Story 5-1 (Media Library Grid View) — library page, grid, API
- Story 5-2 (List View) — table selection should also work

### Error Codes

- `BATCH_PARTIAL_FAILURE` — Some items failed during batch operation
- `BATCH_EMPTY_SELECTION` — No items selected for operation
- `BATCH_DELETE_FAILED` — Batch delete operation failed

### Testing Strategy

- Backend: batch delete removes items, partial failures reported, re-parse resets status
- SelectionToolbar: shows count, renders action buttons, cancel exits mode
- BatchConfirmDialog: shows warning, confirm triggers action
- BatchProgress: shows progress, errors displayed after completion
- PosterCard: checkbox visible when selectable, blue border when selected

### References

- [Source: _bmad-output/planning-artifacts/epics.md#Story-5.7]
- [Source: _bmad-output/planning-artifacts/prd.md#FR40]
- [Source: _bmad-output/planning-artifacts/ux-design-specification.md#Multi-Select-Batch-Operations]

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
