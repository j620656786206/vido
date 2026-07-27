# Story sub-1.7b: Subtitle-status badge — render the 5 new pipeline states

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🟡 MEDIUM** · **FRONTEND-ONLY**
**Origin:** Alexyu ruling 2026-07-27 — *"前端 badge 讓 M1 出貨時就帶 badge"*. Promotes the lane-③ entry `backlog-subtitle-status-fe-rendering` (filed by sub-1-2) into M1 scope.
**Depends on:** **sub-1-7a merged** (the design spec is the verification target — do not start without `flow-j-specs/j2-d.png`) **and** sub-1-2's `[@contract-v1]` `SubtitleStatus` 9-value set.
**Blocks:** nothing. This is a leaf consumer.
**Cross-stack split check:** backend tasks = **0**, frontend tasks = 3 → single story (already the `b` half of a design/frontend split).

---

## Story

As a NAS owner,
I want the library, list, and episode rows to show what the subtitle pipeline actually did to each file,
so that an item the pipeline permanently declined is visibly distinguishable from one it simply hasn't reached yet.

---

## 🚨 Verify against the design, do not invent

sub-1-7a produces `_bmad-output/screenshots/flow-j-specs/j2-d.png`. **That screenshot is the acceptance target for every label, tint, and icon in this story.** The tables below are the *proposal* sub-1-7a started from; if Sally overruled any row, **the spec screen wins and this story's ACs are read as amended**. Check sub-1-7a's Completion Notes for the ratified table before writing code, and compare your rendered UI against `j2-d.png` before marking any task done (mandatory UX verification, dev workflow step 9).

---

## Acceptance Criteria

### AC #1 — `deriveSubtitleStatus` handles all 9 values; terminal states earn a badge, transient ones do not

**Given** `apps/web/src/utils/libraryStatus.ts:102-123`, **when** it receives one of the 5 new values, **then** it returns per the ratified spec — proposal:

| `subtitleStatus` | Returns | `steadyState` |
|---|---|---|
| `probing` / `extracting` / `translating` | **`null`** — transient; the Activity hub + SSE (sub-1-6) own in-flight progress | — |
| `no_text_source` | `{ label: '無字幕源', className: TINT.neutral }` | `false` |
| `skipped` | `{ label: '已略過', className: TINT.neutral }` | `false` |
| the 4 existing values | **unchanged** | unchanged |

**Ordering matters — put the new terminal branches in the right place.** The current function is a 4-step ladder: (1) `found` → language classification, (2) embedded-track inference, (3) `not_found`, (4) `null`. `no_text_source` and `skipped` are **authoritative engine verdicts about this file** and must be checked **before** step 2's embedded-track inference — otherwise a file with an image-only or `und` track infers `有字幕` from `subtitleTracks` and the pipeline's verdict is silently overridden. The three transient values must reach `null` **without** falling into step 2 for the same reason.

Add a short comment saying *why* the new branches sit above track inference — this is the non-obvious part and the next reader will otherwise "simplify" it back.

`deriveFromTracks`, `deriveLifecycleStatus`, `HANT`/`HANS`, and `TINT` are **unchanged**.

### AC #2 — `pickPosterBadge` surfaces the new terminal states on the grid

