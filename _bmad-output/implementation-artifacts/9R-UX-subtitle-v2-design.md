# Story 9R-UX — Subtitle management v2 design (generation-centric, `.pen` flow-f-subtitle-v2)

Status: review

**Epic:** epic-9R-subtitle-route-c (Track 0 — UX design) · **doubles as** Epic 6 `ux3-subtitle-v2` per-flow-recipe step 1 (PH3-M5)
**Owner:** ux-designer (Sally / Pencil MCP) · **Type:** design (delivers `.pen` + screenshots, not code) · **Priority:** P0 — **gates Epic 6 (ux3-subtitle-v2) frontend** · **Feasibility:** PROVEN (Route C POC validated the flow end-to-end)

## Story

As the design system,
I want the subtitle-management surface redrawn to v2 as **generation-centric** — `生成字幕`
(Route C) as the primary action, the dead multi-source fetch UI retired, generation pipeline
states + a **glossary-management surface** made visible — in a new `flow-f-subtitle-v2` folder,
So that dev builds Epic 6 (ux3-subtitle-v2) against a spec that matches the ADR-ratified
reality: **Route A fetch is non-viable for 繁中** (Assrt token unobtainable, Zimuku WAF-dead,
OpenSubtitles 繁中-thin) and **Route C generation is the sole 繁中 path**, with the per-show
glossary (9R-6/7) as the keystone differentiator.

## Context — why the existing design稿 is DEAD, not just stale

**`flow-f-subtitle` f1–f3 shows a fetch UI for sources that no longer work.** The frames show
`[✓ Assrt] [✓ Zimuku] [✓ OpenSub]` source chips and source-scored result rows (Assrt 92 /
Zimuku 78) — the 2026-06-16 live POC + ADR (`adr-subtitle-route-c-generation.md`, ACCEPTED)
de-scoped Route A for 繁中. Drawing dev toward that UI would ship a feature whose backends are
a WAF wall and an unobtainable token. `flow-g-ai-subtitle` g1–g3 (AI correction / transcription)
predates Design Language v2 AND the Route C promotion — generation is no longer an optional
enhancement page, it IS the subtitle feature.

**Three things this story must keep straight:**

