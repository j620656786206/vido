# Story 7b-3: Setup Wizard Multi-Library Step

Status: backlog

## Story

As a **new user running the Setup Wizard**,
I want to **add multiple media folders and assign content types (movie/series) during initial setup**,
So that **Vido immediately knows which folders contain movies and which contain TV shows**.

## Acceptance Criteria

1. **Given** the Setup Wizard reaches the media folder step
   **When** the step renders
   **Then** it shows a multi-library entry form with path input + type dropdown per row
   **And** a "+ 新增媒體庫" button to add more rows

2. **Given** one or more library entries are filled
   **When** the user clicks "下一步"
   **Then** all paths are validated via API (directory exists check)
   **And** validation errors are shown inline per row

3. **Given** the user completes the Setup Wizard
   **When** `CompleteSetup` is called
   **Then** `media_libraries` records are created in DB (one per entry)
   **And** the old `media_folder_path` setting is NOT stored

4. **Given** only one library entry exists
   **When** the user tries to remove it
   **Then** the remove button is hidden (minimum one library required)

5. **Given** duplicate paths are entered
   **When** the user clicks "下一步"
   **Then** a warning is shown: "此路徑已被使用"

## Tasks / Subtasks

### Task 1: Create MediaLibrarySetupStep Component (AC: #1, #4)
- [ ] 1.1 Create `apps/web/src/components/setup/MediaLibrarySetupStep.tsx`
- [ ] 1.2 Implement dynamic row list with path input + type dropdown per row
- [ ] 1.3 Type dropdown: 電影 (movie, Film icon) | 影集 (series, Tv icon)
- [ ] 1.4 "+ 新增媒體庫" button adds new row with defaults
- [ ] 1.5 "✕ 移除" button per row, hidden when only 1 row

### Task 2: Update SetupWizard Integration (AC: #1, #3)
- [ ] 2.1 Replace `MediaFolderStep` import with `MediaLibrarySetupStep` in `SetupWizard.tsx`
- [ ] 2.2 Update step config: change data shape from `{ mediaFolderPath: string }` to `{ libraries: [{ path, contentType }] }`
- [ ] 2.3 Update "下一步" validation logic

### Task 3: Update SetupService (Frontend) (AC: #3)
- [ ] 3.1 Update `apps/web/src/services/setupService.ts` — change `CompleteSetup` payload
- [ ] 3.2 Replace `media_folder_path` field with `libraries` array in request body
- [ ] 3.3 Follow Rule 18: send snake_case `content_type` in request body

### Task 4: Update SetupService (Backend) (AC: #2, #3, #5)
- [ ] 4.1 Modify `apps/api/internal/services/setup_service.go` — `CompleteSetup` method
- [ ] 4.2 Accept `[]SetupLibrary{ Path, ContentType }` instead of single `MediaFolderPath`
- [ ] 4.3 Create `media_libraries` records via `MediaLibraryRepository`
- [ ] 4.4 Validate paths on server side (directory exists + no duplicates)
- [ ] 4.5 Remove `media_folder_path` setting storage

### Task 5: Update Setup Handler (AC: #2)
- [ ] 5.1 Modify `apps/api/internal/handlers/setup_handler.go` — update request struct
- [ ] 5.2 Update `validate-step` endpoint for new media library step data shape

### Task 6: Write Tests (AC: #1–#5)
- [ ] 6.1 Frontend: MediaLibrarySetupStep renders with add/remove/type dropdown
- [ ] 6.2 Frontend: validation displays inline errors
- [ ] 6.3 Frontend: setupService sends correct payload shape
- [ ] 6.4 Backend: setup_service_test.go — updated for library creation flow
- [ ] 6.5 Backend: setup_handler_test.go — updated for new request struct

## Dev Notes

- UX spec: `multi-library-ux-spec.md` Section 1
- Design reference: `ux-design.pen` Setup Wizard screens
- Delete old `MediaFolderStep.tsx` after this story is complete (or keep as reference)
- This is a full-stack story — follow story splitting rule: backend tasks (4, 5) ≤3, frontend tasks (1, 2, 3) ≤3 ✓
