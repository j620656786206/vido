# Epic 13: Request System
**Phase:** Phase 3 — Automation & Integration

Users can one-click request movies or TV shows from explore or detail pages, request specific seasons or individual episodes, track request status through a clear pipeline (pending → searching → downloading → completed → failed), optionally route requests through Sonarr/Radarr APIs for automated downloading, and auto-trigger subtitle search upon download completion. This replaces the need for a separate Overseerr/Jellyseerr instance.

**v4 Feature IDs covered:** P3-001, P3-002, P3-003, P3-004, P3-005

**Dependencies on Completed Work:**
- Epic 2: TMDB media identification (for request targets)
- Epic 4: qBittorrent integration pattern (reusable for download-status monitoring)
- Epic 8: Subtitle engine (for auto-trigger on completion)
- Architecture §7 `DVRPlugin` interface + §8 SSE Hub (status push)

**Success Criteria:**
- Request-to-download pipeline initiates in <30s when Sonarr/Radarr is configured
- Request status accurately reflects real download state
- Auto-subtitle trigger fires within 60s of download completion

---

## 🏗️ Architecture-grounded findings (verified 2026-07-01 — reshapes the breakdown)

A code audit (party-mode review, Winston) changed the sequencing. The "Vido built-in download flow" **cannot fulfil a request today** because two pieces are absent:

1. **No torrent source.** No indexer exists in `apps/api` (`Jackett`/`Prowlarr`/`torznab` = 0 hits) — that is **Epic 15 (Indexer Integration), not started**.
2. **No add-to-qBittorrent capability.** The qBittorrent client (`internal/qbittorrent/client.go`) is **read-only** (Login/Ping/TestConnection/GetTorrents/GetTorrentDetails — no add-torrent).
3. **`DVRPlugin` is a design only.** No `internal/plugins/` dir exists; the §7 `DVRPlugin` interface + Sonarr/Radarr clients are **net-new to build**.
4. **No `requests` data model** yet.

**Decision (Alexyu, 2026-07-01):** do **all** of G-1…G-5 (nothing deferred), but **promote G-4 (Sonarr/Radarr) into the artery as the fulfilment engine** — Sonarr/Radarr do the indexer search + grab + hand-off to the download client, which is exactly why Overseerr/Jellyseerr depend on *arr. The pure-built-in download path is **isolated as an Epic-15-dependent backlog branch** so Epic 13 is not blocked on un-started indexer work.

**Re-sequenced artery:** `13-1 request` → **`13-4 *arr fulfilment engine`** → `13-3 status tracking` → `13-2 partial request` → `13-5 auto-subtitle`.

---

## Data model (foundation — built in 13-1a)

`requests` table (migration): `id`, `tmdb_id`, `media_type` (movie|tv), `title`, `status` (enum: `pending`|`searching`|`downloading`|`completed`|`failed`), `fulfilment_source` (`arr`|`builtin`), `external_id` (the Sonarr/Radarr id once submitted), `seasons`/`episodes` (JSON, for partial — 13-2), `requested_at`, `updated_at`, `error_message`. Status enum is the single source of truth the pipeline (13-3) and FE render against — mirrors the unified lifecycle moat (`想要 → 下載中 x% → 整理中 → 已入庫`).

---

## Sequenced story breakdown

> Each cross-stack story **splits into `a` (backend) / `b` (frontend)** at `create-story` time per the Cross-Stack Split rule. UI stories are **design-first** — a single design story (`13-0`) covers all request surfaces (mirrors the `ux3-4-1` downloads design precedent).

### 13-0 — Requests design (`.pen` flow-g-requests) · design-first · Owner: ux-designer
Covers ALL request UI in one flow: the **request entry** (explore/detail one-click button + lighting up the **Discover PH3-R2 reserved entry** from ux3-3-1), the **partial-request season/episode tree** (G-2, highest design risk per Winston — draw with G-1), and the **request status list page** (G-3). v2 Design Language, N4 four-states, token-only. **GATE A** for all `b` (frontend) stories below.

### 13-1 — One-click request (G-1 / P3-001) · artery #1
- **Story:** As a user browsing explore/detail, I want a one-click 想要 button that records a request for a movie/series, so that I can ask Vido to acquire it without leaving the page.
- **13-1a (BE):** `requests` table + migration; `POST /api/v1/requests` (resolve TMDB target via Epic 2; create `pending` row); `GET /api/v1/requests`; `RequestRepository` + `RequestService`. No fulfilment yet — records intent.
- **13-1b (FE):** request button on explore/detail (movie/TV differentiation); light up the Discover reserved-inert Requests entry (ux3-3-1 PH3-R2) → live. **Design: 13-0.**
- **AC highlights:** request persists as `pending`; duplicate-request guard; button reflects already-requested state; Rule 18 case-boundary; `[@contract-v1]` on the request create/list shape (13-1b + later FE ack).

