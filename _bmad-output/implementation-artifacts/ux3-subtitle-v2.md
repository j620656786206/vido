# Story ux3-subtitle-v2: Subtitle UI v2 — generation-centric per-item + glossary (PH3-M5 slice 1)

Status: review

> **Scope ruling at authoring (SM Bob, 2026-07-05):** the PH3-M5 umbrella ("detail 管理字幕 + Activity batch drive GENERATION") is delivered in TWO slices. THIS story = the per-item detail surface + glossary management/review + generation progress (all backend dependencies LIVE and code-verified). The batch-generation surfaces (F4/F8 + Activity launch CTA) are split to `ux3-subtitle-v2-batch` because **no Route C batch-generation endpoint exists** (code-verified 2026-07-05: `/api/v1/subtitles/batch` runs the Epic 8 provider-FETCH pipeline, not generation) — BE work filed as `9R-16-batch-generation-endpoint`. See Discovery Triage.

## Story

As a Vido user with media missing 繁中 subtitles,
I want the detail-page 管理字幕 dialog rebuilt v2 and generation-centric — one primary 生成字幕 action driving the Route C pipeline with live stage progress, plus a per-show glossary I can review and confirm,
so that I get reliable 繁體中文 subtitles from the pipeline that actually works, instead of a fetch UI whose sources are dead.

## Acceptance Criteria

