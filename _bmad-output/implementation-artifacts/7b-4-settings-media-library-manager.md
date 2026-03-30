# Story 7b-4: Settings Media Library Manager

Status: done

## Story

As a **user managing their media libraries**,
I want to **view, add, edit, and delete media libraries from the Settings page**,
So that **I can manage my library configuration after initial setup without re-running the wizard**.

## Acceptance Criteria

1. **Given** the user navigates to Settings > Scanner
   **When** media libraries are configured
   **Then** each library is displayed as a card showing: type icon, name, paths with status indicators, and media count

2. **Given** the user clicks "+ 新增媒體庫"
   **When** the create modal opens
   **Then** they can enter name, select type, and add paths
   **And** clicking "儲存" creates the library via API

3. **Given** the user clicks "⋮ 更多" → "編輯" on a library card
   **When** the edit modal opens
   **Then** they can modify name, type, and add/remove paths
   **And** clicking "儲存變更" updates the library via API

4. **Given** the user clicks "⋮ 更多" → "刪除"
   **When** the delete confirmation modal opens
   **Then** they see an option "同時移除已掃描的媒體資料"
   **And** clicking "確認刪除" deletes the library via API

5. **Given** a library path status is "not_found"
   **When** the card renders
   **Then** the path shows a red indicator with "無法存取" text

## Tasks / Subtasks

### Task 1: Create Frontend Service (AC: #2, #3, #4)
- [x]1.1 Create `apps/web/src/services/mediaLibraryService.ts`
- [x]1.2 Implement: getLibraries, createLibrary, updateLibrary, deleteLibrary, addPath, removePath, refreshPaths
- [x]1.3 Create TanStack Query hooks in `apps/web/src/hooks/useMediaLibrary.ts`

### Task 2: Create LibraryCard Component (AC: #1, #5)
- [x]2.1 Create `apps/web/src/components/settings/LibraryCard.tsx`
- [x]2.2 Card header: type icon (Film/Tv from lucide) + name + overflow menu (⋮)
- [x]2.3 Card body: path list with status indicators (green/red/gray dot + text)
- [x]2.4 Card footer: folder count + media item count
- [x]2.5 Overflow menu: 編輯, 新增路徑, 刪除

### Task 3: Create LibraryEditModal Component (AC: #2, #3)
- [x]3.1 Create `apps/web/src/components/settings/LibraryEditModal.tsx`
- [x]3.2 Shared modal for create + edit (mode prop)
- [x]3.3 Fields: name input, type dropdown, paths list with add/remove
- [x]3.4 Validation: required name, at least one path, no duplicate paths
- [x]3.5 Submit calls createLibrary or updateLibrary based on mode

### Task 4: Create Delete Confirmation (AC: #4)
- [x]4.1 Delete confirmation dialog with checkbox "同時移除已掃描的媒體資料"
- [x]4.2 Calls deleteLibrary with `?remove_media=true/false`

### Task 5: Create MediaLibraryManager Component (AC: #1)
- [x]5.1 Create `apps/web/src/components/settings/MediaLibraryManager.tsx`
- [x]5.2 Lists all LibraryCards + "+ 新增媒體庫" button
- [x]5.3 Integrate into `ScannerSettings.tsx` — replace env var display section

### Task 6: Write Tests (AC: #1–#5)
- [x]6.1 LibraryCard renders with correct status indicators
- [x]6.2 LibraryEditModal validates required fields
- [x]6.3 MediaLibraryManager fetches and displays libraries
- [x]6.4 Delete confirmation sends correct query param

## Dev Notes

- UX spec: `multi-library-ux-spec.md` Sections 2–3
- Design reference: `ux-design.pen` Settings screens (H1, H4, K-series)
- Design tokens: `border-slate-700`, `bg-slate-800`, `p-6`, `rounded-lg` (match existing)
- Status colors: green-400/red-400/slate-400 (with text, not color-only — accessibility)
- This is frontend-only story — backend API from 7b-2 must be complete first
- Follow Rule 5: use TanStack Query for all API data