### 13-4 — Sonarr/Radarr DVR plugin = fulfilment engine (G-4 / P3-004) · artery #2 · BACKEND-ONLY · **LARGE (may split 13-4a/13-4b)**
- **Story:** As a NAS owner with *arr configured, I want requests routed to Sonarr (TV) / Radarr (movies) so they are actually searched, grabbed, and downloaded — the realistic fulfilment path.
- **Scope:** build `internal/plugins/` + the `DVRPlugin` interface (§7: `AddMovie`, `AddSeries`, `GetQueue`, `Name`, `TestConnection`); **Radarr client** (movies) + **Sonarr client** (TV); plugin config in SQLite (per §7 — `TestConnection()` must pass before save); health-check scheduler (default 60s). On a `pending` request being fulfilled via *arr → call `AddMovie/AddSeries`, store `external_id`, move to `searching`.
- **Split note:** likely `13-4a` (plugin infra + `DVRPlugin` + Radarr + config/health) / `13-4b` (Sonarr + season-aware AddSeries, pairs with 13-2). Backend-only either way → no cross-stack split, but size-split.
- **AC highlights:** graceful degradation when *arr unavailable (Rule: feature disabled, request stays `pending` with a clear reason); Rule 7 new error prefix (e.g. `DVR_*` — extend the §Rule-7 list + code-review sync); config never logged in clear (slog sanitize).

### 13-3 — Request status tracking + SSE (G-3 / P3-003) · artery #3
- **Story:** As a user, I want a request list page showing each request's live status (pending/searching/downloading/completed/failed) so I know what's happening without guessing.
- **13-3a (BE):** status pipeline — a **gated server-side poller** that reconciles `requests` against the *arr queue (`DVRPlugin.GetQueue`) and the existing qBittorrent monitor (Epic 4), derives the status enum, and pushes `request_progress` SSE events. **Reuse the `ux3-4-2b` downloads-SSE broadcaster pattern verbatim** (`Hub.ClientCount()` gate + ticker + `Broadcast`). Add `EventRequestProgress` to the SSE hub.
- **13-3b (FE):** request status list page; lazy SSE consumer (mirror `useScanProgress` — never connect on mount, §8); N4 states. **Design: 13-0.**
- **AC highlights:** status reflects real *arr/qBT state; per-section fail-soft; `[@contract-v1]` on the `request_progress` payload (FE acks); SSE latency <30s success-criterion.

### 13-2 — Partial request: seasons / episodes (G-2 / P3-002) · artery #4
- **Story:** As a user, I want to request specific seasons or individual episodes of a series (not the whole show), so I only acquire what I need.
- **13-2a (BE):** extend `requests` (seasons/episodes JSON) + the create endpoint; map to Sonarr `AddSeries` with monitored-season/episode selection (depends on 13-4b Sonarr).
- **13-2b (FE):** season/episode tree selector on the TV request flow. **Highest design risk → designed in 13-0 alongside G-1.**
- **AC highlights:** whole-series vs partial differentiation; season/episode validity vs TMDB season data (Epic 2); the tree reflects already-owned/already-requested episodes.

### 13-5 — Auto-subtitle trigger (G-5 / P3-005, P1) · artery #5 · BACKEND-ONLY · thin
- **Story:** As a user, I want subtitle search to fire automatically when a requested download completes, so I don't have to manually trigger it.
- **Scope:** on the status pipeline (13-3a) transitioning a request to `completed`, invoke the Epic 8 subtitle engine search for the resolved media. Backend-only event wiring; reuse the existing subtitle batch/search service.
- **AC highlights:** fires within 60s of completion (success criterion); idempotent (no double-search on re-poll); honors the existing subtitle CN/zh-TW policy; failure is logged, never blocks the request from showing `completed`.

---

## 🔒 Backlog branch (isolated — NOT in the artery)

### 13-X — Built-in (no-*arr) download path · **BLOCKED by Epic 15**
The pure-Vido fulfilment path (request → Vido searches an indexer → Vido adds the torrent to qBittorrent) requires **two un-built capabilities**: **(1) indexer integration = Epic 15** (Indexer Integration, P3-016–019, not started) and **(2) a qBittorrent add-torrent capability** (the client is read-only today). Kept as an explicit Epic-13 backlog branch with a hard Epic-15 dependency so the *arr artery above is never blocked on it. Author its stories only after Epic 15 lands.

---

## Sequencing summary

```
13-0 requests-design (Sally, GATE A)
   ↓
13-1 one-click request (13-1a BE model+endpoint → 13-1b FE button)
   ↓
13-4 *arr fulfilment engine (13-4a infra+Radarr → 13-4b Sonarr)   ← promoted into artery
   ↓
13-3 status tracking (13-3a BE pipeline+SSE → 13-3b FE list)      ← reuses ux3-4-2b SSE pattern
   ↓
13-2 partial request (13-2a BE → 13-2b FE season/episode tree)
   ↓
13-5 auto-subtitle trigger (BE-only, hook on completion → Epic 8)

backlog: 13-X built-in download path  ── blocked by Epic 15 (indexer) + qBT add-torrent
```

**Cross-cutting:** the `requests.status` enum is the single source of truth all stories render against; `request_progress` SSE reuses the `ux3-4-2b` gated-broadcaster pattern; every UI story is GATE-A'd on `13-0`; `13-4` introduces a new Rule-7 `DVR_*` error prefix (sync `code-review/instructions.xml`).

---

> **Authoring status (2026-07-01):** epic-level breakdown only (this file). Implementation-ready stories are authored just-in-time via `sm create-story` when each reaches its turn; run `sm sprint-planning` to extract these stories into `sprint-status.yaml` for tracking. `epics.md` (the ux3 cascade) is intentionally untouched — Epic 13 is a separate PRD-v4 feature epic.
