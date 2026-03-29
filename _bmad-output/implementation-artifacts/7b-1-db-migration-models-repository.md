# Story 7b-1: DB Migration, Models & Repository

Status: review

## Story

As a **developer**,
I want to **create the database schema, Go models, and repository layer for media libraries**,
So that **the application has the data foundation to support multi-library management**.

## Acceptance Criteria

1. **Given** the application starts with an existing database
   **When** migration #020 runs
   **Then** `media_libraries` and `media_library_paths` tables are created
   **And** `movies` and `series` tables gain `library_id`, `detected_type`, `override_type` columns

2. **Given** an existing `media_folder_path` setting or `VIDO_MEDIA_DIRS` env var
   **When** migration #020 runs
   **Then** existing paths are migrated into `media_libraries` records with default type "movie"
   **And** existing media records are backfilled with `library_id` based on file path matching

3. **Given** no existing media configuration
   **When** migration #020 runs
   **Then** tables are created with no data (clean install)

4. **Given** the `MediaLibraryRepository` interface
   **When** any CRUD operation is called
   **Then** it correctly persists/retrieves data with proper error handling

## Tasks / Subtasks

### Task 1: Create Migration #020 (AC: #1, #2, #3)
- [ ] 1.1 Create `apps/api/internal/database/migrations/020_create_media_libraries.go`
- [ ] 1.2 CREATE TABLE `media_libraries` (id TEXT PK, name TEXT, content_type TEXT, auto_detect BOOLEAN DEFAULT false, sort_order INTEGER DEFAULT 0, created_at, updated_at)
- [ ] 1.3 CREATE TABLE `media_library_paths` (id TEXT PK, library_id TEXT FK CASCADE, path TEXT UNIQUE, status TEXT DEFAULT 'unknown', last_checked_at, created_at)
- [ ] 1.4 ALTER TABLE `movies` ADD COLUMN library_id, detected_type, override_type
- [ ] 1.5 ALTER TABLE `series` ADD COLUMN library_id, detected_type, override_type
- [ ] 1.6 Data migration: read `settings.media_folder_path` + `VIDO_MEDIA_DIRS`, create library records, backfill `library_id`

### Task 2: Create Go Models (AC: #4)
- [ ] 2.1 Create `apps/api/internal/models/media_library.go`
- [ ] 2.2 Define `MediaLibrary` struct: ID, Name, ContentType, AutoDetect, SortOrder, CreatedAt, UpdatedAt
- [ ] 2.3 Define `MediaLibraryPath` struct: ID, LibraryID, Path, Status, LastCheckedAt, CreatedAt
- [ ] 2.4 Define `MediaLibraryWithPaths` struct (joined view with paths and media_count)

### Task 3: Create Repository Interface (AC: #4)
- [ ] 3.1 Create `apps/api/internal/repository/media_library_repository.go`
- [ ] 3.2 Define `MediaLibraryRepositoryInterface` with: Create, GetByID, GetAll, Update, Delete, AddPath, RemovePath, GetPathsByLibraryID, UpdatePathStatus, GetAllPaths
- [ ] 3.3 Implement SQLite version `SQLiteMediaLibraryRepository`
- [ ] 3.4 Implement `GetAllWithPathsAndCounts()` for Settings UI (joined query)

### Task 4: Write Tests (AC: #1, #2, #3, #4)
- [ ] 4.1 Test migration #020 creates tables with correct schema
- [ ] 4.2 Test data migration from existing settings
- [ ] 4.3 Test clean install (no existing data)
- [ ] 4.4 Test all repository CRUD operations
- [ ] 4.5 Test CASCADE delete (library delete removes paths)
- [ ] 4.6 Test UNIQUE constraint on path

## Dev Notes

- Migration #020 follows existing pattern in `apps/api/internal/database/migrations/`
- UUID generation: use `github.com/google/uuid` (already in go.mod)
- Follow Rule 4: repository layer only, no service logic here
- Follow Rule 15: ensure migration + model fields + repository SQL all in sync
- `auto_detect`, `detected_type`, `override_type` are Phase 2 reserve — create columns but don't use