**Given** `pickPosterBadge` suppresses `steadyState` descriptors (the grid is an EXCEPTION signal, ux3-0-2), **when** an item is `no_text_source` or `skipped`, **then** the badge renders — both are non-steady by construction (AC #1), so **no change to `pickPosterBadge` itself is required**. Assert this with a test rather than editing the function.

Both `PosterCardV2.tsx` and `LibraryListRowV2.tsx` consume `deriveSubtitleStatus` and need **zero edits** — the derivation is centralized. If you find yourself editing either component, stop: the change belongs in `libraryStatus.ts`.

### AC #3 — `EpisodeList` row icons cover the 5 new values

**Given** `apps/web/src/components/media/EpisodeList.tsx:33-60`'s `SUBTITLE_STATUS` map, **when** it is extended, **then** every one of the 9 values has an entry — proposal:

| value | Icon | colour | `spin` | `aria-label` / `title` |
|---|---|---|---|---|
| `probing` | `Loader2` | `text-[var(--accent-text)]` | ✅ | `偵測字幕軌中` |
| `extracting` | `Loader2` | `text-[var(--accent-text)]` | ✅ | `抽取內嵌字幕中` |
| `translating` | `Loader2` | `text-[var(--accent-text)]` | ✅ | `翻譯字幕中` |
| `no_text_source` | `XCircle` | `text-[var(--text-muted)]` | — | `無可用的文字字幕軌` |
| `skipped` | `Minus` | `text-[var(--text-muted)]` | — | `已略過（字幕軌語言不符）` |

The `?? SUBTITLE_STATUS.not_searched` fallback stays as the belt-and-braces default for an unknown value.

**The `aria-label`/`title` is the long form; the badge label is the short form.** The icon has no visible text, so the accessible name carries the full explanation — this is where AC #4's "已略過 must not read as broken" is actually solved.

**AC #5 of sub-1-7a rules on the existing `searching` tint** (`--warning` + spin, inconsistent with accent-for-in-progress). If the spec says re-tint → do it here, in this story, with a one-line comment citing the spec. If the spec says leave it → leave it, and do not "tidy" it.

### AC #4 — Tests: all 9 values, both surfaces, and the ordering trap

**Given** Rule 9 co-location and Rule 16 assertion quality, **then**:

`apps/web/src/utils/libraryStatus.spec.ts` (extend — the file exists):

1. Table-driven over **all 9** values: the returned label / `className` / `steadyState` (or `null`).
2. **The ordering regression test** — the one that fails if the branches are placed wrong: an item with `subtitleStatus: 'no_text_source'` **and** a non-empty `subtitleTracks` JSON containing an `und`/image-ish track asserts `無字幕源`, **not** `有字幕`. Repeat for `skipped`. Without these two cases the ordering bug in AC #1 ships silently.
3. `pickPosterBadge` renders the badge for `no_text_source` / `skipped` and renders **nothing** for the three transient values (AC #2).
4. Existing assertions for the 4 original values stay green, unmodified.

`apps/web/src/components/media/EpisodeList.spec.tsx` (extend — the file exists):

5. Each new value renders its icon with the correct `aria-label` — query by `getByRole('status', { name: … })`, per Rule 16 use `toBeInTheDocument()`, never `toBeTruthy()`.
6. `spin` applies `animate-spin` for the three in-flight values and **not** for the terminal ones.
7. The `!episode.hasLocalFile → null` guard still short-circuits before any status lookup.

### AC #5 — Scope fence

- ❌ **No backend.** Zero files under `apps/api/`. The Go enum is sub-1-2's.
- ❌ **No new gallery fixtures, no new visual baselines.** `PosterCardV2`, `LibraryListRowV2`, and `EpisodeList` have **no** entries in `apps/web/src/routes/test/-gallery.fixtures.tsx` today, so there is no baseline to update — and creating three from scratch is Rule 22 backfill work, not this story's. `-linux` baselines cannot be generated on this machine anyway (darwin; CI opens the bootstrap PR). Verification here is vitest + the AC #6 manual comparison against `j2-d.png`.
- ❌ No `.pen` edit, no screenshot regeneration — that was sub-1-7a.
- ❌ No Activity-hub work, no SSE consumption, no `useSubtitleSearch` change — sub-1-6 owns in-flight progress.
- ❌ No library **filter** by the new statuses. `routes/library.tsx:20-22` documents subtitle-status filtering as an unwired follow-up; wiring it is a separate concern. **Do not touch the `typeof search.subtitleStatus === 'string'` guard** — it is Rule 26-safe for all-alphabetic values (the rule's own carve-out) and "fixing" it is the exact anti-pattern Rule 26 warns about.
- ❌ No `types/library.ts` change. `subtitleStatus?: string` is deliberately loose; narrowing it to a 9-value union is a nice-to-have that would ripple into fixtures and mocks — out of scope, and its absence is why AC #4's runtime tests matter.

### AC #6 — Design verification before completion

**Given** the mandatory UX verification gate, **then** before this story is marked done the implementer renders each new state in the running app (or via a scratch fixture) and **compares it side by side with `_bmad-output/screenshots/flow-j-specs/j2-d.png`**, recording the comparison in Completion Notes. A story that says "matches the design" without having opened the screenshot does not pass.

---

## Tasks / Subtasks

- [ ] **Task 1 — Badge derivation (AC #1, #2)**
  - [ ] 1.1 Read sub-1-7a's Completion Notes + `flow-j-specs/j2-d.png` for the ratified labels/tints. If they differ from the proposal above, implement the spec.
  - [ ] 1.2 Extend `deriveSubtitleStatus`: terminal branches **above** the track-inference step, transient values → `null` **without** reaching it; add the why-comment.
  - [ ] 1.3 Confirm `pickPosterBadge`, `deriveFromTracks`, `deriveLifecycleStatus`, `TINT`, `HANT`/`HANS` are untouched (`git diff` the file and check the hunk count).
- [ ] **Task 2 — Episode row icons (AC #3)**
  - [ ] 2.1 Extend `SUBTITLE_STATUS` with the 5 entries + long-form `aria-label`s.
  - [ ] 2.2 Apply (or deliberately skip) the `searching` re-tint per sub-1-7a AC #5, with a comment citing the spec either way.
  - [ ] 2.3 Leave the `// Implements: <screen-section — pending epic-19-8 mapping>` header as-is — it is an accepted Rule 21 form and re-mapping it is out of scope.
- [ ] **Task 3 — Tests + gates (AC #4, #6)**
  - [ ] 3.1 Extend `libraryStatus.spec.ts` — the 9-value table + the two ordering-regression cases + the `pickPosterBadge` cases.
  - [ ] 3.2 Extend `EpisodeList.spec.tsx` — icon/aria per new value, `animate-spin` presence/absence, the `hasLocalFile` short-circuit.
  - [ ] 3.3 `pnpm nx test web` green (no `run_in_background` — it orphans vitest workers).
  - [ ] 3.4 `pnpm lint:all` green from the repo root (eslint + prettier are the relevant steps here).
  - [ ] 3.5 Render each new state and compare against `j2-d.png`; record the comparison in Completion Notes.

---

## Dev Notes

### Where the change actually goes

| File | Change | Rule notes |
|---|---|---|
| `apps/web/src/utils/libraryStatus.ts` | `deriveSubtitleStatus` branches | **`utils/`, not `components/`** → Rule 21's `local/implements-pen-node-id` ESLint rule does not apply (scoped to `apps/web/src/components/**`), and Rule 23 does not apply (no wall-clock read). |
| `apps/web/src/components/media/EpisodeList.tsx` | `SUBTITLE_STATUS` map | Rule 21 applies; the existing header is already a valid accepted form — **no header edit needed**. |
| `apps/web/src/utils/libraryStatus.spec.ts` | extend | Rule 9 co-location; file exists. |
| `apps/web/src/components/media/EpisodeList.spec.tsx` | extend | Rule 9 co-location; file exists. |
| `PosterCardV2.tsx` / `LibraryListRowV2.tsx` | **none** | They call `deriveSubtitleStatus`; centralized derivation is the whole point. |

### Existing vocabulary — reuse, do not redeclare

- Tints: `apps/web/src/utils/libraryStatus.ts:30-37` — `success` / `accent` / `warning` / `error` / `info` / `neutral`, each a `bg-[var(--x-tint)] text-[var(--x)]` pair. **Accent is reserved for in-progress** (Sally ruling 2026-07-05, cited at :87-88).
- Icons: `EpisodeList.tsx:35-40` — `CheckCircle2` / `XCircle` / `Loader2` (with `spin`) / `Minus`, all `lucide-react`.
- Labels in play: 已入庫 / 整理中 / 失敗 / 繁中 / 簡中 / 有字幕 / 缺字幕 — 3–4 CJK chars. Match the register.
- `HANT` / `HANS` sets are exported precisely so every subtitle surface classifies 繁/簡 identically — **do not redeclare them locally** (their own docstring says so).

### Why the branch ordering is the real bug risk

`deriveSubtitleStatus`'s step 2 infers a badge from `subtitleTracks` JSON and returns as soon as it finds *any* track. A file that the pipeline marked `no_text_source` may still carry image-only (PGS/VobSub) tracks in `subtitleTracks`, and a `skipped` file carries a real text track with an `und`/non-English tag. In both cases step 2 happily returns `有字幕` — the pipeline's authoritative verdict loses to a naive track count, and the user sees "has subtitles" on a file that will never get one. AC #4's two ordering-regression tests exist solely to catch this; they fail against the natural "append the new cases at the bottom" implementation.

### Rule compliance for this story

- **Rule 5** — no server state introduced; this reads props already supplied by the library queries.
- **Rule 9** — both specs co-located, both already exist.
- **Rule 16** — `toBeInTheDocument()` for presence, `getByRole('status', { name })` for the icons, exact `toEqual` on the returned `StatusDescriptor`. No `toBeTruthy` for DOM presence.
- **Rule 17** — no user-facing **doc** changes, so no EN/zh-TW doc pair is owed. (UI copy is zh-TW-only by design; the app is not bilingual.)
- **Rule 20** — this story **consumes** sub-1-2's `[@contract-v1]` `SubtitleStatus` 9-value set. Record the ack in Dev Notes at implementation time in the canonical form: `confirmed against [@contract-v1] sub-1-2 AC #2`. It **stamps nothing new**.
- **Rule 21** — only `EpisodeList.tsx` is under `components/**`; its header is already valid.
- **Rule 24** — see Discovery Triage.
- **Rule 26** — the `library.tsx` search-param guard is safe for all-alphabetic values; **do not touch it** (AC #5).
- **Feedback: no background tests** — run vitest in the foreground; `run_in_background` orphans workers.
- **Feedback: run prettier before commit** — subagent edits skip Prettier; `pnpm run format:check` must be clean.

### Time-dependent visual coverage

**N/A — no wall-clock-reading components touched.** `EpisodeList.tsx` is the only `apps/web/src/components/**` file in scope and it reads no `Date.now()` / `new Date()` / `Date.UTC()` / `Date.parse()`; the badge derivation is a pure function of props. Rule 23 does not apply, and no `clockTime` fixture rows are owed. Reference: `project-context.md` Rule 23; audit `_bmad-output/audit/time-bomb-fixtures-2026-05.md`.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 1.7b] · Alexyu ruling 2026-07-27.
- [Source: `sub-1-7a-subtitle-status-badge-design.md`] — **the verification target**; its Completion Notes carry the ratified table.
- [Source: `sub-1-2-pipeline-state-model.md`#AC #2] — the `[@contract-v1]` 9-value set + the `no_text_source` vs `skipped` semantics (ASR-recoverable vs deliberately routed out).
- [Source: `apps/web/src/utils/libraryStatus.ts`:1-15,30-37,72-92,102-137] — the 4-step ladder, tint tokens, the accent ruling, the exception-signal rule.
- [Source: `apps/web/src/components/media/EpisodeList.tsx`:33-60] — the icon map and the `hasLocalFile` guard.
- [Source: `apps/web/src/routes/library.tsx`:20-22,42] — the deliberately-untouched Rule 26-safe search-param guard.
- [Source: `project-context.md`#Rule 16, #Rule 20, #Rule 21, #Rule 23, #Rule 26].

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

_(Record here: the ratified label/tint/icon table actually implemented, and the AC #6 side-by-side comparison against `j2-d.png`.)_

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO**: state `N/A — no out-of-scope work discovered`.
  - Two items are **pre-triaged as deliberate exclusions**, not discoveries, and need no new entry unless you find a reason to change them: (a) no gallery fixtures / visual baselines exist for `PosterCardV2` / `LibraryListRowV2` / `EpisodeList` — Rule 22 backfill, out of scope (AC #5); (b) library filtering by `subtitle_status` remains unwired (`routes/library.tsx:20-22`) — pre-existing, unchanged by this story.
- Reference: `project-context.md` Rule 24.

### File List
