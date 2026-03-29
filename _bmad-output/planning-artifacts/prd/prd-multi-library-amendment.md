# PRD Amendment: Multi-Library Media Management (Route 2)

> **Amendment to:** P1-001 資料夾掃描
> **Date:** 2026-03-29
> **Author:** John (PM)
> **Status:** DRAFT — Pending stakeholder approval

---

## 1. Background & Motivation

### Current State

P1-001 was marked DONE (Epic 5), but the implementation has critical limitations:

1. **Setup Wizard** saves a single `media_folder_path` to the settings DB — but the **Scanner never reads it**
2. **Scanner** reads `VIDO_MEDIA_DIRS` environment variable (comma-separated paths) — no media type information
3. **No media type per folder** — all paths are treated identically, no distinction between movies and TV shows
4. **Configuration disconnect** — changing settings in the UI has no effect on scanning behavior

### User Need

Users organize media by type on their NAS:

```
/Volumes/data/media/
├── movies/    (32 files — movies only)
└── tv/        (2,066 files — TV series only)
```

They need to tell Vido: "this folder contains movies, that folder contains TV shows" — just like Plex, Jellyfin, and Emby.

### Competitive Analysis (Summary)

| Feature | Plex | Jellyfin | Emby | Kodi | Vido (current) |
|---------|------|----------|------|------|----------------|
| Multiple libraries | ✅ | ✅ | ✅ | N/A | ❌ |
| Multiple folders per library | ✅ | ✅ | ✅ | N/A | ❌ |
| Type per library | ✅ | ✅ | ✅ | Per-source | ❌ |
| Type auto-detection | ❌ | ❌ (Mixed is broken) | ❌ | ❌ | ❌ |