1. **Generation-first IA (ADR Decision 1).** `生成字幕` is the primary action everywhere.
   Fetch survives only as a **dormant best-effort affordance** (OpenSubtitles works but 繁中 is
   thin) — a quiet secondary action, **no source chips**, never presented as the reliable path.
   **Zimuku must not appear anywhere** (9R-14 removes the provider; its AC #2 says "UI no
   longer lists Zimuku").
2. **The glossary is the keystone (ADR Decision 3).** Per-show term ↔ zh-TW mapping
   (隱形戰士/隱形特務 drift is the canonical failure) is the differentiator no OSS tool offers.
   This story makes 9R-6/7 visible: a review/confirm surface, not a hidden table.
3. **Two entry surfaces, one feature (Epic 6 revision).** The detail `管理字幕` CTA (already
   live in `LocalDetailV2.tsx:126`, v2 shell) opens the per-item surface; the **Activity batch
   surface drives library-wide *generation*, not fetch**. Soft-depends on Activity (shipped,
   flow-k) — reuse `ActivityRow-v2`, do not fork a second task-row idiom.

## Design scope — what to draw (in `ux-design.pen`, new folder `flow-f-subtitle-v2`)

Draw to v2 via Pencil MCP per `01-design-language-v2.md`: token-only color, Noto Sans TC for
all CJK, **JetBrains Mono for ALL numerics** (%, counts, timestamps, cost figures — mixed
strings like `107 分` split into Mono number + Noto unit nodes, gap 3-4, cnt/cntUnit precedent),
44px touch floor, Base UI primitives for dialog/menu, four-state standard §7, AA `*-text`
variants (TC-2). New canvas block per the merged-block convention (desktop row above, mobile
row below, captions above frames, ~45px clearance).

Frames to land (codes folder-scoped; suggested — designer may re-letter, keep `-v2` suffix):

- **`F1-D-v2` / `F1-M-v2` — Detail `管理字幕` surface (per-item).** Opened from the detail CTA.
  Shows: current subtitle state (existing tracks: language `繁中`/`簡中`/`英文`, source
  generated/fetched/local, file name); **primary `生成字幕` action** (Route C); the 簡轉繁
  conversion affordance where a 簡中 track exists (CN-content policy: mainland titles keep
  simplified — surface the policy, don't silently convert); a **dormant** secondary
  `搜尋線上字幕`（best-effort，成功率低）affordance with NO source chips; an entry to the
  glossary surface (`名詞對照表`). States: 已有繁中 (steady) / 缺字幕 (generation CTA
  prominent) / 生成中 (progress inline, see F2).
- **`F2-D-v2` / `F2-M-v2` — Generation progress (pipeline states).** The Route C pipeline as a
  stage progression: `提取音訊 → 轉錄中 → 翻譯中 → 簡轉繁 → 完成` (per ADR: extract →
  transcribe → LLM-translate → OpenCC → place), live % + current stage, **real-time push (SSE),
  not polling** (ux3-4-2b gated-broadcaster precedent; lazy visibility-gated FE consumption).
  Include the AI-correction state (AI校正中) where the terminology pass runs. Show cost/quota
  visibility (9R-11: token usage / per-run budget) at least as a summary line — Mono numerics.
- **`F3-D-v2` / `F3-M-v2` — Glossary management (`名詞對照表`, per-show).** Visualizes 9R-6/7:
  rows of `term_src ↔ term_zh` with a **source badge** (字幕/中繼資料/手動 = subtitle |
  metadata | manual), a **confirmed** state (未確認 rows visually distinct, confirm/edit
  affordance), add/edit/delete, empty state (`尚無詞彙 — 生成字幕時自動累積`). This is a
  review/confirm workflow, not a raw table dump.
- **`F4-D-v2` / `F4-M-v2` — Batch generation (library-wide).** The Activity-side surface: batch
  start (scope: 缺字幕 items), per-item progress list with pipeline stage + overall N/M (Mono),
  cancel affordance, budget-ceiling state (9R-11: 已達本次預算上限 — partial completion is a
  normal outcome, not an error). Entry + summary row consistent with shipped `ActivityRow-v2`
  (flow-k) — the Activity hub links in; do NOT duplicate the hub here (D4-1-style boundary,
  mirrors Downloads).
- **Four-state (N4 — DL-v2 §7; design ALL of them or it doesn't ship), per surface:** default /
  loading / empty (distinct from failure) / **fail-soft error**. The two error states that
  matter most here: **`TRANSCRIPTION_DISABLED`** (FFmpeg or API key missing →
  `字幕生成尚未設定` + `前往設定`, surface degrades, page never hard-fails) and **generation
  failure mid-pipeline** (stage-labeled error + `重試` — retry/backoff exists per 9R-4/11).

Reusable components to spec (and **register in the Component Library frame `sJzat`** — v2
convention: cell in the matching section, ref instance + Noto 12 `$text-muted` caption below):

- **`Component/GenerationProgress-v2`** — the pipeline stage stepper + % + stage label
  (variants: in-progress per stage / failed-at-stage / done). Reused across F1/F2/F4.
- **`Component/GlossaryRow-v2`** — term pair + source badge + confirmed state + row actions.

Reuse, don't invent: `ActivityRow-v2` (flow-k) for Activity-side rows; DL-v2 §2.5 status→token
map for all generation states (the lifecycle moat 想要→下載中→整理中→已入庫 already uses it —
subtitle states join the same system, no bespoke palette); v2 shell (detail surface rides
`LocalDetailV2`'s existing dialog entry — this is a surface redraw, not a new destination).

## Key design decisions (resolved with rationale — flag the capability gaps)

1. **Generation-first, fetch-dormant (ADR Decision 1).** `生成字幕` primary; `搜尋線上字幕` is a
   quiet best-effort secondary with no source chips and honest expectation-setting; **Zimuku
   appears nowhere** (9R-14). Do NOT draw a source-picker, score-breakdown rows, or any
   fetch-first flow — that's the superseded f1–f3 world.
2. **⚠️ Capability-honor (Rule 24) — what is REAL today vs target-state.** Verified 2026-07-05:
   - **Live BE:** fetch engine `POST /api/v1/subtitles/{search,download,preview,batch}` +
     `GET /subtitles/batch/status` + `POST /subtitles/batch/cancel`
     (`subtitle_handler.go:61-71`) — all **fetch**-shaped; single-movie transcription
     `POST /api/v1/movies/:id/transcribe` (202 + job ID, gated on `IsAvailable()` →
     `TRANSCRIPTION_DISABLED` when FFmpeg/API key missing, `transcription_handler.go:44-46`);
     translation service wired behind it (Story 9.2b); SSE `subtitle_progress` +
     `subtitle_batch_progress` events exist.
   - **NET-NEW BE (draw as target state, never as live):** the **one-flow Route C pipeline +
     stage SSE** (9R-10, depends 9R-1..4/7/11 — the existing transcribe path has 4 PROVEN
     production bugs, so even "live" transcription is not drawable-as-reliable until 9R-1..4
     land); **glossary storage** (9R-6/7 = migration + repo + service only — see Discovery
     Triage: glossary has **no HTTP surface in any story**, newly filed `9R-15`); **batch
     generation** (today's batch endpoints run the FETCH pipeline; library-wide generation
     needs 9R-10 + 9R-11); **series/episode generation trigger** (transcribe endpoint is
     movies-only — absorbed into 9R-10's trigger scope, comment updated in sprint-status).
   - Consequence: **the FE build story is HARD-BLOCKED** on 9R-6/7/10/11/15 (+ bug fixes
     9R-1..4). Design-ahead is correct (per-flow recipe step 1; ux3-4-1 precedent) — but the
     design must record these gaps on-canvas as spec notes, exactly like ux3-4-1 did for
     download actions/SSE.
3. **Glossary confirm-workflow, not settings-table.** The value is the review loop (auto-mined
   terms arrive `unconfirmed` from subtitle/metadata sources; the user confirms/corrects; the
   next run honors it — 9R-7 AC #2 do-not-retranslate). The design should make confirm cheap
   (row-level, batchable) and show provenance (source badge).
4. **Old frames stay untouched, marked superseded.** f1–f3 (flow-f) and g1–g3 (flow-g) are
   historical record — do NOT mutate or reuse them (mirrors ux3-4-1's rule for stale flow-d);
   mark them superseded (caption/annotation per the standalone-spec-screen convention), pointing
   at `flow-f-subtitle-v2`. flow-g's dedicated redraw beyond what this story covers belongs to
   Epic 7 (ux3-ai-subtitle) — this story's F2 generation-progress spec is the shared foundation.
5. **Two entries, one hand-off (Epic 6 revision).** Detail `管理字幕` = per-item; Activity =
   batch. The Activity hub summary row (shipped, ux3-2-x) links to the batch surface — keep the
   hand-off visually consistent, no second competing entry point (mirrors Downloads D4-1).
6. **Lifecycle consistency (N1).** 缺字幕/繁中 poster-badge states (ux3-0-1/0-2) and the
   generation states here MUST read as one system via DL-v2 §2.5 status→token. A library item
   whose badge says 缺字幕 should visually connect to the 生成字幕 CTA it leads to.
7. **CN-content policy surfaced (project-context §9b).** Mainland titles keep simplified subs
   (dialogue matches audio); conversion is user-overridable per-item. The 管理字幕 surface shows
   the policy state rather than hiding it (a 簡中 track on CN content is CORRECT, not a defect).

## Acceptance Criteria

1. **Given** the v2 Design Language + shipped v2 shell, **when** subtitle management is drawn,
   **then** all frames land in a **new** `flow-f-subtitle-v2` folder (merged desktop+mobile
   block, captions above frames); the stale `f1-f3` (flow-f) + `g1-g3` (flow-g) are **left
   untouched** and **marked superseded** (annotation pointing at the v2 folder).
2. **Given** ADR Decision 1, **then** `生成字幕` is the **primary action** on the detail
   `管理字幕` surface; there are **no fetch-source chips** anywhere; fetch appears only as a
   dormant best-effort secondary affordance; **Zimuku appears nowhere** (9R-14).
3. **Given** the Route C pipeline, **then** generation states are drawn as a stage progression
   (`提取音訊/轉錄中/翻譯中/簡轉繁/AI校正/完成` + failed-at-stage + `重試`) with live progress
   specified as **SSE push, not polling**, and cost/quota visibility (9R-11) present; spec'd
   once as `Component/GenerationProgress-v2` and reused across detail + batch surfaces.
4. **Given** the glossary keystone (9R-6/7), **then** a per-show glossary surface is drawn:
   `term_src ↔ term_zh` rows with source badge (字幕/中繼資料/手動), unconfirmed-vs-confirmed
   states, confirm/edit/add/delete, and an empty state — spec'd as `Component/GlossaryRow-v2`.
5. **Given** the Epic 6 revision, **then** the **Activity batch surface drives library-wide
   generation** (scope, per-item stage progress, budget-ceiling partial-completion state,
   cancel), consistent with shipped `ActivityRow-v2`, entered via the Activity hub row with no
   second competing entry.
6. **Given** N4, **then** all four states are drawn per surface — default / loading / empty
   (distinct-from-failure) / fail-soft error — including **`TRANSCRIPTION_DISABLED` →
   `字幕生成尚未設定` + `前往設定`** and mid-pipeline failure.
7. **Given** Rule 24 capability-honor, **then** the design records the BE gaps **on-canvas as
   spec notes** — (a) one-flow pipeline + stage SSE = 9R-10, (b) glossary HTTP API = 9R-15
   (net-new, filed by this story), (c) batch generation = 9R-10+11, (d) series trigger =
   9R-10 scope — and **nothing is drawn as already-live**; the CN-content 簡繁 policy state is
   surfaced per §9b.
8. **Given** v2 enforcement (zero-tolerance, 2026-07-04 ruling), **then** color is token-only,
   all CJK is Noto Sans TC, **all numerics JetBrains Mono** (mixed strings split into
   number+unit nodes), AA `*-text` variants for colored body text, 44px touch targets on
   mobile, and **both new reusable components are registered in the Component Library frame
   (`sJzat`)** per the v2 cell convention.
9. **Given** the UX screenshots workflow (CLAUDE.md), **then** `scripts/export-pen-screenshots.py`
   `SCREENS` dict is extended with every new `flow-f-subtitle-v2` node ID → code, screenshots
   regenerate, and **only genuinely-changed PNGs** are committed alongside the `.pen` (regen is
   non-deterministic).

## Tasks / Subtasks (designer)

- [ ] (AC #1) Pencil MCP: `get_editor_state(include_schema:true)` + `get_guidelines`; locate the
      v2 shell/atoms + `ActivityRow-v2` + Component Library (`sJzat`); create the
      `flow-f-subtitle-v2` block per the merged-block convention.
- [ ] (AC #7) Re-verify the capability audit before drawing (routes in `subtitle_handler.go` /
      `transcription_handler.go`; SSE event set) — then draw generation/glossary/batch as target
      state with on-canvas BE-gap spec notes (9R-10/11/15).
- [ ] (AC #2, #3) Draw `F1-D-v2`/`F1-M-v2` (管理字幕: current tracks, 生成字幕 primary, 簡轉繁
      + CN-policy state, dormant fetch secondary, glossary entry) and `F2-D-v2`/`F2-M-v2`
      (pipeline stage progression + SSE note + cost line). Spec `Component/GenerationProgress-v2`.
- [ ] (AC #4) Draw `F3-D-v2`/`F3-M-v2` glossary surface + empty state. Spec
      `Component/GlossaryRow-v2`.
- [ ] (AC #5) Draw `F4-D-v2`/`F4-M-v2` batch generation (scope, progress list, budget-ceiling
      partial completion, cancel) + verify the Activity-hub hand-off reads consistently.
- [ ] (AC #6) Draw the four-state set per surface, incl. `TRANSCRIPTION_DISABLED` fail-soft and
      failed-at-stage + 重試.
- [ ] (AC #1) Mark f1-f3 + g1-g3 superseded (annotation → `flow-f-subtitle-v2`); zero mutation.
- [ ] (AC #8) Token/font lint pass on ALL new-frame content (including anything copied from
      existing flows — zero tolerance); split mixed number+unit strings; register both new
      components in `sJzat`.
- [ ] (AC #9) Update `SCREENS` dict; `python3 scripts/export-pen-screenshots.py`; commit `.pen`
      + only-changed PNGs together (`feat(9R-UX): subtitle management v2 design — generation-centric (.pen flow-f-subtitle-v2)`).

## Dev Notes

- **ADR (source of truth):** `_bmad-output/planning-artifacts/architecture/adr-subtitle-route-c-generation.md`
  — Decisions 1 (Route C sole 繁中 path, fetch dormant, Zimuku removed), 3 (glossary keystone),
  4 (cloud translation, Whisper cloud-default/local-opt-in — S2-gated, do NOT draw an engine
  picker as live), 5 (4 production bugs + VAD), 6 (metadata localization = same infra, S1-gated
  — do NOT draw .nfo localization UI in this story).
- **Story breakdown:** `_bmad-output/implementation-artifacts/subtitle-route-c-stories-2026-06.md`
  — Track 0 (this story's 4 core ACs, expanded here), 9R-6/7 (glossary shape: `term_src`,
  `term_zh`, `language`, `source` subtitle|metadata|manual, `confirmed`), 9R-10 (pipeline +
  SSE), 9R-11 (cost/quota), 9R-14 (Zimuku gone).
- **Epic 6 framing:** `_bmad-output/planning-artifacts/epics.md` §"Epic 6: ux3-subtitle-v2"
  (2026-06-16 REVISION — generation status + glossary management; fetch UI dormant; f1–f3
  superseded, "redrawn generation-centric by story 9R-UX before this epic's frontend").
- **Design Language v2:** `_bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md`
  — tokens §2 (status→token §2.5), type §3 (TY-1/TY-2, Mono numerics), atoms §5.1, shell §6,
  four-state §7, a11y §8.
- **Phase-3 map:** `…/03-phase3-destination-epic-map.md` §3 (flow F → Detail 管理字幕 + 活動
  batch; "feature surface, not a destination"), §6 (per-flow recipe step 1 = design-first).
- **Backend capability anchors (audit 2026-07-05):**
  `apps/api/internal/handlers/subtitle_handler.go:61-71` (6 fetch-shaped routes — the ONLY
  subtitle HTTP surface); `apps/api/internal/handlers/transcription_handler.go:44-46`
  (`POST /movies/:id/transcribe`, movies-only, 202+jobID, `TRANSCRIPTION_DISABLED` gate);
  SSE hub event set (project-context §8): `subtitle_progress`, `subtitle_batch_progress` exist;
  NO generation-stage events, NO glossary routes, NO batch-generation route.
- **FE anchors:** `apps/web/src/components/media/LocalDetailV2.tsx:126` (管理字幕 CTA — the v2
  entry already exists; this story redesigns what it opens);
  `apps/web/src/components/subtitle/SubtitleSearchDialog.tsx` + `BatchSubtitleDialog.tsx`
  (the fetch-era dialogs this design replaces; BatchSubtitleDialog wired in `routes/library.tsx`);
  legacy `MediaDetailPanel.tsx:283` (搜尋字幕 — legacy shell, untouched by strangler rule).
- **Precedents:** `ux3-4-1-downloads-design.md` (the format + capability-audit pattern this
  story mirrors; its v2.1 outcome: Pencil inline-agent build + Sally adversarial review,
  mobile sheet-first actions); 13-0 requests design (flow-l; spawned the 2026-07-04
  zero-tolerance font ruling + Component Library registration convention);
  `ux3-2-1-activity-design.md` (ActivityRow-v2, the batch-surface consistency anchor).
- **SSE pattern to spec against:** ux3-4-2b gated broadcaster (`ClientCount()==0 ⇒ skip`) +
  visibility-gated lazy `EventSource` (downloads retro takeaway #1) — Epic 6/9R FE will reuse
  this shape; the design notes "real-time push" accordingly.
- **Memory:** _feedback-design-system-conformance-pen_ (font zero-tolerance incl. copied
  content; sJzat registration); _project-pen-flow-layout-convention_ (merged block, captions
  above, ~45px clearance); _feedback-pencil-label-overlap_; _feedback-pencil-spec-standalone-screen_
  (BE-gap spec notes get their own annotation space, not crammed into mockups);
  _pencil-mcp-edits-need-manual-save_ (Cmd+S before screenshot regen).
- **Web research:** N/A — design-only story; no external library/API version surface. (Pencil
  MCP + internal conventions only.)

### Project Structure Notes

- Design-only story: edits `ux-design.pen` + `_bmad-output/screenshots/flow-f-subtitle-v2/` +
  `scripts/export-pen-screenshots.py` (`SCREENS`). No app code → cross-stack split check N/A
  (0 backend tasks / 0 frontend tasks).
- Branch off `main` (`feat/9R-UX-subtitle-v2-design` suggested); gh account `j620656786206`;
  screenshots regen requires Pencil.app running.
- **Sequencing:** design-ahead is explicitly allowed (per-flow recipe step 1; ux3-4-1
  precedent). Downstream FE (Epic 6) must NOT start until 9R-6/7/10/11/15 (+ 9R-1..4 fixes)
  ship — the design records these as on-canvas BE-gap notes.

### Time-dependent visual coverage

- N/A — design-only story; adds/modifies no `apps/web/src/components/**/*.{ts,tsx}`. (Rule 23
  re-evaluates at the Epic 6 FE story — generation timestamps/ETAs there are server-supplied,
  but the FE story must re-check any wall-clock reads it introduces.)

### References

- [Source: _bmad-output/planning-artifacts/architecture/adr-subtitle-route-c-generation.md#Decisions-1-6]
- [Source: _bmad-output/implementation-artifacts/subtitle-route-c-stories-2026-06.md#Track-0-9R-UX]
- [Source: _bmad-output/planning-artifacts/epics.md#Epic-6-ux3-subtitle-v2 (2026-06-16 REVISION)]
- [Source: _bmad-output/planning-artifacts/ux-redesign/01-design-language-v2.md#§2-§8]
- [Source: _bmad-output/planning-artifacts/ux-redesign/03-phase3-destination-epic-map.md#§3,§6]
- [Source: apps/api/internal/handlers/subtitle_handler.go#RegisterRoutes] — capability audit (fetch-only routes)
- [Source: apps/api/internal/handlers/transcription_handler.go#RegisterRoutes] — movies-only transcribe, TRANSCRIPTION_DISABLED gate
- [Source: project-context.md#§8-SSE-Hub,§9b-Subtitle-Engine] — SSE event set; CN-content 簡繁 policy
- [Source: _bmad-output/implementation-artifacts/epic-ux3-downloads-v2-retro-2026-07-03.md#Key-Takeaways] — HTTP-surface capability check; gated-SSE shape

## Dev Agent Record

### Agent Model Used

claude-fable-5 (SM Bob, create-story yolo)

### Debug Log References

### Completion Notes List

- Ultimate context engine analysis completed — comprehensive designer guide created
  (create-story 2026-07-05: ADR + 9R breakdown + Epic 6 revision + BE/FE capability audit +
  downloads-v2 design/retro precedents baked in).
- Design DONE 2026-07-05 — drawn by Pencil In-App AI agent (yolo) from
  `9R-UX-subtitle-v2-design-prompt.md`, reviewed + corrected by Sally via Pencil MCP.
- **Frames landed (15, block at x≈31590 y≈-5921):** `F1-D-v2` r1EY9 · `F1-M-v2` JkdfH ·
  `F2-D-v2` S9Rbrq · `F3-D-v2` JbXai · `F3-M-v2` k8sJl4 · `F4-D-v2` U8rRtv · `F5-D-v2` f6ZxY ·
  `F6-D-v2` dlfMR · `F6-M-v2` buepS · `F7-D-v2` A85GFD · `F8-D-v2` i9Nun1 · `F8-M-v2` H717g ·
  `F9-D-v2` JMqPg · `F10-D-v2` olDlj · `F11-D-v2` l8FsB (extra, see deviation). Components:
  `Component/GenerationProgress-v2` XkGvG (registered in new sJzat `progress-v2` row luza9),
  `Component/GlossaryRow-v2` nDSEd (registered in `content-cards-v2` row, cell Fx24g).
  Annotations: BE-gaps note CJVC5; superseded notes a2ncs (flow-f) + jRbiH (flow-g); old
  f1-f3/g1-g3 frames untouched.
- **Sally review fixes (2):** deleted F8-M's hidden duplicate sheet stack (agent drew the
  mobile sheet twice; visible render unchanged); added exploration annotation rhhQ0 to F11.
- **Deviation — F11-D-v2 (variant B, full-page generation workspace):** inline agent's
  unprompted extra frame. Kept + annotated 「非 story 規格，dev 以 F1–F10 dialog 流為準」;
  exported as `f11-d-v2.png`. **Pending Alexyu ruling** — adopt into spec (possibly for Epic 7
  ux3-ai-subtitle) or drop.
- Review verdict vs AC #1–#9: PASS. Zero Zimuku/source-chip leakage; number+unit strings
  split Mono+Noto throughout; badge text uses base `$info`/`$warning` on tints (computed AA
  5.4:1/6.1:1 on dark — conforms, matches existing v2 component convention; no `-text`
  variants exist for info/warning in the token set).
- Screenshots: 15 new PNGs in `flow-f-subtitle-v2/` + genuinely-changed
  `design-system/component-library.png` (sJzat gained the two registration cells); all other
  regenerated PNGs reverted as non-deterministic regen noise.

### Discovery Triage

- **YES — out-of-scope work discovered during story creation, triaged:**
  - **③ backlog-with-carry-forward-link — glossary has NO HTTP surface in any planned story.**
    9R-6 covers migration + repository CRUD; 9R-7 covers the service layer. No story adds
    glossary HTTP endpoints, so the glossary-management UI this story draws (AC #4) would have
    nothing to call (exactly the retro-ux3-4 "verify the HTTP surface, not the Go client"
    pattern). **Filed `9R-15-glossary-http-api` (backlog) in sprint-status.yaml at discovery
    time (2026-07-05)** — REST CRUD for per-media glossary terms (list/add/edit/confirm/delete),
    depends 9R-6/7. Gates Epic 6 FE glossary section; does NOT gate this design.
  - **① expand-scope-in-place (on 9R-10, not this story) — series/episode generation trigger.**
    `POST /movies/:id/transcribe` is movies-only; the detail 管理字幕 surface is drawn for both
    movies and series. Absorbed into 9R-10's trigger-conditions scope (its AC #2 defines
    triggers); 9R-10's sprint-status comment annotated 2026-07-05 to make the series trigger
    explicit rather than implied.
- Reference: `project-context.md` Rule 24; origin: this story's capability audit (2026-07-05).

### File List

- `_bmad-output/implementation-artifacts/9R-UX-subtitle-v2-design.md` (this story)
- `_bmad-output/implementation-artifacts/9R-UX-subtitle-v2-design-prompt.md` (inline-agent prompt)
- `_bmad-output/implementation-artifacts/sprint-status.yaml` (9R-UX → review; +9R-15; 9R-10
  annotation; ux3-subtitle-v2 epic note)
- `ux-design.pen` (flow-f-subtitle-v2 block: 15 frames + 2 components + annotations; sJzat cells)
- `scripts/export-pen-screenshots.py` (SCREENS +15)
- `_bmad-output/screenshots/flow-f-subtitle-v2/` (15 new PNGs)
- `_bmad-output/screenshots/design-system/component-library.png` (genuine change: sJzat cells)

## Change Log

| Date       | Change                                                                                                                                                                                                                                                                                     |
| ---------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 2026-07-05 | Story authored (SM create-story, yolo) — expands the 9R breakdown Track-0 mini-spec into the full design story; doubles as Epic 6 (ux3-subtitle-v2) per-flow-recipe step 1. Capability audit baked in (fetch-only subtitle routes; movies-only transcribe; no glossary HTTP surface → filed 9R-15; no generation SSE). Status: ready-for-dev (design not started). |
| 2026-07-05 | Design DONE — Pencil inline-agent (yolo) + Sally MCP review. 15 frames + 2 components (both sJzat-registered) + BE-gaps/superseded annotations; review fixes: F8-M duplicate-sheet deletion, F11 exploration note. F11 variant B = unprompted extra, kept + flagged, pending Alexyu ruling. SCREENS +15; screenshots regenerated (new folder + component-library.png only). Status → review. |
