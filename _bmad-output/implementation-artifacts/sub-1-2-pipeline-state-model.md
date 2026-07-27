# Story sub-1.2: Pipeline state model — provenance table + status enum

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1` — Automatic Traditional-Chinese subtitles for English media (M1) · **Risk: 🟡 MEDIUM** · **BACKEND-ONLY**
**Source:** `_bmad-output/planning-artifacts/epics-subtitle-pipeline.md` § Story 1.2 · architecture **D2** + § Schema finding + § M1 Pilot Instrumentation + **P9**
**Depends on:** **nothing.** Implementation step **2 of 7**, but it shares **zero files** with sub-1-1 (`internal/ai/**` + `go.mod`) — see § Parallelism below. Can be built and merged before, after, or alongside sub-1-1.
**Blocks:** sub-1-4 (routing verdicts that map to `skipped`/`no_text_source` — the DB **writes** land with the orchestrator, 1.5b; corrected 2026-07-27 at 1.4 drafting), sub-1-5a/1.5b (prompt-version definer / provenance + cache key from `RunVersion`), sub-1-6 (enumerates by status).
**Split note (2026-07-27, IR-r2 M1):** sub-1-5 → **1.5a** + **1.5b**. `RunVersion`'s cache-key + provenance consumer is **sub-1-5b**; the P11 prompt-version definer is **sub-1-5a**. References to "sub-1-5" elsewhere in this file read as the pair.
**Cross-stack split check:** backend tasks = 4, frontend tasks = **0** → single story. Frontend consumption of the new enum values is deliberately deferred and tracked — see AC #7.

---

## Story

As the pipeline,
I want a provenance table and extended `subtitle_status` values,
so that runs are tracked, resumable, and a no-text-source item is representable.

---

## 🔎 Two findings from the codebase that change the shape of this story

**Read these before writing the migration — they are not in the epic text.**

### Finding 1 — `subtitle_status` has **no CHECK constraint**. There is nothing to ALTER.

The epic AC reads *"`subtitle_status` accepts `probing`/`extracting`/…"*, which sounds like a DDL change. It isn't:

```go
// 018_add_subtitle_fields.go:18,25 — movies + series
ALTER TABLE movies ADD COLUMN subtitle_status TEXT DEFAULT 'not_searched';
// 025_add_episode_subtitle_fields.go:26 — episodes
ALTER TABLE episodes ADD COLUMN subtitle_status TEXT DEFAULT 'not_searched';
```

Plain `TEXT`, no `CHECK(...)`, on all three tables. SQLite already "accepts" every value. **The enum is enforced in Go and consumed by the frontend — that is where the contract lives, and that is what this story extends.** Migration 030 therefore creates the provenance table **only**; it must not attempt to rebuild `movies`/`series`/`episodes` to add a constraint (SQLite has no `ALTER TABLE … ADD CONSTRAINT`; emulating it means a full table rebuild — enormous risk for zero requirement).

### Finding 2 — item-grain provenance ≠ cue-grain cache. They are two mechanisms, and this story owns one.

The epic AC says *"a completed item/cue has a provenance record"*. That slash hides a design decision. The architecture separates them:

| Grain | Mechanism | Key | Owner |
|---|---|---|---|
| **Item** (one produced subtitle) | **`subtitle_runs` table** — this story | `(media_id, media_type)` + the version tuple | **sub-1-2** |
| **Cue / segment** | the existing **tiered cache** (AD #4) | `hash(source cue) + metadata + glossary + prompt + model` (D4) | **sub-1-5** |

**Do not build a per-cue provenance table.** D2 explicitly rejects two storage mechanisms for one concept (Rule 24's superseded-mechanism corollary; the `series.seasons` / `seasons` precedent). This story ships the item grain and the **shared version tuple type** that sub-1-5 reuses to build the cue-grain cache key. One concept, one home.

---

## Acceptance Criteria

### AC #1 — Migration **030** creates the `subtitle_runs` provenance table

**Given** the migration registry's latest version is **029** (`029_drop_new_shell_enabled_setting.go` — verified 2026-07-27), **when** migration **030** runs, **then** a `subtitle_runs` table and its two indexes exist.

**Table name is `subtitle_runs` (plural), not the architecture prose's `subtitle_run`.** Rule 6 mandates `snake_case plural` for tables and `project-context.md` is the bible; the architecture's singular is prose, not a stamped contract. Live counter-examples (`show_glossary`, `connection_history`) are mass nouns; `run` is a count noun, so Rule 6 applies cleanly. Flagged for Alexyu below — non-blocking, build `subtitle_runs`.

```sql
CREATE TABLE IF NOT EXISTS subtitle_runs (
    id               TEXT PRIMARY KEY,
    media_id         TEXT NOT NULL,
    media_type       TEXT NOT NULL CHECK(media_type IN ('movie','series','episode')),
    tmdb_id          INTEGER,                          -- the tmdb id USED for context (FR26/FR27 attribution); NULL when unmatched
    metadata_hash    TEXT NOT NULL DEFAULT '',         -- ┐
    glossary_version TEXT NOT NULL DEFAULT '',         -- │ the VERSION TUPLE (AC #4)
    prompt_version   TEXT NOT NULL DEFAULT '',         -- │
    model_id         TEXT NOT NULL DEFAULT '',         -- ┘
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK(status IN ('pending','running','completed','failed','skipped')),
    source_language  TEXT,                             -- routed source language (FR9) — written by sub-1-4
    output_path      TEXT,                             -- the placed .zh-Hant.srt — written by sub-1-5 AFTER a successful place (P9)
    cue_count        INTEGER,
    cache_enabled    INTEGER NOT NULL DEFAULT 0,       -- RESERVED (D4) — written by sub-1-5, stays 0 here
    error_message    TEXT,
    started_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at     TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_subtitle_runs_media  ON subtitle_runs(media_id, media_type);  -- resume lookup (AC #5) hot path
CREATE INDEX IF NOT EXISTS idx_subtitle_runs_status ON subtitle_runs(status);                -- batch enumeration (sub-1-6)
```

Rulings baked in:

- **`media_type` uses the *internal table* vocabulary `movie|series|episode`**, deliberately **NOT** `requests.media_type`'s TMDB vocabulary (`movie|tv`, migration 027). Provenance is keyed to *our* rows, not to a TMDB request. `subtitle_status` lives on all three tables (migrations 018 + 025), so all three grains must be representable. Getting this backwards is the exact confusion 13-1a called out in the opposite direction.
- **`media_id` is `TEXT`**, mirroring `show_glossary.media_id` (migration 028) — movie int ids are stringified. Same convention, no new one invented.
- **`cache_enabled` is a reserved column** written by sub-1-5 (D4: *"if the genuine prefix cannot clear 4096 tokens, explicitly disable prompt caching for that show **and record it**"*). Reserving it now follows the 027 precedent (`seasons`/`episodes` reserved NULL for 13-2a) and avoids a migration 031 for one boolean.
- **`Down()` drops the table** (`DROP TABLE IF EXISTS subtitle_runs`), mirroring 027. No column-drop dance is needed — nothing is added to an existing table.

**Structure:** `apps/api/internal/database/migrations/030_create_subtitle_runs_table.go`, self-registering via its own `init()` → `Register(&createSubtitleRunsTable{migrationBase: NewMigrationBase(30, "create_subtitle_runs_table")})`. **There is no central list to edit** — `registry.go` collects from `init()`. A duplicate version number is a hard `Register` error at startup, so confirm nothing else claims 30.

### AC #2 — `[@contract-v1]` `SubtitleStatus` extended to 9 values (frontend-consumed wire contract)

**Given** `subtitle_status` is serialized to the frontend (`json:"subtitle_status"` on `models.Movie`, `models.Series`, `models.Episode`) and used as a URL search param (`apps/web/src/routes/library.tsx:22` — the Rule 26 precedent), **when** the enum is extended, **then** it carries a Rule 20 `[@contract-v1]` stamp.

`apps/api/internal/models/movie.go:51-59` — 4 existing values are **unchanged**; 5 are **added**:

```go
// SubtitleStatus represents the subtitle lifecycle state of a media file.
// [@contract-v1] — frontend-consumed wire contract (json:"subtitle_status" on
// Movie/Series/Episode; a URL search param on /library). Consumers: sub-1-4,
// sub-1-5, sub-1-6, and the deferred frontend rendering work (AC #7).
type SubtitleStatus string

const (
    // --- search-flavoured, pre-existing (migrations 018/025). UNCHANGED. ---
    SubtitleStatusNotSearched SubtitleStatus = "not_searched"
    SubtitleStatusSearching   SubtitleStatus = "searching"
    SubtitleStatusFound       SubtitleStatus = "found"
    SubtitleStatusNotFound    SubtitleStatus = "not_found"

    // --- pipeline-flavoured, added by sub-1-2. ---
    SubtitleStatusProbing      SubtitleStatus = "probing"        // ffprobe: enumerating tracks (FR1)
    SubtitleStatusExtracting   SubtitleStatus = "extracting"     // ffmpeg -c copy in flight (FR2/3)
    SubtitleStatusTranslating  SubtitleStatus = "translating"    // LLM translation in flight (FR10)
    SubtitleStatusNoTextSource SubtitleStatus = "no_text_source" // TERMINAL — no usable text track (FR5)
    SubtitleStatusSkipped      SubtitleStatus = "skipped"        // TERMINAL — routed out (und / non-English tag, FR9/P0)
)
```

Plus two helpers (new — no validation existed before, which is why the DB drift in Finding 1 went unnoticed):

```go
// AllSubtitleStatuses is the authoritative ordered value set. The single source
// of truth the [@contract-v1] stamp refers to — extend here, nowhere else.
func AllSubtitleStatuses() []SubtitleStatus

// IsValid reports whether s is a known status.
func (s SubtitleStatus) IsValid() bool

// IsTerminal reports whether s is an end state no further pipeline stage will
// advance (found / not_found / no_text_source / skipped).
func (s SubtitleStatus) IsTerminal() bool
```

`IsValid` is **additive** — no existing call site validates `SubtitleStatus`, so adding it cannot reject data that used to be written. Do **not** retro-wire it into `UpdateSubtitleStatus` in this story (that would be an unrequested behaviour change on the shipped search path); sub-1-4 onwards use it on the write paths they own.

**`no_text_source` vs `skipped` — the distinction sub-1-4 depends on, define it here so it isn't re-invented:**

| Value | Meaning | Recoverable by? |
|---|---|---|
| `no_text_source` | The file has **no usable text subtitle track at all** (image-only tracks, or none). FR5. | P2 ASR (FR29) |
| `skipped` | A text track exists but the pipeline **deliberately declined** it — `und`, or a non-English/non-CJK tag. FR9 + **P0** (`und` is *never* treated as English). | a corrected track tag / manual flow |

### AC #3 — `models.SubtitleRun` + run-status enum

**Given** the new table, **then** `apps/api/internal/models/subtitle_run.go` defines the row model, mirroring `models/glossary.go` (migration 028 / Story 9R-6 — the nearest precedent, same subsystem, same recency):

```go
type SubtitleRunStatus string

const (
    SubtitleRunPending   SubtitleRunStatus = "pending"
    SubtitleRunRunning   SubtitleRunStatus = "running"
    SubtitleRunCompleted SubtitleRunStatus = "completed"
    SubtitleRunFailed    SubtitleRunStatus = "failed"
    SubtitleRunSkipped   SubtitleRunStatus = "skipped"
)

// Media grains a run can be attached to — the INTERNAL table vocabulary,
// deliberately not requests.media_type's TMDB 'movie'|'tv' (AC #1).
const (
    SubtitleRunMediaMovie   = "movie"
    SubtitleRunMediaSeries  = "series"
    SubtitleRunMediaEpisode = "episode"
)

type SubtitleRun struct {
    ID              string            `db:"id"               json:"id"`
    MediaID         string            `db:"media_id"         json:"media_id"`
    MediaType       string            `db:"media_type"       json:"media_type"`
    TMDbID          *int64            `db:"tmdb_id"          json:"tmdb_id,omitempty"`
    MetadataHash    string            `db:"metadata_hash"    json:"metadata_hash"`
    GlossaryVersion string            `db:"glossary_version" json:"glossary_version"`
    PromptVersion   string            `db:"prompt_version"   json:"prompt_version"`
    ModelID         string            `db:"model_id"         json:"model_id"`
    Status          SubtitleRunStatus `db:"status"           json:"status"`
    SourceLanguage  string            `db:"source_language"  json:"source_language,omitempty"`
    OutputPath      string            `db:"output_path"      json:"output_path,omitempty"`
    CueCount        int               `db:"cue_count"        json:"cue_count,omitempty"`
    CacheEnabled    bool              `db:"cache_enabled"    json:"cache_enabled"`
    ErrorMessage    string            `db:"error_message"    json:"error_message,omitempty"`
    StartedAt       time.Time         `db:"started_at"       json:"started_at"`
    CompletedAt     *time.Time        `db:"completed_at"     json:"completed_at,omitempty"`
}

func (r *SubtitleRun) Validate() error   // media_id / media_type / status required + in-set (ValidationError, glossary.go pattern)
func (r *SubtitleRun) Version() RunVersion
```

Nullable DB columns map to pointers (`*int64`, `*time.Time`) or are scanned through `sql.Null*` into plain strings/ints — pick one and be consistent with the repository's scan (AC #5 exists to catch the mismatch).

### AC #4 — `[@contract-v1]` `RunVersion` — the shared version tuple

**Given** the M1 pilot re-runs translations repeatedly to compare prompt quality, **when** a run is recorded, **then** it carries the **full** version tuple, so that a prompt or model change makes a prior run non-matching rather than silently reusable.

```go
// RunVersion is the identity of "which inputs produced this translation".
// [@contract-v1] — consumed by sub-1-5 (composes the cue-grain cache key as
// hash(cue) + RunVersion, D4) and sub-1-6. Changing a field name or the
// canonical string form is a Rule 20 bump + downstream stale-mark.
type RunVersion struct {
    MetadataHash    string // snapshot hash of the TMDb metadata injected as context (FR26)
    GlossaryVersion string // M1: always "" — the glossary is P2, but the field is versioned NOW (D4 cross-dep)
    PromptVersion   string // the P11 constant's value at run time
    ModelID         string // e.g. "claude-haiku-4-5"
}

// Equal reports tuple equality — the resume predicate (AC #5).
func (v RunVersion) Equal(other RunVersion) bool
```

**Why every field is mandatory even when empty** (architecture § M1 Pilot Instrumentation, verbatim): *"Omitting prompt/model is a silent-failure trap: changing the prompt and re-running would return the cached prior translation, making two variants look identical and yielding a false 'the prompt made no difference' conclusion. The pilot's comparison data would be invalid with no error surfacing."* `GlossaryVersion` is always `""` in M1 and **must still exist** — D4's cross-component note: *"M1 must version the glossary field even while it is always empty."*

This story **defines and persists** the tuple. It does **not** compute `MetadataHash` (sub-1-5, from the TMDb context it assembles) and does **not** define `PromptVersion`'s constant (sub-1-5, P11 — the constant lives with the prompt text).

### AC #5 — `SubtitleRunRepository` with Rule 15 column sync, and the resume predicate

**Given** a re-run, **when** a completed run exists for the same media **with a matching `RunVersion`**, **then** it can be skipped (NFR-R3 resume basis).

`apps/api/internal/repository/subtitle_run_repository.go`, mirroring `glossary_repository.go`:

```go
type SubtitleRunRepositoryInterface interface {
    Create(ctx context.Context, run *models.SubtitleRun) error
    Update(ctx context.Context, run *models.SubtitleRun) error
    FindByID(ctx context.Context, id string) (*models.SubtitleRun, error)
    // FindCompletedRun is the RESUME PREDICATE: the most recent 'completed' run
    // for this media whose version tuple matches. A prompt/model bump yields no
    // match, so the pilot's re-run is not silently skipped. Returns (nil, nil)
    // when there is none — absence is not an error.
    FindCompletedRun(ctx context.Context, mediaID, mediaType string, v models.RunVersion) (*models.SubtitleRun, error)
    ListByStatus(ctx context.Context, status models.SubtitleRunStatus, limit int) ([]models.SubtitleRun, error)
}
```

Registered on the aggregate in **both** constructors — `repository/registry.go:10` (`Repositories.SubtitleRuns`), `NewRepositories` (:39) **and** `NewRepositoriesWithCache` (:71). Missing the second is the classic half-wiring bug; grep for `Glossary:` and add a sibling line at each hit.

**🚨 Rule 15 DB Column Sync is the headline risk of this story.** The precedent is exact: *bugfix-20-1* — `series.seasons` (migration 006) was never added to `seriesSelectColumns`/`scanSeries`, so `GetSeasons` always returned `[]` and the season accordion was empty for **every** series, **undetected because the unit test mocked the repo**.

Mandatory here:

1. A single `subtitleRunColumns` const string used by **every** SELECT — never hand-written twice.
2. `INSERT`, `UPDATE`, the SELECT column list, and the row `Scan` must all name the **same 16 columns**. Count them.
3. **Integration tests against a real in-memory SQLite DB, not a mocked repo.** A round-trip test that writes a fully-populated `SubtitleRun`, reads it back, and asserts **field-by-field equality on all 16 columns** — including `cache_enabled`, `tmdb_id`, and `completed_at`, the three most likely to be silently dropped.

### AC #6 — Scope fence: what this story deliberately does **not** ship

**Given** the temptation to build ahead, **then** none of the following lands here, and a reviewer seeing them should push back:

- ❌ No `SUBTITLE_` error codes, no Rule 7 list change, **no `code-review/instructions.xml` sync** (prefix count stays 16) — that is **sub-1-3**.
- ❌ No SSE stage values, no `docs/sse-event-types*.md` — **sub-1-3**.
- ❌ No ffprobe/ffmpeg/extractor/router/detector code — **sub-1-4**. No value is ever written to `subtitle_status` by this story — the orchestrator (**1.5b+**) owns the writes; 1.4 only decides.
- ❌ No cache-key construction, no `MetadataHash` computation, no prompt-version constant, no `placer.go` call, no `cache_enabled` write — **sub-1-5**.
- ❌ No orchestrator, no feature flag, no scanner enqueue, no `batch.go:244` — **sub-1-6**.
- ❌ No service layer, no handler, no route, **no Swagger annotations** (Rule 15: this story adds zero HTTP surface).
- ❌ No `UpdateSubtitleStatus` behaviour change on the shipped search path; no retro-wiring of `IsValid()` into existing writers.
- ❌ No frontend, no `.pen`, no visual baselines — see AC #7.

### AC #7 — Frontend consumption is deferred, with evidence and a tracked entry

**Given** the enum is a frontend-consumed contract, **when** this backend-only story lands, **then** the frontend is verified to **degrade safely** and the rendering work is filed as a tracked backlog entry (Rule 24 lane ③), **not** left as a prose mention.

Evidence gathered at story-authoring time (2026-07-27) — this is why deferral is safe, not an assumption:

| Surface | Current handling of an unknown value | Verdict |
|---|---|---|
| `apps/web/src/types/library.ts:41,106,173` | typed `subtitleStatus?: string` — a loose string, **not** a union | ✅ no TS compile break |
| `apps/web/src/utils/libraryStatus.ts:102-123` `deriveSubtitleStatus` | falls through `found` → embedded-track inference → `not_found` → **`return null`** | ✅ fail-soft: **no badge**, never throws (its own docstring: *"Returns null when genuinely unknown (badge absent, never errors — F3)"*) |
| `apps/web/src/routes/library.tsx:42` search-param guard | `typeof search.subtitleStatus === 'string'` | ✅ **Rule 26-safe** — the guard only drops all-digit values; every status value is alphabetic. Do **not** "fix" it. |
| `apps/web/src/components/media/EpisodeList.tsx:44-47` `SubtitleStatusIcon` | `episode.subtitleStatus ?? 'not_searched'` then compares known values | ✅ renders the default icon |
| `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx:87-92` | `subtitleStatus === 'found'` guard only | ✅ unaffected |

**And** — critically — **no backend code writes a new value until sub-1-4**, so nothing can reach the frontend during this story's lifetime.

The gap that remains: an item parked in `extracting`/`translating`/`no_text_source`/`skipped` renders **no badge at all**, which is a UX regression against showing progress or a terminal reason. That is out of scope **here** but **is in M1 scope** — Alexyu ruled on 2026-07-27 that M1 ships with the badges, so the work is carried by **sub-1-7a** (UX spec screen) → **sub-1-7b** (frontend), both `ready-for-dev`. It is **not** covered by sub-1-6, which owns the SSE *progress* surface (F3), not library/detail badge rendering.

**What that means for this story: nothing changes.** sub-1-2 stays backend-only and does not wait on 1-7a/1-7b — it *produces* the `[@contract-v1]` enum they consume. Ship it independently.

---

## Tasks / Subtasks

- [ ] **Task 1 — Migration 030 (AC #1)**
  - [ ] 1.1 Confirm no migration claims version 30: `grep -rn "NewMigrationBase(30" apps/api/internal/database/migrations/`.
  - [ ] 1.2 Write `030_create_subtitle_runs_table.go` with its own `init()` + `Register(...)`; `Up` creates the table + both indexes, `Down` drops the table. Head comment cites Story sub-1-2, D2, and P9 (mirror 027/028's comment density).
  - [ ] 1.3 Write `030_create_subtitle_runs_table_test.go` mirroring `027_create_requests_table_test.go`: `:memory:` DB + `migration.Up(tx)`, then subtests for — minimal insert applies defaults (`status='pending'`, `cache_enabled=0`); `media_type` CHECK rejects `'tv'` (the TMDB-vocabulary trap); `status` CHECK rejects an out-of-set value; both indexes exist (`SELECT name FROM sqlite_master WHERE type='index'`).
  - [ ] 1.4 **Do not** touch `movies`/`series`/`episodes` DDL (Finding 1).

- [ ] **Task 2 — Enum + models (AC #2, #3, #4)**
  - [ ] 2.1 Extend `models/movie.go:54-59` with the 5 new constants + the `[@contract-v1]` doc comment; add `AllSubtitleStatuses()`, `IsValid()`, `IsTerminal()`. Existing 4 constants untouched.
  - [ ] 2.2 New `models/subtitle_run.go`: `SubtitleRunStatus` + media-grain constants + `SubtitleRun` + `Validate()` (`ValidationError`, `glossary.go` pattern) + `Version()`.
  - [ ] 2.3 Add `RunVersion` + `Equal()` with the `[@contract-v1]` doc comment naming sub-1-5 / sub-1-6 as consumers.
  - [ ] 2.4 `models/subtitle_run_test.go` + extend `models/movie_test.go`: `IsValid` accepts all 9 / rejects junk; `AllSubtitleStatuses()` has exactly 9 entries and contains every declared constant (a table-driven guard that catches "added a const, forgot the slice"); `IsTerminal` true for exactly `found`/`not_found`/`no_text_source`/`skipped`; `RunVersion.Equal` differs when **any single** field differs — assert all four independently, since a 3-field comparison would pass a naive test.

- [ ] **Task 3 — Repository + registration (AC #5)**
  - [ ] 3.1 `repository/subtitle_run_repository.go` mirroring `glossary_repository.go`; define `SubtitleRunRepositoryInterface` alongside the impl (the glossary precedent — `interfaces.go` is not the only legal home; Rule 11 only requires it live in the repository package).
  - [ ] 3.2 One `subtitleRunColumns` const reused by every SELECT; INSERT / UPDATE / SELECT / `Scan` all cover the **same 16 columns** (Rule 15).
  - [ ] 3.3 Implement `FindCompletedRun` — `status='completed'` **AND** all four tuple columns equal **AND** `(media_id, media_type)` match, `ORDER BY started_at DESC LIMIT 1`; `sql.ErrNoRows` → `(nil, nil)`.
  - [ ] 3.4 Register `SubtitleRuns` on the `Repositories` struct (`registry.go:10`) and in **both** `NewRepositories` (:39) and `NewRepositoriesWithCache` (:71). Verify with `grep -n "Glossary:" registry.go` → add a sibling at each hit.
  - [ ] 3.5 Confirm **no** `cmd/api/main.go` change is needed — this story adds no service or handler, so nothing beyond `repository.NewRepositories(db)` consumes it (Rule 15 wiring check).

- [ ] **Task 4 — Tests, lint, contract hygiene (AC #5, #7)**
  - [ ] 4.1 `repository/subtitle_run_repository_test.go` — **integration, real `:memory:` SQLite, not a mock** (Rule 15 / bugfix-20-1). Round-trip a fully-populated run and assert **field-by-field on all 16 columns**; explicitly cover `cache_enabled`, `tmdb_id` (both NULL and set), and `completed_at` (both NULL and set).
  - [ ] 4.2 `FindCompletedRun` tests: exact-tuple match returns the row; **one** test per tuple field mutated → returns `(nil, nil)`; a `failed`/`running` run with a matching tuple returns `(nil, nil)`; no rows → `(nil, nil)` with **no** error.
  - [ ] 4.3 File the lane-③ entry `backlog-subtitle-status-fe-rendering` in `sprint-status.yaml` if it is not already present, bidirectionally linked to this story (AC #7). *(Pre-filed at story creation — verify, don't duplicate.)*
  - [ ] 4.4 `go test ./...` green from `apps/api/`; `pnpm lint:all` green from the repo root.

---

## Dev Notes

### Parallelism — this story does **not** wait on sub-1-1

The architecture's 7-step sequence is a *pipeline build order*, not a compile dependency. File-level overlap between sub-1-1 and sub-1-2 is **zero**:

| Story | Files touched |
|---|---|
| sub-1-1 | `internal/ai/claude.go`, `internal/ai/claude_test.go`, `internal/ai/provider.go`, `go.mod`, `go.sum` |
| **sub-1-2** | `internal/database/migrations/030_*`, `internal/models/movie.go`, `internal/models/subtitle_run.go`, `internal/repository/subtitle_run_repository.go`, `internal/repository/registry.go` |

Both can be developed and merged in either order. **sub-1-5 is the first story that needs both** (it calls the SDK client *and* writes provenance).

### Repo patterns to mirror (do not invent new ones)

| Need | Mirror this | Why |
|---|---|---|
| Migration file + `init()` registration | `027_create_requests_table.go` | Most recent table-creating migration; identical shape (CHECK enums + indexes + `DROP TABLE` Down). |
| Migration test | `027_create_requests_table_test.go` | `:memory:` + direct `migration.Up(tx)`; subtests assert defaults **and** that each CHECK rejects. |
| Model file (struct + const enum + `Validate`) | `models/glossary.go` | Same subsystem (subtitles), same recency (9R-6), uses `ValidationError`. |
| `media_id TEXT` convention | `show_glossary.media_id` (migration 028) | Movie int ids stringified — already the house convention for subtitle-subsystem tables. |
| Repository + interface + registration | `glossary_repository.go` + `registry.go:28,59,90` | Interface declared next to the impl; registered in **both** constructors. |

### The migration registry — how it actually works

`registry.go` has **no list to edit**. Each migration file's `init()` calls `Register(...)`; `GetAll()` sorts by version. Consequences:

- Adding the file is the whole registration step.
- `Register` returns an error on a duplicate version — hence Task 1.1's grep.
- `Down()` is real and exercised by `runner_test.go`; make it correct, not a stub.
- SQLite here is `modernc.org/sqlite` (pure Go, driver name `"sqlite"`) in tests — **not** `mattn/go-sqlite3`. Copy the import block from `027_create_requests_table_test.go` verbatim.

### Rule compliance for this story

- **Rule 1** — all code under `apps/api/`. ✅
- **Rule 4 / Rule 11** — Repository layer only; no service, no handler. Interface lives in the repository package.
- **Rule 6** — table `subtitle_runs` (snake_case **plural**); Go file `subtitle_run_repository.go` (snake_case); struct `SubtitleRun` (PascalCase); JSON `snake_case`. All four dimensions are in the AC.
- **Rule 7** — **no new error codes.** `Validate()` returns the existing `ValidationError`. No `code-review/instructions.xml` sync (AC #6).
- **Rule 8** — `TIMESTAMP` columns, `time.Time` in Go, ISO 8601 on the wire. `started_at` defaults to `CURRENT_TIMESTAMP`.
- **Rule 9** — tests co-located (`030_*_test.go`, `subtitle_run_test.go`, `subtitle_run_repository_test.go`).
- **Rule 13** — every `error` propagated with `fmt.Errorf("…: %w", err)`; `sql.ErrNoRows` is the **one** intentional swallow (→ `(nil, nil)`) and carries a comment saying why.
- **Rule 15** — the headline rule (AC #5): DB Column Sync + integration test over a real DB. Also: `main.go` wiring check → nothing to wire (Task 3.5); Swagger → no endpoints added.
- **Rule 16** — `assert.Equal` on each scanned field, `require.NoError` before dereferencing, `assert.Error` on each CHECK-violation insert. No `assert.NotNil`-as-proof.
- **Rule 20** — `[@contract-v1]` on **AC #2** (`SubtitleStatus` value set) and **AC #4** (`RunVersion`). **Upstream consumed: none** — this story acks no version-stamped AC (sub-1-1's `[@contract-v1]` `CachingCompleter` is acked by sub-1-5, not here). Producer-side stale-mark grep is nil at authoring time (no downstream consumer is drafted yet).
- **Rule 24** — one discovery, triaged to lane ③ (AC #7 / Discovery Triage below).
- **Rule 26** — the `library.tsx` search-param guard is verified safe for all-alphabetic enum values; **do not touch it**.

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** Zero files under `apps/web/src/components/**`; backend-only Go. Rule 23 does not apply. No baselines to regenerate.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 1.2] — the three epic ACs.
- [Source: `subtitle-pipeline-architecture.md`#Schema finding that forces a wire-contract change] — 4 search-flavoured values, FR5 unrepresentable, Rule 20 obligation, `subtitle_search_score`/`subtitle_last_searched` stay unset rather than repurposed.
- [Source: `subtitle-pipeline-architecture.md`#D2 — Pipeline-state persistence] — split by concern; migration is 030 (latest is 029); the rejected dual-storage design.
- [Source: `subtitle-pipeline-architecture.md`#M1 Pilot Instrumentation] — the fully-versioned tuple and the silent-failure trap; provenance fields enumerated.
- [Source: `subtitle-pipeline-architecture.md`#D4 — Chunking and cache key] — cue-grain cache key; "M1 must version the glossary field even while it is always empty"; the 4096-token disable-and-record ruling behind `cache_enabled`.
- [Source: `subtitle-pipeline-architecture.md`#P9] — provenance is written AFTER a successful place (why `output_path` is nullable and sub-1-5 owns the write order).
- [Source: `subtitle-pipeline-architecture.md`#Delta tree] — `subtitle_run_repository.go` + `models/subtitle_run.go` + `030_subtitle_pipeline_state.go` placement.
- [Source: `prd.md`] — FR5 (no usable text source), FR9 (routing), NFR-R3 (granular recovery / resume).
- [Source: `project-context.md`#Rule 15] — DB Column Sync + the bugfix-20-1 precedent; #Rule 6 naming; #Rule 20 stamping; #Rule 24 triage; #Rule 26 TanStack coercion.
- [Source: `apps/api/internal/database/migrations/018_add_subtitle_fields.go`:18,25 + `025_add_episode_subtitle_fields.go`:26] — the no-CHECK evidence for Finding 1.
- [Source: `apps/api/internal/repository/registry.go`:10,39,71] — the two-constructor registration trap.
- [Source: `apps/web/src/utils/libraryStatus.ts`:102-123] — the fail-soft `return null` that makes AC #7's deferral safe.

---

## Dev Agent Record

### Agent Model Used

_(fill in at implementation)_

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?** — **YES**, two items, both known at authoring time:
  - **③ backlog-with-carry-forward-link → ⬆️ PROMOTED to in-epic stories the same day.** Originally filed as `backlog-subtitle-status-fe-rendering`; on the Alexyu ruling of 2026-07-27 ("前端 badge 讓 M1 出貨時就帶 badge") it became **`sub-1-7a-subtitle-status-badge-design`** + **`sub-1-7b-subtitle-status-badge-frontend`** (both `ready-for-dev`; `epics-subtitle-pipeline.md` Epic 1 gained Story 1.7a/1.7b). The backlog entry is retained as `superseded` for the audit trail — do not work it, work the two stories. **Discovery record:** the 5 new `subtitle_status` values render as **no badge** in `deriveSubtitleStatus` (fail-soft `return null`) and as the default icon in `EpisodeList`. Safe for M1 backend work — nothing writes them until sub-1-4 — but a UX gap once the pipeline is live. **Not** covered by sub-1-6 (which owns the SSE *progress* surface, not library/detail badge rendering).
  - **① expand-scope-in-place** → **AC #1**. `subtitle_status` has **no DB CHECK constraint** (migrations 018/025 are plain `TEXT DEFAULT`), so the epic AC's "accepts the new values" needs **no DDL** — the enum is a Go + frontend contract only. Absorbed by restating AC #1 as "create the provenance table only, do not attempt to add a CHECK", plus an explicit prohibition on a SQLite table rebuild. Tracked by the AC itself, not deferred.
- Reference: `project-context.md` Rule 24; origin retro-19-P1.

### File List

_(fill in at implementation)_

---

## Open Questions for Alexyu (non-blocking — the dev proceeds with the stated ruling)

1. **Table name `subtitle_runs` (plural) vs the architecture's prose `subtitle_run`.** I ruled plural: Rule 6 mandates `snake_case plural`, `project-context.md` is the bible, and the architecture's singular is prose rather than a stamped contract. Renaming after merge means a follow-up migration, so say now if you'd rather match the doc.
2. **`no_text_source` and `skipped` as `subtitle_status` values on the media row.** They are terminal *pipeline* outcomes living in a column whose other values are *search* outcomes. The alternative is keeping them on `subtitle_runs` only and leaving the media row at `not_found`. I followed the architecture (FR5 explicitly needs the media row to express it, and the library filter reads that column) — flagging because it is the one place the two vocabularies genuinely mix.
3. ~~**Frontend badge work.**~~ ✅ **RESOLVED 2026-07-27 (Alexyu): M1 ships with the badges.** Promoted out of backlog into **sub-1-7a** (UX spec screen, Sally) → **sub-1-7b** (frontend). Both `ready-for-dev`; Epic 1 in `epics-subtitle-pipeline.md` updated. sub-1-2 is unaffected and does not wait on them.