**Sources:**
- [Plex: Creating Libraries](https://support.plex.tv/articles/200288926-creating-libraries/)
- [Jellyfin: Libraries](https://jellyfin.org/docs/general/server/libraries/)
- [Jellyfin Mixed Library issues](https://github.com/jellyfin/jellyfin/issues/15720)

### Decision: Route 2 — Progressive Enhancement

Phase 1 delivers multi-folder + manual type assignment (matching Plex/Jellyfin baseline).
Schema reserves fields for future auto-detection (Phase 2, if needed).

**Justification:**
- NAS accuracy test: 2,097 real files, 0 misclassifications by regex parser
- But 99%+ of users already organize folders by type (Sonarr/Radarr enforce this)
- Auto-detection solves a problem that rarely exists in practice
- Reserve capability at near-zero cost (~2 DB fields + ~20 lines of code)

---

## 2. Functional Requirements (Amendment)

### P1-001-A: Media Library CRUD (NEW)

| ID | 功能 | Priority | Description |
|----|------|----------|-------------|
| P1-001-A1 | 建立媒體庫 | P0 | Create a named media library with a content type (movie \| series). Each library has a display name and one content type. |
| P1-001-A2 | 新增資料夾路徑 | P0 | Add one or more filesystem paths to a library. Validate path exists and is a directory. Each path belongs to exactly one library. |
| P1-001-A3 | 編輯媒體庫 | P1 | Update library name, content type, or add/remove paths. |
| P1-001-A4 | 刪除媒體庫 | P1 | Delete a library and its path associations. Optionally remove scanned media records from that library. |
| P1-001-A5 | 路徑狀態監控 | P1 | Each path tracks accessibility status (accessible, not_found, not_readable, not_directory). Status refreshable on demand. |

### P1-001-B: Setup Wizard Update (MODIFY)

| ID | 功能 | Priority | Description |
|----|------|----------|-------------|
| P1-001-B1 | 多庫設定步驟 | P0 | Replace the single `media_folder_path` input with a multi-library setup flow: add folders, assign type per folder. Minimum one library required. |
| P1-001-B2 | 類型選擇 | P0 | Content type options: 電影 (movie), 影集 (series). UI dropdown with icons. |

### P1-001-C: Scanner Integration (MODIFY)

| ID | 功能 | Priority | Description |
|----|------|----------|-------------|
| P1-001-C1 | 按庫掃描 | P0 | Scanner reads media libraries from DB (not env var). Scans each library's paths. Assigns library's content_type to scanned media items. |
| P1-001-C2 | 環境變數降級 | P1 | `VIDO_MEDIA_DIRS` becomes fallback: if DB has no libraries, create a default "Media" library from env var with type "mixed". Log deprecation warning. |

### P1-001-D: Settings UI (NEW)

| ID | 功能 | Priority | Description |
|----|------|----------|-------------|
| P1-001-D1 | 媒體庫管理頁面 | P0 | Settings page listing all media libraries as cards. Each card shows: name, type icon, folder count, path statuses. |
| P1-001-D2 | 新增/編輯庫 UI | P0 | Modal or inline form to create/edit a library: name, type, paths. Path input with validation feedback. |

### P1-001-E: Future-Proofing (Route 2 Schema Reserve)

| ID | 功能 | Priority | Description |
|----|------|----------|-------------|
| P1-001-E1 | auto_detect 欄位 | P2 | `media_libraries.auto_detect` BOOLEAN DEFAULT false. Not exposed in Phase 1 UI. |
| P1-001-E2 | detected/override type | P2 | `media_items` gains `detected_type` and `override_type` fields. Phase 1: both NULL, library type used. Phase 2: parser fills detected_type, user can override. |

---

## 3. Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-ML1 | Library CRUD API response time | <200ms |
| NFR-ML2 | Path validation response time | <500ms (filesystem stat) |
| NFR-ML3 | Migration from single path | Automatic, zero-downtime |
| NFR-ML4 | Backward compatibility | `VIDO_MEDIA_DIRS` continues to work as fallback |

---

## 4. Data Model

### New Tables

```sql
-- Media libraries (top-level grouping)
CREATE TABLE media_libraries (
    id TEXT PRIMARY KEY,          -- UUID
    name TEXT NOT NULL,           -- Display name (e.g., "我的電影")
    content_type TEXT NOT NULL,   -- "movie" | "series"
    auto_detect BOOLEAN NOT NULL DEFAULT false,  -- Route 2 reserve
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Filesystem paths belonging to a library
CREATE TABLE media_library_paths (
    id TEXT PRIMARY KEY,          -- UUID
    library_id TEXT NOT NULL REFERENCES media_libraries(id) ON DELETE CASCADE,
    path TEXT NOT NULL UNIQUE,    -- Filesystem path
    status TEXT NOT NULL DEFAULT 'unknown',  -- accessible|not_found|not_readable|not_directory
    last_checked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Modified Tables

```sql
-- Add to existing movies/series tables
ALTER TABLE movies ADD COLUMN library_id TEXT REFERENCES media_libraries(id);
ALTER TABLE series ADD COLUMN library_id TEXT REFERENCES media_libraries(id);

-- Route 2 reserve fields (Phase 2)
ALTER TABLE movies ADD COLUMN detected_type TEXT;
ALTER TABLE movies ADD COLUMN override_type TEXT;
ALTER TABLE series ADD COLUMN detected_type TEXT;
ALTER TABLE series ADD COLUMN override_type TEXT;
```

### Deprecated

- `settings.media_folder_path` — superseded by `media_library_paths`
- `VIDO_MEDIA_DIRS` env var — demoted to fallback; deprecation warning logged

---

## 5. Migration Strategy

### From Current State

1. Read existing `media_folder_path` from settings DB
2. Read `VIDO_MEDIA_DIRS` from environment
3. Merge unique paths
4. For each path: create a `media_library` with name derived from folder name, type "movie" (default)
5. User can adjust types after migration via Settings UI

### Zero-Downtime

- Migration runs automatically on app startup
- Existing media records get `library_id` backfilled based on their file path matching
- Scanner immediately switches to DB-based library reading

---

## 6. Out of Scope (Phase 1)

- Auto-detection of content type (reserved for Phase 2)
- Dynamic views / tag system (reserved for Phase 3)
- Library-level access control (not needed — single user)
- Mixed content type libraries (intentionally excluded — Jellyfin proved this is unreliable)
- Nested library paths (one path = one library, no overlapping)

---

## 7. Success Criteria

1. User can create ≥2 media libraries with different content types via Setup Wizard
2. User can manage libraries (add/edit/delete) from Settings page
3. Scanner uses DB libraries instead of env var
4. Existing NAS deployment migrates automatically without data loss
5. Library grid view shows content type indicators