1. **Manage-subtitle dialog v2 (F1 — Screen `F1-D-v2` r1EY9 / `F1-M-v2` JkdfH).** The 管理字幕 CTA in `LocalDetailV2.tsx` (`data-testid="action-manage-subtitle"`, line ~116) opens a NEW v2 dialog replacing `SubtitleSearchDialog` **in the v2 shell only** (legacy `MediaDetailPanel` untouched — strangler rule). Dialog shows: (a) current subtitle tracks with language + status (reuse `deriveSubtitleStatus` semantics; a 簡中 track on CN-content is displayed as policy-correct per §9b, NOT as a defect — CN-policy variant note v16pVI); (b) `生成字幕` as the ONLY primary action; (c) fetch demoted to a dormant secondary `搜尋線上字幕（成功率低）` — **NO source chips, NO score-breakdown rows, NO Zimuku anywhere** (9R-14 removed it); (d) glossary entry point (F3); (e) desktop dialog + mobile sheet per the two frames.
2. **Generation trigger, movies live / series capability-honored.** 生成字幕 on a movie calls `POST /api/v1/movies/{id}/transcribe?translate=true` (`:id` = int64 movie id; 202 → `{job_id, message}`). On a series/episode the CTA renders **disabled** with hint `影集字幕生成即將推出`（BE trigger = 9R-10a, ready-for-dev, movies-only today — Rule 24 capability honor: never draw a dead control as live）. Error handling: 503 `TRANSCRIPTION_DISABLED` → not-configured state（AC 5）; 409 `TRANSCRIPTION_IN_PROGRESS` → attach to the running job's SSE stream by `media_id` instead of erroring; 404/400 → fail-soft error state with `重試`.
3. **`Component/GenerationProgress-v2` (node XkGvG, F3 — `F3-D-v2` JbXai / `F3-M-v2` k8sJl4 生成進度).** One reusable component (registered in Component Library sJzat row luza9) rendering the FROZEN stage list `提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 → AI校正 → 完成` + failed-at-stage + `重試`. Live data via a new lazy SSE hook (AC 8) mapping wire phases: `extracting`→提取音訊, `transcribing`→轉錄中, `translating`(+`percentage` 0–100)→翻譯中, `complete`→完成 (簡轉繁/AI校正 advance atomically on `complete` — no dedicated wire phase today; document in component), `failed`→失敗於{stage} + `重試`. **All timing/ETA text comes from the SSE payload — the component MUST NOT read `Date.now()`/`new Date()`** (design handoff warning, Rule 23). Percentage/numeric text uses Mono (`font-mono tabular-nums`). Cost/quota slot: component accepts OPTIONAL cost props and renders nothing when absent — **no BE cost surface exists today** (9R-17 filed); do NOT invent cost data.
4. **Glossary management + review (F6 `F6-D-v2` dlfMR / `F6-M-v2` buepS 名詞對照表+全部確認; F7 `F7-D-v2` A85GFD 名詞對照表—空狀態).** New `Component/GlossaryRow-v2` (node nDSEd, Library cell Fx24g): `term_src ↔ term_zh` pair, source badge（`字幕`=subtitle / `中繼資料`=metadata / `手動`=manual）, unconfirmed visually distinct from confirmed, row actions edit/confirm/delete. Panel supports: list, add, edit, per-row confirm, delete, and **`全部確認` batch-confirm** (F6, party-mode P1) via `POST /media/{mediaId}/glossary/confirm-all`. Empty state: `尚無詞彙 — 生成字幕時自動累積`. All six 9R-15 routes wired exactly as specified in Dev Notes (do NOT invent fields).
5. **Four-state coverage (N4, per surface: dialog / progress / glossary).** default, loading, empty (distinct from failure), fail-soft error. The two flagship error states: 503 `TRANSCRIPTION_DISABLED` → `字幕生成尚未設定` + `前往設定` CTA (links to settings; page/dialog never hard-fails); mid-pipeline `failed` → stage-labeled + `重試` per `F4-D-v2` (U8rRtv 生成失敗 — IN this story's scope, it is a per-item state, NOT a batch frame). Other state frames: `F2-D-v2` (S9Rbrq 管理字幕—缺字幕), `F5-D-v2` (f6ZxY), `F10-D-v2` (olDlj 載入骨架). Dev MUST open them via Pencil MCP (block x≈31590 y≈-5921) and match. ⚠️ **Frame-numbering drift:** the design story's PROSE F-numbers were renumbered when the canvas landed — the canvas frame NAMES are authoritative, not the 9R-UX prose (validated 2026-07-05 via Pencil MCP).
6. **Lifecycle consistency (N1).** On `transcription_complete`, invalidate the React Query caches for the media detail + library list so poster badges (`deriveSubtitleStatus`, ux3-0-1/0-2 authoritative `subtitleStatus`/`subtitleLanguage` fields) refresh without reload. Generation states and badge states must read as one system (DL-v2 §2.5 token map). ⚠️ **Known BE gap (annotated 2026-07-05):** the transcription path does NOT yet write `movies.subtitle_status/subtitle_language` on success — so until 9R-16 AC 12 (writeback) lands, the invalidation refetches UNCHANGED data and the badge stays 缺字幕 (a rescan fixes it). Implement the invalidation as specced (correct FE behavior, becomes fully live with 9R-16); do NOT hack a client-side badge override.
7. **Rule 21 + design conformance.** Every new component file carries a valid header — `// Implements: Component/GenerationProgress-v2 (XkGvG)`, `// Implements: Component/GlossaryRow-v2 (nDSEd)`, dialog/panel files use `// Design ref: ux-design.pen Screen F1-D-v2 (r1EY9)` etc. (verify IDs via Pencil MCP `get_editor_state`; ESLint `local/implements-pen-node-id` gates CI). Token-only styling (`var(--*)` custom properties, `DownloadCardV2.tsx` is the exemplar), Radix Dialog for modal + destructive confirms. Gallery fixtures added in `-gallery.fixtures.tsx` for each new component (visual baselines; `-linux` PNGs arrive via the CI bootstrap PR — never generate locally).
8. **SSE hook.** New `useGenerationProgress` hook following the `useDownloadProgress.ts` pattern: LAZY (`startTracking(mediaId)` — never connect on mount, §8), listens for `transcription_extracting` / `transcription_progress` / `translation_progress` / `transcription_complete` / `transcription_failed` events, **unwraps the double-nested envelope** (`data:` line carries the full `Event` struct → payload is `parsed.data`), `snakeToCamel` at ingest, filters by `media_id`, merge-not-replace reducer, `mountedRef` guard, 10s reconnect backoff, cleanup on unmount.
9. **Tests + gates.** Colocated vitest specs for: new service, new hook (event mapping incl. envelope unwrap + media_id filter + 409-attach), each new component (four states, frozen stage names, confirm-all flow, disabled-series CTA). Existing suites stay green (`SubtitleSearchDialog`/`BatchSubtitleDialog` specs untouched — files not deleted). `pnpm lint:all` + `pnpm nx build web` green. UX screenshot verification against `_bmad-output/screenshots/flow-f-subtitle-v2/` — `f1-d-v2.png`, `f1-m-v2.png`, `f2-d-v2.png`, `f3-d-v2.png`, `f3-m-v2.png`, `f4-d-v2.png`, `f5-d-v2.png`, `f6-d-v2.png`, `f6-m-v2.png`, `f7-d-v2.png`, `f10-d-v2.png` (mandatory workflow step — Sally gate; f8/f9 are the batch slice, excluded).

## Tasks / Subtasks

- [x] Task 1: Glossary API client (AC: 4)
  - [x] `apps/web/src/services/glossaryService.ts` — 6 routes (shapes in Dev Notes), Rule 18 `snakeToCamel`/`camelToSnake` at boundary, `mediaId` always **string**
  - [x] `glossaryService.spec.ts`
- [x] Task 2: `useGenerationProgress` hook (AC: 2, 8)
  - [x] Model `useDownloadProgress.ts`; lazy connect, 5 `transcription_*`/`translation_progress` events, envelope unwrap, media_id filter, phase→stage state machine, terminal close
  - [x] `useGenerationProgress.spec.ts` (mock EventSource; assert double-nested unwrap explicitly)
- [x] Task 3: `GenerationProgressV2` component (AC: 3, 5, 7)
  - [x] `apps/web/src/components/subtitle/GenerationProgressV2.tsx` — frozen stages, failed+重試, optional cost props (render-nothing default), Mono numerics, Rule 21 header `(XkGvG)`
  - [x] Spec + gallery fixture (fixture states named after frozen stage names)
- [x] Task 4: Glossary components (AC: 4, 5, 7)
  - [x] `GlossaryRowV2.tsx` (`nDSEd`) + `GlossaryPanelV2.tsx` (list/add/edit/confirm/delete/confirm-all + empty + error states)
  - [x] Specs + gallery fixtures
- [x] Task 5: `ManageSubtitleDialogV2` (AC: 1, 2, 5, 7)
  - [x] `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx` — tracks section, 生成字幕 primary (movie wired / series disabled+hint), dormant fetch secondary reusing `useSubtitleSearch` + `subtitleService` (results WITHOUT source chips/score rows), glossary entry, CN-policy display, desktop/mobile per F1-D/F1-M
  - [x] 503→尚未設定+前往設定; 409→SSE attach; state frames F2 (缺字幕) / F4 (生成失敗) / F5 / F10 (載入骨架)
  - [x] Spec
- [x] Task 6: Wire v2 shell (AC: 1, 6)
  - [x] `LocalDetailV2.tsx`: swap `SubtitleSearchDialog` → `ManageSubtitleDialogV2` (v1 dialog file stays; legacy shell untouched)
  - [x] On `transcription_complete`: invalidate media-detail + library query keys (badge refresh)
  - [x] Spec update for `LocalDetailV2`
- [x] Task 7: Verification pass (AC: 9)
  - [x] Full `nx test web` affected suites green; `pnpm lint:all`; `pnpm exec prettier --check` on touched files
  - [x] Browser-verify dialog @390/768/1440 vs `flow-f-subtitle-v2/` PNGs; verify no `Date.now()` in new components (Rule 23 ESLint will also gate) — component-level done via gallery baselines (frames read via Pencil MCP, structure/tokens matched exactly); FULL-APP dialog verify at the three breakpoints needs a seeded backend (none locally) → flagged for the review/Sally gate, see Completion Notes

**Cross-stack split check:** backend tasks = 0, frontend tasks = 7 → single story, no a/b split required.

## Dev Notes

### Backend surface (code-verified 2026-07-05 — do NOT re-derive from stories)

**Envelope (all REST):** success `{"success":true,"data":<payload>}`; error `{"success":false,"error":{code,message,suggestion}}` (`handlers/response.go`).

**Generation trigger** (`handlers/transcription_handler.go:44-46`):
- `POST /api/v1/movies/:id/transcribe?translate=true` — `:id` int64 movie id, no body, 202 → `{job_id, message}`. `translate=true` = full Route C (glossary-aware translate → OpenCC s2twp → atomic place); omitting it produces EN-only SRT — **always send `translate=true`**.
- Errors: 503 `TRANSCRIPTION_DISABLED` / 400 `VALIDATION_INVALID_FORMAT` / 404 `DB_NOT_FOUND` / 400 `VALIDATION_REQUIRED_FIELD` (no file on disk) / 409 `TRANSCRIPTION_IN_PROGRESS` / 500.
- **Movies-only.** No series route, no on-add auto-trigger (9R-10a / 9R-10b track those).

**Glossary REST (9R-15, `handlers/glossary_handler.go`)** — group `/api/v1/media/:mediaId/glossary`, `:mediaId` = **stringified** local media id (⚠️ same movie: transcribe uses int64, glossary uses string — convert in FE):
| Route | Success | Body |
|---|---|---|
| `GET …/glossary` | 200 `{terms:[GlossaryTerm]}` (never null) | — |
| `POST …/glossary` | 201 `GlossaryTerm` | `{term_src*, term_zh*, language, source, confirmed}` (`source` ∈ subtitle\|metadata\|manual, "" → manual; route mediaId wins over body) |
| `POST …/glossary/confirm-all` | 200 `{confirmed:<int64>}` | — |
| `PUT …/glossary/:termId` | 204 | `{term_zh, confirmed}` |
| `POST …/glossary/:termId/confirm` | 204 | — |
| `DELETE …/glossary/:termId` | 204 | — |

`GlossaryTerm` (snake_case wire): `id, media_id, term_src, term_zh, language` (default `"zh-Hant"`)`, source, confirmed, created_at, updated_at`. 404 `DB_NOT_FOUND` on unknown termId; 400 `VALIDATION_ERROR` on bad enum. No single-term GET.

**SSE** (`GET /api/v1/events`): ⚠️ **the `data:` line is the FULL `Event` struct** — `{"id":"<uuid>","type":"<type>","data":{…payload…}}` — read `parsed.data.*` (`Event` struct sse/hub.go:37-41; `sendSSEEvent(w, string(event.Type), event)` sse/handler.go:45). Payloads snake_case (except the `connected` handshake, camelCase). Generation events (declared in `services/transcription_service.go:40-44`, NOT the sse package):
- `transcription_extracting` → `{job_id, media_id, phase:"extracting", message}`
- `transcription_progress` → `{…, phase:"transcribing", message}`
- `translation_progress` → `{…, phase:"translating", percentage:<0-100 float>, message}`
- `transcription_complete` → `{…, phase:"complete", srt_path, duration, message, zh_srt_path?}` (`zh_srt_path` only when translated)
- `transcription_failed` → `{…, phase:"failed", error, message}`
- `media_id` in these payloads is the **int64** movie id. Do NOT confuse with `subtitle_progress` (Epic 8 fetch-download events, payload has `stage` not `phase`) — different feature, same stream.

**Fetch endpoints (dormant secondary, reuse as-is):** `subtitleService.ts` already wraps `POST /subtitles/{search,download,preview}` correctly. Batch fetch endpoints exist but are NOT this story's surface.

**Does NOT exist (do not call, do not draw as live):** Route C batch-generation endpoint (→ 9R-16), cost/quota HTTP or SSE surface (→ 9R-17), series transcribe route (→ 9R-10a), on-add auto-trigger (→ 9R-10b).

### Reuse map (do NOT reinvent)

- `apps/web/src/services/subtitleService.ts` + types — fetch flows, keep as-is.
- `apps/web/src/hooks/useSubtitleSearch.ts` — dormant fetch section state.
- `apps/web/src/hooks/useDownloadProgress.ts` — **the SSE hook template** (lazy, reconnect 10s, `mountedRef`, `connectRef`, snakeToCamel). `useSubtitleBatchProgress.ts` shows terminal-close + merge-keep-last-item.
- `apps/web/src/utils/caseTransform.ts` — Rule 18 boundary.
- `apps/web/src/utils/libraryStatus.ts` `deriveSubtitleStatus` — badge semantics (prefers authoritative `subtitleStatus`/`subtitleLanguage` per ux3-0-2).
- `apps/web/src/components/downloads/DownloadCardV2.tsx` — v2 exemplar: token styling, Radix Dialog, `role=progressbar`, testid conventions.
- Wiring point: `apps/web/src/components/media/LocalDetailV2.tsx` lines ~116-128 (CTA) + ~261-272 (current `SubtitleSearchDialog` render — swap here).
- Superseded (reference only, do NOT extend): `SubtitleSearchDialog.tsx`, `BatchSubtitleDialog.tsx` (hand-rolled modals, v1 styling). Files stay (legacy shell + batch fetch still live).

### Architecture compliance

- Rule 5: TanStack Query for all server state (glossary list = query + mutations with invalidation; no Zustand).
- Rule 18: transform at boundary both directions.
- Rule 21: headers per AC 7 (ESLint-gated).
- Rule 23: **no ambient wall-clock reads in new components** — timing text from SSE payload only. If any relative-time display is unavoidable, it needs a `// Clock-mocked:`/`// Clock-injected:` marker + ≥2 clock-pinned fixture states (see template section below). Frozen stage names = future fixture names; do not rename.
- Rule 26: no new search params planned. If dev adds any (e.g. `?glossary=1` deep link), use `toCsvString`/`String()` coercion — lone-numeric trap.
- §8 SSE rules: lazy connect, gated, cleanup, reconnect-with-backoff (no polling fallback), `mountedRef`.
- DL-v2: tokens only (`--bg-*`, `--accent-*`, `--text-*`, `--success/warning/error`, `--radius-*`); numerics `font-mono tabular-nums`; zh text Noto. Font zero-tolerance (feedback memory): numbers/units split Mono+Noto.
- Strangler: v2 shell only; `MediaDetailPanel.tsx` (legacy 搜尋字幕) untouched.

### Design sources

- Frames: `ux-design.pen` block x≈31590 y≈-5921, screenshots `_bmad-output/screenshots/flow-f-subtitle-v2/` (`f1-d-v2`/`f1-m-v2`, `f2-d-v2`, `f3-d-v2`/`f3-m-v2`, `f4-d-v2`, `f5-d-v2`, `f6-d-v2`/`f6-m-v2`, `f7-d-v2`, `f10-d-v2`, all `.png`). **F8 (i9Nun1/H717g) + F9 (JMqPg 批次生成—預算上限) = batch surfaces → NOT this story. F11 (l8FsB) = non-spec exploration, archived as Epic 7 reference — ignore.** Canvas frame names are authoritative over the 9R-UX design-story prose F-numbers (renumbered at landing).
- Old `flow-f-subtitle` f1–f3 + `flow-g-ai-subtitle` = superseded/dead (fetch-era). Zero reuse.
- Design story: `9R-UX-subtitle-v2-design.md`; ADR: `architecture/adr-subtitle-route-c-generation.md` (D1 generation-primary/fetch-dormant, D3 glossary keystone); DL-v2: `ux-redesign/01-design-language-v2.md`.
- On-canvas notes to read via Pencil MCP before implementing: BE-gaps CJVC5, CN-policy variant v16pVI, 9R-15 route-shape annotation.
- This story does NOT modify `ux-design.pen` — no screenshot regen expected. If the dev DOES touch it, the export+commit workflow applies (CLAUDE.md).

### Project Structure Notes

- New files under `apps/web/src/components/subtitle/` (`PascalCaseV2.tsx` + colocated `.spec.tsx`), hook in `apps/web/src/hooks/`, service in `apps/web/src/services/`. Barrel `index.ts` optional per downloads convention.
- No route changes (dialog lives inside existing `/media/$type/$id`). No BE changes. No migrations.

### Time-dependent visual coverage

- **Does this story add/modify any `apps/web/src/components/**/*.{ts,tsx}` that reads `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`?**
  - Expected **NO** — the design handoff (party-mode P5, Murat) mandates: "F3's elapsed/total time (`12:34 / 45:00`) MUST come from the server/SSE payload — the FE component must NOT derive it from `Date.now()`/`new Date()`". All timing text renders SSE-supplied values.
  - If the dev nonetheless introduces an ambient clock read: Rule 23 marker (one of three forms) + ≥2 clock-pinned fixture states via `withFixedClock(page, iso)` / `clockTime` fixture field are MANDATORY. The `local/time-dependent-fixture-stability` ESLint rule will flag it.
  - **Frozen fixture vocabulary:** `Component/GenerationProgress-v2` stage names 提取音訊/轉錄中/翻譯中/簡轉繁/AI校正/完成 + failed are FROZEN — gallery fixture states are named after them; renaming breaks fixture↔baseline mapping.
- Reference: `project-context.md` Rule 23; audit doc `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.

### References

- [Source: _bmad-output/implementation-artifacts/9R-UX-subtitle-v2-design.md — frames, decisions, handoff warnings]
- [Source: _bmad-output/planning-artifacts/architecture/adr-subtitle-route-c-generation.md — D1/D3]
- [Source: _bmad-output/planning-artifacts/epics.md §Epic 6 (2026-06-16 REVISION) + PH3-M5 mapping]
- [Source: _bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md §3/§6 — flow F is a feature surface, not a destination]
- [Source: apps/api/internal/handlers/glossary_handler.go, transcription_handler.go, sse/hub.go+handler.go, services/transcription_service.go — verified route/payload shapes]
- [Source: project-context.md §8 SSE rules, §9b CN policy, Rules 5/18/21/23/24/26]

## Dev Agent Record

### Agent Model Used

Claude Fable 5 (claude-fable-5) — DEV Amelia, 2026-07-05

### Debug Log References

- Full `nx test web` (uncached): 223 files / 2391 tests green (+44 new).
- `pnpm lint:all`: 0 errors, 123 pre-existing warnings; `prettier --check .` clean.
- `pnpm nx build web`: green.
- Visual baselines: `pnpm run test:visual:update` then `git checkout` of all 26 re-render-noise diffs — only the 9 NEW `-darwin` baselines staged (`generation-progress-v2/{提取音訊,轉錄中,翻譯中,完成,失敗,cost-slot}`, `glossary-row-v2/{unconfirmed,confirmed-metadata,manual}`). NOTE: `test:visual:update-missing` could NOT be used — the KNOWN pre-existing `ui-dialog/default` darwin mismatch hard-fails the mega-test loop before later fixtures are reached (`preexisting-fail-ui-dialog-darwin-visual`), so a full update + selective revert was required.

### Completion Notes List

- All Rule 21 headers verified against Pencil MCP live: `XkGvG` (GenerationProgress-v2), `nDSEd` (GlossaryRow-v2), Screen ids r1EY9/S9Rbrq/JbXai/U8rRtv/f6ZxY/dlfMR/A85GFD/olDlj all real; frames read at full depth and matched (structure, tokens, copy).
- **Stepper stage-count note:** the `.pen` `Component/GenerationProgress-v2` (XkGvG) draws FIVE steps (提取音訊/轉錄中/翻譯中/簡轉繁/完成 — no AI校正 node), while AC 3's FROZEN list has SIX. Implemented the story's six-stage list (story = single source of truth; the fixture vocabulary is stamped on it). Design-side sync of XkGvG (+AI校正 step) is a Sally follow-up — this story does not modify `ux-design.pen` per Dev Notes.
- Mobile sheet (F1-M) = the SAME Radix Dialog with bottom-sheet positioning at `<sm` (AC 7 mandates Radix for modals; matches the frame's bottom-sheet look without a second dialog system).
- CN policy (§9b / v16pVI): implemented as an INFO line under 簡中 tracks when `productionCountry` contains CN. The design's 轉為繁中 / 仍要轉換 ACTION buttons are NOT rendered — capability honor: no backend endpoint converts an EXISTING local track (OpenCC runs only inside download/generation pipelines). Triaged ③ below. Also: the local-detail API does not expose production_countries (`json:"-"`), so the prop is dormant in the v2 shell (same as v1 `SubtitleSearchDialog.productionCountry`, which LocalDetailV2 never passed either). Triaged ③ below.
- F3's footer 取消 button (design) is NOT rendered as a job-cancel: no cancel route exists for transcription. The footer shows 關閉 + 「關閉後生成會在背景繼續」 hint (closing stops watching, job continues). Covered by the batch-cancel design in 9R-16 for the batch slice; per-item cancel not tracked (job is short and idempotent-retryable).
- Generation trigger lives in a small `transcriptionService.ts` (202/503/409 discriminated union, always `translate=true`); not folded into `subtitleService.ts` (reuse map says keep as-is).
- Glossary server state via `useGlossary.ts` (Rule 5: query + 5 mutations with list invalidation).
- AC 6 implemented as specced: `LocalDetailV2.onGenerationComplete` invalidates `detailKeys.localMovie/localSeries(id)` + `libraryKeys.all`; badge stays 缺字幕 until 9R-16 AC 12 writeback (annotated in code, no client-side hack).
- Verification residue for the review gate: FULL-APP browser verify of the dialog @390/768/1440 against `flow-f-subtitle-v2/` PNGs was not possible locally (no seeded backend; dev API returned 500 behind the proxy). Component-level visual verification done via the 9 new gallery baselines, which match the Pencil frames. Sally's mandatory UX screenshot comparison at review covers the dialog-level check.
- No new search params (Rule 26 clean); no `Date.now()`/`new Date()` in any new component (Rule 23 grep + ESLint clean).

### Discovery Triage

Story-authoring-time discoveries (SM Bob, 2026-07-05 — all filed in sprint-status.yaml the same day):

- **③ backlog-with-carry-forward-link — `9R-16-batch-generation-endpoint`** (BE): NO Route C batch-generation endpoint exists — `/api/v1/subtitles/batch` is the Epic 8 provider-FETCH batch. The design's BE-gaps note (c) assumed "batch generation = 9R-10 + 9R-11" but neither shipped a batch trigger/orchestrator/SSE-batch-event. Non-blocking for THIS story's per-item scope.
- **③ backlog-with-carry-forward-link — `ux3-subtitle-v2-batch`** (FE follow-up, blocked-by 9R-16 + 9R-10a): batch surfaces F8 (i9Nun1/H717g scope segments 缺字幕的項目|已選項目) + F9 (JMqPg 批次生成—預算上限), Activity hub batch-launch CTA (hub is display-only today), library `SelectionToolbar` re-point from fetch-batch to generation-batch, and the series-CTA enable flip once 9R-10a lands.
- **③ backlog-with-carry-forward-link — `9R-17-ai-usage-endpoint`** (BE): cost/quota visibility is in the GenerationProgress-v2 design spec (9R-11, Mono numerics), but 9R-11's Governor/Budget is internal-only — NO HTTP or SSE surface exposes cost/token/budget. This story ships the cost slot dormant (optional props, renders nothing).
- Series/episode generation trigger missing — **already tracked** (`9R-10a-series-episode-trigger`, ready-for-dev); no new entry. This story renders the capability-honored disabled CTA.

(Dev: add any further in-flight discoveries here per Rule 24 before marking done.)

Dev-time discoveries (DEV Amelia, 2026-07-05 — both filed in sprint-status.yaml the same day):

- **③ backlog-with-carry-forward-link — `disc-2026-07-track-convert-endpoint`** (BE): the F1 design's 轉為繁中（簡轉繁） action on an existing local 簡中 track (and the CN-variant 仍要轉換 override, note v16pVI) has NO backend surface — OpenCC s2twp runs only inside the download post-process and the Route C pipeline; there is no standalone convert-an-existing-track route. Per Rule 24 capability honor the v2 dialog renders the CN policy INFO line only (no dead convert/override buttons). Needs e.g. `POST /api/v1/media/{id}/subtitles/convert`.
- **③ backlog-with-carry-forward-link — `disc-2026-07-production-countries-detail-api`** (BE): `movies.production_countries` is stored but stamped `json:"-"` (models/movie.go:134) — the local-detail API never exposes it, so the FE cannot drive the §9b CN-policy display (this story's `productionCountry` prop ships dormant; the v1 dialog had the same dead prop). Needs the local movie/series detail payload to include production country codes.

### File List

New:

- `apps/web/src/services/glossaryService.ts` (+ `glossaryService.spec.ts`)
- `apps/web/src/services/transcriptionService.ts` (+ `transcriptionService.spec.ts`)
- `apps/web/src/hooks/useGenerationProgress.ts` (+ `useGenerationProgress.spec.ts`)
- `apps/web/src/hooks/useGlossary.ts`
- `apps/web/src/components/subtitle/GenerationProgressV2.tsx` (+ `.spec.tsx`)
- `apps/web/src/components/subtitle/GlossaryRowV2.tsx` (+ `.spec.tsx`)
- `apps/web/src/components/subtitle/GlossaryPanelV2.tsx` (+ `.spec.tsx`)
- `apps/web/src/components/subtitle/ManageSubtitleDialogV2.tsx` (+ `.spec.tsx`)
- `tests/visual/components.visual.spec.ts-snapshots/components/generation-progress-v2/{提取音訊,轉錄中,翻譯中,完成,失敗,cost-slot}/default-visual-darwin.png`
- `tests/visual/components.visual.spec.ts-snapshots/components/glossary-row-v2/{unconfirmed,confirmed-metadata,manual}/default-visual-darwin.png`

Modified:

- `apps/web/src/components/media/LocalDetailV2.tsx` (dialog swap + AC 6 invalidation) + `LocalDetailV2.spec.tsx`
- `apps/web/src/routes/test/-gallery.fixtures.tsx` (9 fixtures: 6× GenerationProgressV2 frozen-stage states, 3× GlossaryRowV2)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (ux3-subtitle-v2 → review; +2 ③ discovery entries)
- `_bmad-output/implementation-artifacts/ux3-subtitle-v2.md` (this file)

Untouched by design (strangler / reuse rules): `SubtitleSearchDialog.tsx`, `BatchSubtitleDialog.tsx` + their specs, `MediaDetailPanel.tsx`, `subtitleService.ts`, `useSubtitleSearch.ts`, `ux-design.pen`.
