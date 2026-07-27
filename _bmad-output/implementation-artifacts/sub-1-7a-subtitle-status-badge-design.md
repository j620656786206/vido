# Story sub-1.7a: Subtitle-status badge design spec + Epic 2 `.pen` copy unblock

Status: ready-for-dev

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🟢 LOW** · **UX / DESIGN-ONLY** (Sally — `ux-designer`, **not** dev)
**Origin:** Alexyu ruling 2026-07-27 — *"前端 badge 讓 M1 出貨時就帶 badge"*. Promotes the lane-③ entry `backlog-subtitle-status-fe-rendering` (filed by sub-1-2) into M1 scope.
**Scope expanded 2026-07-27 (Alexyu, party-mode):** the **three open `.pen` copy revisions that block Epic 2** are batched into this same Pencil session — see **AC #7**. Rationale: one `.pen` edit, one screenshot regeneration, one commit. Splitting them means paying the non-deterministic re-render risk twice for changes to the same file.
**Depends on:** sub-1-2's `[@contract-v1]` `SubtitleStatus` 9-value set (the *contract*, not its merge — design can start immediately).
**Blocks:** sub-1-7b (frontend implementation verifies against this spec's screenshot) **and — via AC #7 — `sub-2-1-key-config-page` / `epic-subtitle-pipeline-m1-5` (Epic 2), which are blocked on exactly those three revisions.**
**Requires:** **Pencil.app running** (`ux-design.pen` is read/written via Pencil MCP only — never `Read`/`Grep`).

---

## Story

As Sally (UX),
I want a standalone spec screen defining the badge label, tint, and icon for the 5 new `subtitle_status` values,
so that sub-1-7b has something concrete to verify against instead of a developer guessing zh-TW copy and colour tokens.

---

## The design question this story exists to answer

`apps/web/src/utils/libraryStatus.ts:10-12` currently states the governing principle:

> *"The transient process states (簡轉繁 / AI 校正中) are ephemeral (subtitle-engine SSE, **no persisted per-item field**) → surfaced by the Activity hub, NOT this badge"*

**sub-1-2 voids that rationale.** `probing` / `extracting` / `translating` are now *persisted* per-item column values. The stated reason for keeping process states off the badge no longer holds — but the underlying principle (the poster badge is an **EXCEPTION signal**, ux3-0-2; steady and noisy states are suppressed to avoid always-on grid noise) may still hold.

Someone has to rule on that. This story is that ruling, recorded where the implementation can verify against it.

---

## Acceptance Criteria

### AC #1 — A standalone spec screen exists in `flow-j-specs`

**Given** the Pencil design-decision convention (spec/annotation content gets its **own standalone screen**, never crammed into an existing mockup — bugfix-10-6 ruling), **when** this story completes, **then** `ux-design.pen` carries a new desktop spec screen rendering to `_bmad-output/screenshots/flow-j-specs/j2-d.png`, sibling to the existing `j1-d.png` (PosterCard info-density).

The `SCREENS` dict in `scripts/export-pen-screenshots.py` is extended with the new node id → `("flow-j-specs", "j2-d")`.

### AC #2 — The badge-vs-Activity-hub ruling is stated explicitly on the screen

**Given** the tension above, **then** the spec screen states the ruling in one sentence, with its rationale, so sub-1-7b and every future reader inherit the decision rather than re-litigating it.

**Recommended ruling (Bob's proposal — Sally rules; this is a starting position, not a constraint):**

> **Transient states stay off the poster/list badge; terminal states earn one.** The poster badge is an EXCEPTION signal — an in-flight `extracting`/`translating` is not an exception, it is normal progress, and belongs to the Activity hub + SSE surface (sub-1-6). A terminal `no_text_source` / `skipped` **is** an exception: it tells the user this file will never get a subtitle automatically, which is exactly the "something needs your attention" the badge exists for. The **detail/episode-row icon** is a different surface with a different budget — it already spins for `searching`, so in-progress icons there violate no principle.

### AC #3 — Per-state design tokens are specified for both surfaces

**Given** two distinct rendering surfaces, **then** the screen specifies, for each of the 5 new values, the poster/list badge treatment **and** the `EpisodeList` row-icon treatment.

**Proposal to ratify or overrule:**

| `subtitle_status` | Poster / list badge | Tint | `EpisodeList` icon | Icon colour |
|---|---|---|---|---|
| `probing` | **none** (Activity hub) | — | `Loader2` + spin | `--accent` |
| `extracting` | **none** (Activity hub) | — | `Loader2` + spin | `--accent` |
| `translating` | **none** (Activity hub) | — | `Loader2` + spin | `--accent` |
| `no_text_source` | **`無字幕源`** | `neutral` | `XCircle` | `--text-muted` |
| `skipped` | **`已略過`** | `neutral` | `Minus` | `--text-muted` |

Constraints the proposal is already built to respect — **carry them into whatever you rule**:

- **Accent is reserved for in-progress states** (your own gate ruling, 2026-07-05, cited in `libraryStatus.ts:87-88`). Hence accent for the three spinners, never for terminal states.
- **Badge labels run 3–4 CJK characters** in the existing set (已入庫 / 整理中 / 失敗 / 繁中 / 簡中 / 有字幕 / 缺字幕). `無字幕源` and `已略過` match that register; a longer literal like `無可用字幕來源` will not fit the pill.
- Tint tokens are the existing six in `libraryStatus.ts:30-37` — **no new colour token**.

### AC #4 — Two copy collisions are resolved on the screen

**Given** the existing badge vocabulary, **then** the spec resolves both of these explicitly rather than leaving the implementer to guess:

1. **`無字幕源` vs the existing `缺字幕`.** Both mean "no subtitle", but the recovery differs: `缺字幕` = we searched online and found nothing (re-search may help); `無字幕源` = this file has no text track to extract (only P2 ASR can help). Keep them distinct, merge them, or reword — state which and why.
2. **`已略過` needs to read as deliberate, not broken.** `skipped` means the pipeline correctly declined a track (an `und` or non-English tag — P0: `und` is **never** treated as English). If the label reads like a failure, users will file bugs against correct behaviour. A tooltip/`title` string is in scope for the spec.

### AC #5 — One pre-existing inconsistency is flagged (fix or accept, on the screen)

**Given** `EpisodeList.tsx:38` renders the existing `searching` state with `--warning` + spin while the badge system reserves **accent** for in-progress, **then** the spec states whether `searching` is re-tinted to accent for consistency or deliberately left as-is.

**Do not let sub-1-7b change it silently either way** — if the spec says re-tint, it is in 7b's scope with a stated reason; if the spec says leave it, 7b leaves it.

### AC #7 — The three Epic 2 blockers are closed in the same Pencil session

**Given** Epic 2 (M1.5, in-app provider-key configuration) is **blocked-by three `.pen` copy revisions** that PR #177 did not address, **and given** this story already opens `ux-design.pen`, **when** this story completes, **then** all three are resolved here.

| # | Screen | Pencil node | Screenshot | Revision |
|---|---|---|---|---|
| 1 | **F2-D-v2** | `S9Rbrq` | `flow-f-subtitle-v2/f2-d-v2.png` | Copy **轉錄 → 抽取內嵌字幕**. M1's primary path extracts an existing embedded track; "轉錄" describes ASR, which is **P2**. The current copy promises a capability M1 does not ship. |
| 2 | **F5-D-v2** | `f6ZxY` | `flow-f-subtitle-v2/f5-d-v2.png` | **前往設定 must reflect M1 behaviour.** In M1 the key is an env-var and there is **no** settings page (that is Epic 2 / FR25), so the link is a dead loop. State the M1 behaviour explicitly — either the link is absent in M1, or it points somewhere real. |
| 3 | **F5-D-v2** | `f6ZxY` | `flow-f-subtitle-v2/f5-d-v2.png` | **FFmpeg framing.** ffmpeg presence is a **deployment concern** (it must be bundled in the Docker image — the 2026-06 audit proved silent degradation when absent), **not a user setting**. Reframe so it does not read as something the user configures. |

**These are copy/framing changes only** — no layout rework, no new screens, no component changes. Revisions 2 and 3 land on the same screen (`f6ZxY`), so that screen is touched once.

**Closing these unblocks `sub-2-1-key-config-page`.** Record in Completion Notes that the Epic 2 blocker is cleared, so `sprint-status.yaml`'s `epic-subtitle-pipeline-m1-5` blocked-by note and `implementation-readiness-report-subtitle-pipeline.md` § 5's open cross-document action can both be closed.

### AC #6 — Screenshots regenerated and *selectively* staged

**Given** the mandatory workflow in `CLAUDE.md`, **when** `ux-design.pen` changes, **then**:

1. `python3 scripts/export-pen-screenshots.py` is run (spawns its own Pencil MCP **stdio** server — safe alongside an active Pencil MCP session).
2. ⚠️ **A full regen is non-deterministic** — every PNG re-renders with byte diffs at identical dimensions. Stage **only** the screens whose design actually changed:
   - `_bmad-output/screenshots/flow-j-specs/j2-d.png` (new — AC #1)
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f2-d-v2.png` (AC #7 revision 1)
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f5-d-v2.png` (AC #7 revisions 2 + 3)

   `git checkout` every other changed PNG so the commit carries no re-render noise. **Exactly three PNGs should appear in `git status` after cleanup** — if a fourth survives, you staged noise.
3. The `.pen` file, the `SCREENS` dict change (the new `j2-d` node only — F2/F5 nodes are already mapped at `scripts/export-pen-screenshots.py:194,198`), and the three screenshots are committed **together**.
4. Commit convention: `feat: update UX design — subtitle-status badge spec + Epic 2 copy revisions`.

### AC #9 — `ux-design-specification.md` is synced (it is still authoritative)

**Given** Alexyu ruled on 2026-07-27 (IR Step 1) that `_bmad-output/planning-artifacts/ux-design-specification.md` **remains authoritative** — it is not superseded by `ux-design.pen` / `ux-redesign/` and is not an archive — **then** this story updates it so it does not silently contradict the new spec screen.

**The sync target is one line, not a 222 KB rewrite.** Located during the IR:

> `ux-design-specification.md:1086` — § Design System Foundation → Customization Strategy:
> `**StatusBadge:** Parse status, download status, metadata source indicators`

That enumeration is the document's definition of what `StatusBadge` covers. After sub-1-7b it also covers **subtitle-pipeline status**, so the line is incomplete. Required:

1. Extend the `StatusBadge` enumeration to include subtitle-pipeline status.
2. Add a pointer to the authoritative detail — the `.pen` spec screen (`flow-j-specs/j2-d.png`) — rather than duplicating the per-state table. **Two sources of truth for the same table is the failure mode this AC exists to prevent**; the doc references, the `.pen` decides.

**Verified scope bound (IR Step 1, 2026-07-27):** a `grep` for `badge` across all 6,373 lines returns exactly **five** hits, and only line 1086 concerns a status badge of this kind (the others are the "Source: Douban" indicator and a "3 pending" nav count). The document has **no per-state badge table** to conflict with, and its subtitle content is entirely about v4-era fansub-filename parsing and subtitle-timeline matching. If you find yourself editing more than ~2 places, stop — the ruling was "sync it", not "rewrite it".

Line ~733 (`9. 狀態指示與回饋 / Status Indicators & Feedback`) is pattern-inspiration prose, **not** a component contract — leave it alone unless it makes a claim the new states falsify.

### AC #8 — Scope fence

- ❌ **No frontend code.** Zero files under `apps/web/`. Implementation is sub-1-7b.
- ❌ No new colour tokens, no change to `StatusDescriptor`'s shape, no new badge component.
- ❌ No change to the four existing `subtitle_status` values' treatment (`not_searched` / `searching` / `found` / `not_found`) — except the AC #5 `searching` tint *ruling*, which may be "leave it".
- ❌ No Activity-hub design (sub-1-6 owns the SSE progress surface).
- ❌ **AC #7 is copy/framing only.** Do not take the open `.pen` file as licence for layout rework, new screens, or unrelated polish on F2/F5 — that would put un-reviewed design changes inside an M1 story and blow the "exactly three PNGs" check.
- ❌ No Epic 2 *design* work beyond the three revisions (the key-config page itself is `sub-2-1`).

---

## Tasks / Subtasks

> **⚠️ Task order is deliberate — do the decided work first.** The three Epic 2 copy revisions (AC #7) are **already ruled**; the badge spec (AC #1–#5) may need a round of iteration with Alexyu. Authoring the badge screen first would leave the copy fixes sitting in the same uncommitted `.pen` while the badge is debated — Epic 2 stays blocked on a decision that has nothing to do with it. Land the settled work first so it is always in a committable state. (Sally, party-mode 2026-07-27.)

- [ ] **Task 1 — Session setup + close the three Epic 2 copy revisions (AC #7)**
  - [ ] 1.1 Confirm Pencil.app is running; `get_editor_state(include_schema: true)` before any other Pencil MCP call.
  - [ ] 1.2 F2-D-v2 (node `S9Rbrq`): 轉錄 → 抽取內嵌字幕.
  - [ ] 1.3 F5-D-v2 (node `f6ZxY`): state the M1 behaviour of 前往設定 (no settings page exists until Epic 2 / FR25 — the current link is a dead loop).
  - [ ] 1.4 F5-D-v2 (same node): reframe FFmpeg as a deployment concern, not a user setting.
  - [ ] 1.5 Re-check label/title overlap on both edited screens — copy length changed.
- [ ] **Task 2 — Design the badge spec screen (AC #1–#5)**
  - [ ] 2.1 Read `j1-d` (the existing `flow-j-specs` screen) to match the spec-screen layout convention.
  - [ ] 2.2 Author the new standalone spec screen: the AC #2 ruling sentence, the AC #3 per-state table rendered as real badge/icon samples (not prose), the AC #4 copy resolutions, the AC #5 flag.
  - [ ] 2.3 Verify no label/title overlaps other content (the recurring Pencil pitfall).
  - [ ] 2.4 If the badge design needs another round with Alexyu, **Task 1's work is already settled** — it can be exported and committed on its own rather than waiting.
- [ ] **Task 3 — Export + wire the screenshots (AC #6)**
  - [ ] 3.1 Add **only** the new j2 node id to `SCREENS` in `scripts/export-pen-screenshots.py` → `("flow-j-specs", "j2-d")`. F2/F5 are already mapped (`:194`, `:198`) — do not touch those lines.
  - [ ] 3.2 Run `python3 scripts/export-pen-screenshots.py`.
  - [ ] 3.3 Stage exactly three PNGs (`j2-d.png`, `f2-d-v2.png`, `f5-d-v2.png`) + `ux-design.pen` + the script change; `git checkout` every other regenerated PNG. Verify `git status` shows **three** PNGs, no more.
- [ ] **Task 4 — Sync `ux-design-specification.md` (AC #9)**
  - [ ] 4.1 Extend the `StatusBadge` enumeration at `~:1086` to include subtitle-pipeline status.
  - [ ] 4.2 Add a pointer to `flow-j-specs/j2-d.png` as the authoritative per-state detail — **do not copy the table in** (two sources of truth is the failure mode).
  - [ ] 4.3 Re-grep `badge` to confirm nothing else in the document now contradicts the spec screen. Expected: ≤ 2 edited locations.
- [ ] **Task 5 — Hand off**
  - [ ] 5.1 Record the ratified table (labels, tints, icons) in Completion Notes so sub-1-7b can implement without re-opening Pencil.
  - [ ] 5.2 If any proposal in AC #3/#4/#5 was overruled, say so explicitly — 7b's ACs quote the proposal and must be corrected if it changed.
  - [ ] 5.3 Record that the **Epic 2 blocker is cleared** (AC #7), naming all three revisions, so `sprint-status.yaml`'s `epic-subtitle-pipeline-m1-5` / `sub-2-1-key-config-page` blocked-by notes and the IR report § 5 open action can be closed.

---

## Dev Notes

- `ux-design.pen` is encrypted: **Pencil MCP tools only**, never `Read`/`Grep`.
- `flow-j-specs/` currently holds exactly one screen (`j1-d.png`) — desktop-only (`-d`), no mobile counterpart. Follow that: spec screens are reference material, not responsive surfaces.
- The six tint tokens are defined at `apps/web/src/utils/libraryStatus.ts:30-37` (`success` / `accent` / `warning` / `error` / `info` / `neutral`) as CSS-var pairs. Reuse; do not invent.
- Existing icon vocabulary in `EpisodeList.tsx:35-40`: `CheckCircle2` / `XCircle` / `Loader2` (spin) / `Minus`, all `lucide-react`.

### Time-dependent visual coverage

**N/A — no `apps/web/src/components/**` files touched.** This story produces a `.pen` screen and one screenshot; zero React components change. Rule 23 does not apply.

### References

- [Source: `epics-subtitle-pipeline.md`#Story 1.7a] · Alexyu ruling 2026-07-27 (promote FE badges into M1).
- [Source: `sub-1-2-pipeline-state-model.md`#AC #2] — the `[@contract-v1]` 9-value set and the `no_text_source` vs `skipped` semantic table.
- [Source: `apps/web/src/utils/libraryStatus.ts`:1-15,30-37,87-91,102-137] — the governing principle, the tint tokens, the 2026-07-05 accent ruling, `pickPosterBadge`'s exception-signal rule.
- [Source: `apps/web/src/components/media/EpisodeList.tsx`:33-60] — the `SUBTITLE_STATUS` icon map and the `searching`/`warning` inconsistency.
- [Source: `CLAUDE.md`#UX Design Screenshots Workflow] — mandatory regen + selective staging.
- [Source: `.claude/memory/feedback_pencil_spec_standalone_screen.md`] — spec screens stand alone (bugfix-10-6).
- [Source: `.claude/memory/feedback_pencil_label_overlap.md`] — labels must not overlap.

---

## Dev Agent Record

### Agent Model Used

### Debug Log References

### Completion Notes List

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - If **NO** beyond the two pre-recorded items: state `N/A — no further out-of-scope work discovered`.
  - **① expand-scope-in-place → AC #7.** The three Epic 2 `.pen` copy revisions were pre-existing open work (IR report 2026-07-26 § 5, "⚠️ Open"; tracked in `sprint-status.yaml` as Epic 2's blocked-by note). Alexyu ruled 2026-07-27 to absorb them into this story's Pencil session rather than run a second `.pen` change — the added **AC #7** is what tracks them, per lane ①'s requirement that absorbed work gets its own AC.
  - **① expand-scope-in-place → AC #5.** The pre-existing `searching` = `--warning` + spin inconsistency (vs accent-reserved-for-in-progress) is ruled on here; any resulting code change is carried by sub-1-7b's AC #3. Not a deferred discovery.

### File List
