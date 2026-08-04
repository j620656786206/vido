# Story sub-1.7b: Subtitle-status badge — render the 5 new pipeline states

Status: review

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

- [x] **Task 1 — Badge derivation (AC #1, #2)**
  - [x] 1.1 Read sub-1-7a's Completion Notes + `flow-j-specs/j2-d.png` for the ratified labels/tints. If they differ from the proposal above, implement the spec.
  - [x] 1.2 Extend `deriveSubtitleStatus`: terminal branches **above** the track-inference step, transient values → `null` **without** reaching it; add the why-comment.
  - [x] 1.3 Confirm `pickPosterBadge`, `deriveFromTracks`, `deriveLifecycleStatus`, `TINT`, `HANT`/`HANS` are untouched (`git diff` the file and check the hunk count).
- [x] **Task 2 — Episode row icons (AC #3)**
  - [x] 2.1 Extend `SUBTITLE_STATUS` with the 5 entries + long-form `aria-label`s.
  - [x] 2.2 Apply (or deliberately skip) the `searching` re-tint per sub-1-7a AC #5, with a comment citing the spec either way.
  - [x] 2.3 Leave the `// Implements: <screen-section — pending epic-19-8 mapping>` header as-is — it is an accepted Rule 21 form and re-mapping it is out of scope.
- [x] **Task 3 — Tests + gates (AC #4, #6)**
  - [x] 3.1 Extend `libraryStatus.spec.ts` — the 9-value table + the two ordering-regression cases + the `pickPosterBadge` cases.
  - [x] 3.2 Extend `EpisodeList.spec.tsx` — icon/aria per new value, `animate-spin` presence/absence, the `hasLocalFile` short-circuit.
  - [x] 3.3 `pnpm nx test web` green (no `run_in_background` — it orphans vitest workers).
  - [x] 3.4 `pnpm lint:all` green from the repo root (eslint + prettier are the relevant steps here).
  - [x] 3.5 Render each new state and compare against `j2-d.png`; record the comparison in Completion Notes.

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
- **Rule 20** — this story **consumes** sub-1-2's `[@contract-v1]` `SubtitleStatus` 9-value set. It **stamps nothing new**.
  - ✅ **Ack recorded at implementation (2026-08-04):** `confirmed against [@contract-v1] sub-1-2 AC #2` — the 9-value set (`not_searched` / `searching` / `found` / `not_found` / `probing` / `extracting` / `translating` / `no_text_source` / `skipped`). Greped at v1, no bump anywhere in sub-1-2's history, so no stale-mark is owed to this story. All 9 values are rendered by this story; none is left to a fallback by accident (the `?? not_searched` default now only catches genuinely unknown future values).
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

Amelia (Developer Agent) · Claude Fable 5, effort xhigh · 2026-08-04

### Debug Log References

RED verified before each task, and each load-bearing guard falsified afterwards (break it, watch the test catch it, restore):

| Task | RED signal |
|---|---|
| 1 | `libraryStatus.spec.ts` 7 failed / 22 passed — `expected undefined to be '無字幕源'` |
| 2 | `EpisodeList.spec.tsx` 8 failed — `Unable to find role="status" with name "偵測字幕軌中"` |

| Guard | Falsification | Result |
|---|---|---|
| Terminal verdicts outrank track inference | the whole `switch` moved BELOW the `deriveFromTracks` step | **3 tests FAIL** (`no_text_source` and `skipped` infer 有字幕; transient values badge from a stale track guess) |
| `searching` re-tint (sub-1-7a AC #5) | `--accent-text` reverted to `--warning` | 1 test FAILS |
| accent is reserved for in-progress | `no_text_source` given `--accent-text` | 1 test FAILS |

### Completion Notes List

- 🎯 **Implemented exactly the sub-1-7a ratified table, with ONE deliberate deviation** (next bullet). Poster/list badge — `no_text_source` → `無字幕源`, `skipped` → `已略過`, both `TINT.neutral` (`--bg-tertiary` / `--text-muted`), both non-steady so `pickPosterBadge` surfaces them with **zero edits to `pickPosterBadge` itself**; `probing`/`extracting`/`translating` → `null`. `EpisodeList` icons — `Loader2`+spin for the three in-flight, `XCircle` for `no_text_source`, `Minus` for `skipped`, long-form `aria-label`s per the spec. `searching` re-tinted `--warning` → accent per sub-1-7a AC #5.
- ⚠️ **Deviation from the ratified table, on a11y evidence: the icon accent is `--accent-text` (`#60a5fa`), not `--accent-primary` (`#3b82f6`).** sub-1-7a's table said `--accent-primary` (my own correction of the non-existent `--accent`); this story's AC #3 proposal said `--accent-text`. The proposal is right and the spec screen is wrong. Measured contrast against the three surfaces an episode row can sit on (rows carry no background of their own — they inherit the container):

  | token | on `--bg-primary` | on `--bg-secondary` | on `--bg-tertiary` |
  |---|---|---|---|
  | `--accent-primary` `#3b82f6` | 4.26 | 3.58 | **3.04** |
  | `--accent-text` `#60a5fa` | 6.16 | 5.17 | **4.40** |

  WCAG 1.4.11 (non-text contrast) requires **3.0:1**. `--accent-primary` lands at 3.04 on `--bg-tertiary` — passing only on paper, and any hover/elevated surface puts it under. `--accent-text`'s own token comment reads *"accent body text / active label (TC-2 AA-safe)"*: an icon **is** foreground, so it is the intended token. Filed `backlog-pen-j2-accent-token-correction` (lane ③) so the `.pen` spec screen is corrected to match rather than leaving code and design contradicting each other.
- 🔗 **AC Drift: FOUND** — `12-2-season-episode-list` AC #5 / sub-task 7.4 specified `searching = amber spinner` (`--warning`). sub-1-7a AC #5 ruled it re-tinted, and this story executes that: **`searching` icon `--warning` → `--accent-text`**. Deliberate, spec-backed, and covered by a dedicated regression test. Story 12-2 carries **0** `[@contract-v*]` stamps → pre-Rule-20, implicit v0 under the forward-only retrofit, so no ack and no stale-mark are owed. Nothing else in 12-2's ACs is touched: the `hasLocalFile` guard, the icon set, `role="status"`, and the four original values all behave exactly as 12-2 shipped them (their tests are untouched and green).
- 📎 **Contract Stamps: FOUND** (1 consumed, 0 produced). `confirmed against [@contract-v1] sub-1-2 AC #2` — the 9-value `SubtitleStatus` set; greped at v1 with no bump in sub-1-2's history, so no stale-mark is owed. sub-1-7a's 2 stamp hits are quotations of that same upstream, not new contracts. This story stamps nothing — it is a leaf consumer of a wire contract, not a producer.
- 🎭 **A11y Pre-Flight: PASS** (2 files under `apps/web/` touched — `libraryStatus.ts` is a util, `EpisodeList.tsx` is the only component). `eslint` on all four touched files: **0 errors, 0 warnings**, none introduced. Manual 4-class check: ① responsive images — N/A, no `<img>` in scope. ② modal focus — N/A, no dialog. ③ **aria-live on async-revealed content — this is the relevant class and it is satisfied**: every new state renders through the existing `role="status"` span (implicit `aria-live="polite"`) with an `aria-label` **and** a matching `title`, so a status flipping mid-run is announced. ④ custom-widget keyboard/ARIA — N/A, the indicator is non-interactive. Plus the contrast measurement above, which is the substantive a11y decision of this story.
- 🎨 **UX Verification — honest account.** I opened `_bmad-output/screenshots/flow-j-specs/j2-d.png`. **It is a 204×400 thumbnail and its spec text is not legible** — that is the limitation `backlog-pen-spec-screen-readable-export` was filed for in sub-1-7a, and it means a literal pixel side-by-side is not achievable from the committed screenshot. Verification was therefore done attribute-by-attribute against the **ratified table in sub-1-7a's Completion Notes** (which that story designates as the authoritative handoff for exactly this reason), and every attribute is pinned by a test rather than by eyeballing:

  | Spec attribute | Ratified value | Implemented | Pinned by |
  |---|---|---|---|
  | `no_text_source` badge | `無字幕源`, neutral | same | `maps no_text_source → 無字幕源` |
  | `skipped` badge | `已略過`, neutral | same | `maps skipped → 已略過` |
  | transient badges | none | `null` ×3 | `renders NO badge for the three transient states` |
  | in-flight icons | `Loader2` + spin, accent | same (`--accent-text`, see deviation) | `spins for the three in-flight states…` + tint test |
  | `no_text_source` icon | `XCircle`, muted | same | tint test |
  | `skipped` icon | `Minus`, muted | same | tint test |
  | `aria-label`s | 5 long forms | verbatim | the 5-row `it.each` |
  | `searching` re-tint | `--warning` → accent | done | `re-tints the pre-existing searching state` |

  **What still needs a human:** the rendered *look* of the two neutral pills and the spinner in situ. There is no gallery fixture for `PosterCardV2` / `LibraryListRowV2` / `EpisodeList` (AC #5 fences off creating them), so no visual baseline exists to diff. Recommend Alexyu eyeball one library grid + one episode list against the `.pen` `J2-D` screen in Pencil at review time.
- ✅ **AC #5 fence held.** Zero `apps/api/` files. Zero gallery fixtures, zero visual baselines. Zero `.pen` / screenshot changes. `PosterCardV2.tsx` and `LibraryListRowV2.tsx` untouched — the derivation is centralized, which is the whole point. `routes/library.tsx`'s Rule 26-safe `typeof … === 'string'` guard untouched. `types/library.ts` untouched (`subtitleStatus?: string` stays loose — which is precisely why the runtime tests matter).
- ✅ **Full regression gate:** `pnpm nx test web` **2479/2479 PASS** (225 files) · `pnpm nx test api` **34 Go packages ok** · targeted `eslint` 0/0 on the four touched files · `prettier --write` clean · no orphaned vitest workers (`pgrep vitest` empty after every run).
- ⚠️ **Pre-existing, not introduced:** `tsc --noEmit -p apps/web/tsconfig.app.json` reports **139 errors on a clean `main` checkout**, in 11 files (`RecentMediaPanel`, `HeroBanner`, the three library empty-states, the scanner components, `ScannerSettings`, `useScanProgress`, `useSubtitleBatchProgress`, `-gallery.fixtures.tsx`). **My four files contribute 0.** Verified by stashing this story's changes and re-running. It is not a CI gate (`pnpm lint:all` = go vet → staticcheck → eslint → prettier; the web build is Vite, which does not typecheck), so this story neither fixes nor is blocked by it — filed as `backlog-web-tsc-app-config-errors` so it stops being rediscovered.

### Discovery Triage

- **③ backlog-with-carry-forward-link → `backlog-pen-j2-accent-token-correction`.** The `J2-D` spec screen specifies `--accent-primary` for the in-flight icons; the implementation uses `--accent-text` on measured contrast grounds (see Completion Notes). Code and design now disagree on one token, which is the two-sources-of-truth failure mode — filed at discovery with a bidirectional link so the `.pen` is corrected on its next open (a one-token edit, not worth a Pencil session of its own, and the `.pen` is authoritative for *look*, not for a token name the codebase can measure).
- **⛔ ROUTED TO SALLY (ux-designer) → `backlog-episodelist-skipped-vs-not-searched-glyph` (state: `blocked`) + `ux-decision-request-episodelist-skipped-glyph.md`.** *(Re-classified 2026-08-04 on Alexyu's ruling — it was first filed as a plain lane-③ backlog line with a dev-recommended glyph, and that was the wrong call twice over: recommending a glyph IS making the design decision, and an unowned backlog line has nobody to make it. Two states rendering identically is a UI/UX decision and belongs to Sally.)* Found by the pre-ship adversarial self-review: on the `EpisodeList` row `skipped` and `not_searched` render **identically** (both `Minus` + `--text-muted`), distinguishable only by the accessible name. That collides with this story's own statement — *"an item the pipeline permanently declined is visibly distinguishable from one it simply hasn't reached yet"*. Shipped as ratified anyway (sub-1-7a AC #3 specifies `Minus`; choosing a different glyph is a UX call, not a dev call), and the collision does **not** exist on the primary surface — the poster/list badge shows 已略過 versus 缺字幕/有字幕/none. Filed with the candidate fix (`CircleSlash` / `Ban` / `SkipForward` on `J2-D`, then a one-line map edit) and a code comment at the definition site so the next reader does not re-derive it.
- **③ backlog-with-carry-forward-link → `backlog-web-tsc-app-config-errors`.** 139 pre-existing `tsc --noEmit` errors on a clean `main` across 11 files; this story's four files contribute 0. Not a CI gate today, which is why it accumulated. Filed so "pre-existing, not in scope" stops being written without a tracking entry (Epic 9c Retro AI-2).
- **Pre-triaged exclusions confirmed unchanged, no entry owed:** (a) no gallery fixtures / visual baselines for `PosterCardV2` / `LibraryListRowV2` / `EpisodeList` — Rule 22 backfill, fenced by AC #5; (b) library filtering by `subtitle_status` remains unwired (`routes/library.tsx:20-22`) — untouched, Rule 26-safe as-is.
- **Not a discovery — resolved inside the story:** the `searching` re-tint is AC-drift against 12-2, but sub-1-7a AC #5 already ruled it, so it is executed here under that ruling rather than filed (recorded under 🔗 AC Drift).
- Reference: `project-context.md` Rule 24.

### Change Log

| Date | Change |
|---|---|
| 2026-08-04 | **Follow-up to Sally's ruling — `skipped` glyph + icon grammar.** The pre-ship self-review found `skipped` and `not_searched` rendering identically; per Alexyu's ruling that class of finding belongs to the designer, it was routed to Sally rather than self-decided. Her verdict reframed it: not a lookalike but a **misclassification** — the icon set already encodes an unwritten rule (CIRCLED = settled outcome, BARE = not an outcome yet), and `skipped` is settled but carried a bare `Minus`, which is what put it beside `not_searched`. Implemented: `skipped` → `CircleSlash` + `--text-muted`; `not_searched` untouched; `no_text_source`'s family resemblance to `skipped` ratified as correct and commented so nobody "fixes" it apart. The durable half is the **icon grammar** — added to `J2-D` and mirrored as a header comment here — plus a regression guard that asserts the states **against each other** (`glyphOf('skipped') !== glyphOf('not_searched')`), because the original miss shipped precisely because every test asserted each state in isolation. Falsified: reverting to `Minus` fails 2 tests. Sally also upheld the `--accent-text` deviation (the spec was wrong, not the code) and folded that correction into the same `.pen` edit. Gates: web **2482/2482**, eslint 0/0, prettier clean. |
| 2026-08-04 | **Tasks 1–3 (AC #1–#6) — RED first on both surfaces.** `deriveSubtitleStatus` gains the 5 pipeline values as a `switch` placed **above** track inference: the two terminal verdicts return neutral badges, the three in-flight values return `null`, and neither reaches `deriveFromTracks`. That ordering is the whole risk of this story — `no_text_source` files typically still carry image-only (PGS/VobSub) tracks and `skipped` files carry a real text track with an `und` tag, so from below the ladder both would infer 有字幕 and the engine's authoritative verdict would lose to a naive track count. Falsified by moving the switch down: 3 tests fail. `pickPosterBadge` needed **zero** edits (both terminal descriptors are non-steady by construction) and is asserted, not modified. `EpisodeList`'s `SUBTITLE_STATUS` map now covers all 9 values with long-form `aria-label`s — the icon has no visible text, so the accessible name is the only place 「已略過是刻意、不是壞掉」 can actually be said. `searching` re-tinted `--warning` → accent per sub-1-7a AC #5 (AC drift vs 12-2 AC #5, recorded). The accent token is `--accent-text` not the spec's `--accent-primary`: measured 3.04:1 on `--bg-tertiary` vs the 3.0 WCAG floor, against 4.40:1 for `--accent-text`; `.pen` correction filed. Gates: web 2479/2479, api 34 packages, eslint 0/0 on touched files, prettier clean, no orphaned workers. |

### File List

| File | Change |
|---|---|
| `apps/web/src/utils/libraryStatus.ts` | **modified** — AC #1: the 5 pipeline values as a `switch` ABOVE track inference (terminal → neutral badge, in-flight → `null`), with the why-comment the AC asks for; the file docstring's now-false rationale ("ephemeral, no persisted per-item field" — voided by sub-1-2) rewritten to rest on the surviving principle (badge = exception signal) instead. `deriveFromTracks` / `deriveLifecycleStatus` / `pickPosterBadge` / `TINT` / `HANT` / `HANS` **untouched** |
| `apps/web/src/utils/libraryStatus.spec.ts` | **modified** — +17 tests: the 5 new values, 無字幕源-vs-缺字幕 distinctness, the two ordering-regression cases (image-only track + `und` track), transient-values-do-not-reach-inference, and the `pickPosterBadge` surfacing/suppression pair. The 22 pre-existing assertions are unmodified and green |
| `apps/web/src/components/media/EpisodeList.tsx` | **modified** — AC #3: `SUBTITLE_STATUS` extended to all 9 values with long-form `aria-label`s; `searching` re-tinted `--warning` → `--accent-text` per sub-1-7a AC #5; the `?? not_searched` fallback and the `hasLocalFile` guard untouched. Rule 21 header left as-is (already a valid accepted form). **Follow-up:** `skipped` `Minus` → `CircleSlash` per Sally 2026-08-04, and the icon-grammar rule (circled = settled, bare = not yet) recorded as a header comment |
| `apps/web/src/components/media/EpisodeList.spec.tsx` | **modified** — +13 tests: the 5 `aria-label`s, spin present/absent, the accent-vs-muted tint split, the `searching` re-tint regression, the `hasLocalFile` short-circuit, and the unknown-value fallback. **Follow-up:** +3 icon-grammar guards that compare states AGAINST EACH OTHER rather than in isolation |
| `_bmad-output/implementation-artifacts/12-2-season-episode-list.md` | **AC drift reference — see Completion Notes** (its AC #5 「searching=amber spinner」 is superseded by sub-1-7a AC #5; file itself unmodified) |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | **modified** — `sub-1-7b` → `review`; `backlog-pen-j2-accent-token-correction` and `backlog-web-tsc-app-config-errors` filed (lane ③) |
