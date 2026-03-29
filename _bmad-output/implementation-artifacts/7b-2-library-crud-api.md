# Story 7b-2: Library CRUD API

Status: done

## Story

As a **frontend developer**,
I want to **have RESTful API endpoints for managing media libraries and paths**,
So that **the Setup Wizard and Settings UI can create, read, update, and delete libraries**.

## Acceptance Criteria

1. **Given** a valid POST request to `/api/v1/libraries`
   **When** the request contains name, content_type, and paths[]
   **Then** a new library is created with all paths validated and added
   **And** the response returns the created library with status 201

2. **Given** a GET request to `/api/v1/libraries`
   **When** libraries exist in the database
   **Then** all libraries are returned with their paths, statuses, and media_count

3. **Given** a PUT request to `/api/v1/libraries/:id`
   **When** the request contains valid update fields
   **Then** the library is updated and the response returns the updated library

4. **Given** a DELETE request to `/api/v1/libraries/:id`
   **When** `?remove_media=true` is specified
   **Then** the library, its paths, and associated media records are deleted

5. **Given** a POST request to `/api/v1/libraries/:id/paths`
   **When** the path already exists in another library
   **Then** the response returns 409 with error code `LIBRARY_DUPLICATE_PATH`

6. **Given** a POST request to `/api/v1/libraries/:id/paths/refresh`
   **When** paths are checked against the filesystem
   **Then** each path's status is updated (accessible, not_found, not_readable, not_directory)

## Tasks / Subtasks

### Task 1: Create Library Service (AC: #1–#6)
- [x]1.1 Create `apps/api/internal/services/media_library_service.go`
- [x]1.2 Define `MediaLibraryServiceInterface` with: CreateLibrary, GetAllLibraries, GetLibrary, UpdateLibrary, DeleteLibrary, AddPath, RemovePath, RefreshPathStatuses
- [x]1.3 Implement path validation (os.Stat check for directory existence and readability)
- [x]1.4 Implement duplicate path check across all libraries
- [x]1.5 Implement media count aggregation per library

### Task 2: Create Library Handler (AC: #1–#6)
- [x]2.1 Create `apps/api/internal/handlers/library_handler.go`
- [x]2.2 `POST /api/v1/libraries` — CreateLibrary
- [x]2.3 `GET /api/v1/libraries` — GetAllLibraries (with paths, counts)
- [x]2.4 `PUT /api/v1/libraries/:id` — UpdateLibrary
- [x]2.5 `DELETE /api/v1/libraries/:id` — DeleteLibrary (?remove_media query param)
- [x]2.6 `POST /api/v1/libraries/:id/paths` — AddPath
- [x]2.7 `DELETE /api/v1/libraries/:id/paths/:pathId` — RemovePath
- [x]2.8 `POST /api/v1/libraries/:id/paths/refresh` — RefreshPathStatuses

### Task 3: Wire into main.go (AC: all)
- [x]3.1 Register `MediaLibraryRepository` in dependency injection
- [x]3.2 Register `MediaLibraryService` with repository dependency
- [x]3.3 Register `LibraryHandler` with service dependency
- [x]3.4 Add all 7 routes to router under `/api/v1/libraries`

### Task 4: Write Tests (AC: #1–#6)
- [x]4.1 Service unit tests (mock repository)
- [x]4.2 Handler unit tests (mock service)
- [x]4.3 Test validation error responses (empty name, invalid type, duplicate path)
- [x]4.4 Test path refresh with mock filesystem
- [x]4.5 Test error codes: LIBRARY_NOT_FOUND, LIBRARY_DUPLICATE_PATH, LIBRARY_PATH_NOT_ACCESSIBLE

## Dev Notes

- Follow Rule 4: Handler → Service → Repository
- Follow Rule 3: standard `{ success, data }` / `{ success, error }` response format
- Follow Rule 6: snake_case JSON fields (Rule 18 transformation at API boundary)
- Follow Rule 7: use LIBRARY_* error codes defined in project-context.md
- Follow Rule 15: wire into main.go before marking complete
- Path validation must handle network mount timeouts (use context with 5s deadline)
