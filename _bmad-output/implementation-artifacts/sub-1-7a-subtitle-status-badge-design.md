# Story sub-1.7a: Subtitle-status badge design spec + Epic 2 `.pen` copy unblock

Status: review

**Epic:** `epic-subtitle-pipeline-m1` (M1) · **Risk: 🟢 LOW** · **UX / DESIGN-ONLY** (Sally — `ux-designer`, **not** dev)
**Origin:** Alexyu ruling 2026-07-27 — *"前端 badge 讓 M1 出貨時就帶 badge"*. Promotes the lane-③ entry `backlog-subtitle-status-fe-rendering` (filed by sub-1-2) into M1 scope.
**Scope expanded 2026-07-27 (Alexyu, party-mode):** the **three open `.pen` copy revisions that block Epic 2** are batched into this same Pencil session — see **AC #7**. Rationale: one `.pen` edit, one screenshot regeneration, one commit. Splitting them means paying the non-deterministic re-render risk twice for changes to the same file.
**Depends on:** sub-1-2's `[@contract-v1]` `SubtitleStatus` 9-value set (the *contract*, not its merge — design can start immediately).
**Blocks:** sub-1-7b (frontend implementation verifies against this spec's screenshot) **and — via AC #7 — `sub-2-1b-key-config-page` / `epic-subtitle-pipeline-m1-5` (Epic 2), which are blocked on exactly those three revisions.** (2-1 was split into 2-1a backend / 2-1b frontend on 2026-07-27; the `.pen` dependency sits on the frontend half.)
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

**Closing these unblocks `sub-2-1b-key-config-page`** (the frontend half of the 2026-07-27 split; 2-1a is backend and never depended on the `.pen` work). Record in Completion Notes that the Epic 2 blocker is cleared, so `sprint-status.yaml`'s `epic-subtitle-pipeline-m1-5` blocked-by note and `implementation-readiness-report-subtitle-pipeline.md` § 5's open cross-document action can both be closed.

### AC #6 — Screenshots regenerated and *selectively* staged

**Given** the mandatory workflow in `CLAUDE.md`, **when** `ux-design.pen` changes, **then**:

1. `python3 scripts/export-pen-screenshots.py` is run (spawns its own Pencil MCP **stdio** server — safe alongside an active Pencil MCP session).
2. ⚠️ **A full regen is non-deterministic** — every PNG re-renders with byte diffs at identical dimensions. Stage **only** the screens whose design actually changed:
   - `_bmad-output/screenshots/flow-j-specs/j2-d.png` (new — AC #1)
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f2-d-v2.png` (AC #7 revision 1)
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f5-d-v2.png` (AC #7 revisions 2 + 3)
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f1-d-v2.png` (**AC #10** — same 轉錄 fix)
   - `_bmad-output/screenshots/flow-f-subtitle-v2/f1-m-v2.png` (**AC #10** — same 轉錄 fix)

   `git checkout` every other changed PNG so the commit carries no re-render noise. **Exactly five PNGs should appear in `git status` after cleanup** — if a sixth survives, you staged noise. *(Count amended from three to five on 2026-08-03 by AC #10; the guard is unchanged in kind — an exact expected count, not a ceiling to be relaxed on sight.)*
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

### AC #10 — [Rule 24 lane ①, added at implementation 2026-08-03] The 轉錄 over-promise on F1-D-v2 / F1-M-v2

The Pencil session found the string AC #7 revision 1 exists to remove — `轉錄＋AI 翻譯，約需數分鐘` — **verbatim on two further screens** that the architecture's V2 ① inventory missed:

| Screen | Pencil node | Text node | Screenshot |
|---|---|---|---|
| F1-D-v2 | `r1EY9` | `X7exGq` | `flow-f-subtitle-v2/f1-d-v2.png` |
| F1-M-v2 | `JkdfH` | `qR6hi` | `flow-f-subtitle-v2/f1-m-v2.png` |

**Therefore:** the same one-word substitution (轉錄 → 抽取內嵌字幕) is applied to both, absorbed into this story rather than deferred. Reasoning, in order:

1. **This is not a new ruling.** The decision — "M1 extracts an existing embedded track; 轉錄 describes ASR, which is P2" — was made in architecture V2 ①. What was incomplete was its *inventory*, not its content. Applying a settled ruling to instances it missed is execution, not scope creep.
2. **The story's own batching rationale applies verbatim.** AC #7 exists because splitting `.pen` work "means paying the non-deterministic re-render risk twice for changes to the same file". Deferring these two would do exactly that, for a one-word copy fix.
3. **F1 is the generation *entry* surface** — arguably more load-bearing than F2 for the false promise, since it is where the user decides to start.

AC #6's expected PNG count moves 3 → 5 in the same edit, so the "exactly N" guard keeps working rather than being silently blunted.

**Deliberately NOT absorbed — checked and ruled, not overlooked:** `UNVRU` (D6-D-v2, 連線失效 fail-soft) also carries a 前往設定 button, which the Pencil session flagged as a possible sibling of AC #7 revision 2. **It is not a dead link and stays untouched:** `apps/web/src/routes/settings/` ships 11 real pages including `connection.tsx` and `qbittorrent.tsx`, so a download/connection failure has a genuine settings destination. AC #7 revision 2 is specific to the *translation key*, which has no page until FR25 (Epic 2).

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

- [x] **Task 1 — Session setup + close the three Epic 2 copy revisions (AC #7)**
  - [x] 1.1 Confirm Pencil.app is running; `get_editor_state(include_schema: true)` before any other Pencil MCP call.
  - [x] 1.2 F2-D-v2 (node `S9Rbrq`): 轉錄 → 抽取內嵌字幕.
  - [x] 1.3 F5-D-v2 (node `f6ZxY`): state the M1 behaviour of 前往設定 (no settings page exists until Epic 2 / FR25 — the current link is a dead loop).
  - [x] 1.4 F5-D-v2 (same node): reframe FFmpeg as a deployment concern, not a user setting.
  - [x] 1.5 Re-check label/title overlap on both edited screens — copy length changed.
- [x] **Task 2 — Design the badge spec screen (AC #1–#5)**
  - [x] 2.1 Read `j1-d` (the existing `flow-j-specs` screen) to match the spec-screen layout convention.
  - [x] 2.2 Author the new standalone spec screen: the AC #2 ruling sentence, the AC #3 per-state table rendered as real badge/icon samples (not prose), the AC #4 copy resolutions, the AC #5 flag.
  - [x] 2.3 Verify no label/title overlaps other content (the recurring Pencil pitfall).
  - [x] 2.4 If the badge design needs another round with Alexyu, **Task 1's work is already settled** — it can be exported and committed on its own rather than waiting.
- [x] **Task 3 — Export + wire the screenshots (AC #6)**
  - [x] 3.1 Add **only** the new j2 node id to `SCREENS` in `scripts/export-pen-screenshots.py` → `("flow-j-specs", "j2-d")`. F2/F5 are already mapped (`:194`, `:198`) — do not touch those lines.
  - [x] 3.2 Run `python3 scripts/export-pen-screenshots.py`.
  - [x] 3.3 Stage exactly three PNGs (`j2-d.png`, `f2-d-v2.png`, `f5-d-v2.png`) + `ux-design.pen` + the script change; `git checkout` every other regenerated PNG. Verify `git status` shows **three** PNGs, no more.
- [x] **Task 4 — Sync `ux-design-specification.md` (AC #9)**
  - [x] 4.1 Extend the `StatusBadge` enumeration at `~:1086` to include subtitle-pipeline status.
  - [x] 4.2 Add a pointer to `flow-j-specs/j2-d.png` as the authoritative per-state detail — **do not copy the table in** (two sources of truth is the failure mode).
  - [x] 4.3 Re-grep `badge` to confirm nothing else in the document now contradicts the spec screen. Expected: ≤ 2 edited locations.
- [x] **Task 5 — Hand off**
  - [x] 5.1 Record the ratified table (labels, tints, icons) in Completion Notes so sub-1-7b can implement without re-opening Pencil.
  - [x] 5.2 If any proposal in AC #3/#4/#5 was overruled, say so explicitly — 7b's ACs quote the proposal and must be corrected if it changed.
  - [x] 5.3 Record that the **Epic 2 blocker is cleared** (AC #7), naming all three revisions, so `sprint-status.yaml`'s `epic-subtitle-pipeline-m1-5` / `sub-2-1b-key-config-page` blocked-by notes and the IR report § 5 open action can be closed.

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

Amelia (Developer Agent) · Claude Fable 5, effort xhigh · 2026-08-03 — **repo-side only**. The `.pen` edits were executed by **Pencil's own Inline AI Agent** against a prompt authored here, deliberately: driving the edits through Pencil MCP from this session would have cost a full `get_app_state` canvas load per round. This session authored the prompt (including every ruling, literal and token value), ruled on the three questions the inline agent escalated, and did all repo-side work: `SCREENS`, the export run, selective staging, the doc sync, and this record.

### Debug Log References

| Symptom | Diagnosis | Resolution |
|---|---|---|
| `python3 scripts/export-pen-screenshots.py` → `ERROR: Pencil.app not found at /Applications/Pencil.app` | The app shipped as `Pencil.app` and has been **renamed to `Pen.app`** (v1.2.2). `MCP_BIN` was a hardcoded absolute path, so the rename read as "the export script is broken". | `MCP_BIN_CANDIDATES` + `resolve_mcp_bin()` probes `Pen.app` then `Pencil.app`; the error now prints every path it tried. Export then ran **147/147**. |
| Staging step reverted the 4 tracked PNGs it was meant to keep | `git add $KEEP` with an unquoted multi-path variable resolved as ONE pathspec → `git add` failed → the following `git checkout -- _bmad-output/screenshots` then reverted everything, including the 4 intended files. | Re-ran the export (the `.pen` is the source of truth, so nothing was lost) and staged the four with explicit `--` separated arguments before the checkout. Final `git status` verified at exactly five PNGs. |

### Completion Notes List

- 🎯 **The ratified table — this is sub-1-7b's authoritative handoff** (Task 5.1). Read this, not the PNG (see the ⚠️ below):

  | `subtitle_status` | Poster / list badge | Badge tint | `EpisodeList` icon | Icon colour | `aria-label` / tooltip |
  |---|---|---|---|---|---|
  | `probing` | **none** (Activity hub) | — | `Loader2` + spin | `--accent-primary` `#3b82f6` | `偵測字幕軌中` |
  | `extracting` | **none** | — | `Loader2` + spin | `--accent-primary` `#3b82f6` | `抽取字幕中` |
  | `translating` | **none** | — | `Loader2` + spin | `--accent-primary` `#3b82f6` | `翻譯字幕中` |
  | `no_text_source` | **`無字幕源`** | `neutral` — bg `#2e3b56` / text `#a0aabe` | `XCircle` | `--text-muted` `#a0aabe` | `無可用字幕來源` · tooltip `此檔案沒有可用的文字字幕軌` |
  | `skipped` | **`已略過`** | `neutral` — bg `#2e3b56` / text `#a0aabe` | `Minus` | `--text-muted` `#a0aabe` | `已略過字幕生成` · tooltip `字幕軌語言非英文，已依規則略過` |

  Plus the AC #5 ruling: **`searching` is re-tinted `--warning` → `--accent-primary`** (it IS an in-progress state; two colours for one meaning next to the three new spinners reads as a distinction that does not exist). That change belongs to 7b, with this as its stated reason.

- ⚠️ **AC #3's proposal cited a token that does not exist.** `--accent` is not defined in `apps/web/src/styles.css` and `var(--accent)` has **zero** occurrences repo-wide; the real tokens are `--accent-primary` `#3b82f6` / `--accent-tint` `#3b82f61f` / `--accent-text` `#60a5fa`. Caught while authoring the prompt, corrected to `--accent-primary` in both the spec screen and the table above. Copying the story's proposal verbatim would have shipped 7b an invalid CSS var that silently renders as no colour.
- ✅ **AC #2 / #4 / #5 rulings** are as proposed in the story, with two sharpenings: `無字幕源` and `缺字幕` stay **distinct** (different next actions — re-search vs P2 ASR only; merging invites users to retry something that cannot work), and `searching` is **re-tinted** rather than left alone.
- ✅ **AC #7 — the Epic 2 blocker is CLEARED.** All three revisions landed: ① `S9Rbrq`/`TXYYF` 轉錄＋AI 翻譯 → 抽取內嵌字幕＋AI 翻譯 · ② `f6ZxY` panel retitled `尚未設定翻譯服務金鑰` + body `請設定 CLAUDE_API_KEY 環境變數後重啟伺服器。設定頁面將於 M1.5 提供。`, the 前往設定 button (`UcEr3`) **deleted** and replaced with a `查看部署說明` text link (`iQb2i`) plus the note `M1：金鑰為環境變數，無設定頁（FR25 屬 Epic 2）` (`LMH8J`) · ③ FFmpeg reframed as deployment fact via the same body rewrite plus `FFmpeg／FFprobe 已內建於 Docker 映像檔，無需另行安裝。` (`QGf46`). **The panel's copy is byte-aligned with sub-1-6's shipped 409 `AI_NOT_CONFIGURED` response** — design and backend now say the same sentence. `sprint-status.yaml`'s `epic-subtitle-pipeline-m1-5` / `sub-2-1b-key-config-page` blocked-by notes and `implementation-readiness-report-subtitle-pipeline.md` § 5 / § Recommendations item 2 are all updated.
- ✅ **AC #10** — the same 轉錄 fix applied to `X7exGq` (F1-D-v2) and `qR6hi` (F1-M-v2). The inline agent also surfaced, and correctly **did not** touch, the Flow-G 轉錄 strings (Whisper ASR progress / stage labels) — those describe ASR itself and are correct.
- ✅ **`UNVRU` (D6-D-v2)'s 前往設定 verified as a live link, not a sibling defect** — `apps/web/src/routes/settings/` ships 11 real pages including `connection.tsx` and `qbittorrent.tsx`. Left untouched deliberately; see AC #10.
- ⚠️ **The exported PNG is a 204×400 thumbnail — the spec text in it is NOT readable.** This is Pencil `get_screenshot`'s cap, not a regression: every screenshot in the repo is ≤400px on its long edge (`j1-d` 222×400, `design-system-reference` 131×400). It undercuts AC #1's premise that 7b "verifies against the screenshot" — for a mockup a thumbnail is a fine reminder, for a **text-dense spec sheet** it is not. Two consequences, both handled: the ratified table above is the real handoff (which is exactly why Task 5.1 exists), and `backlog-pen-spec-screen-readable-export` is filed (lane ③). 7b's UX gate should compare against the **`.pen` screen in Pencil**, not the PNG.
- 📐 **Three escalations from the inline agent, ruled here:** ① caption at 45px (convention) rather than matching `XlFIq`'s 30px — `XlFIq` is the outlier, and since captions sit *outside* the frame they do not enter the export, so re-aligning `j1` would cost a PNG re-render for zero pixels of deliverable; left as-is. ② `J2-D` height 2435 > `j1`'s 2241 — content-driven, width held at 1240, 46px clearance to the next block verified. ③ the F5 warning panel keeps `$warning-tint` — the capability genuinely is unavailable, so fail-soft warning semantics are still correct; only the copy was wrong.
- 🧾 **AC #9 sync is one line.** `ux-design-specification.md:1086`'s `StatusBadge` enumeration now includes subtitle-pipeline status and **points at** `flow-j-specs/j2-d.png` rather than restating the table (two sources of truth is the failure mode the AC names). Re-grepped `badge` across the document afterwards: every other hit is a parse-pending count, the Douban source indicator, a rating/episode-count overlay, or the `--radius-sm` comment — nothing contradicts the new spec. **1 location edited**, inside the AC's "≤ 2" bound.
- 🚧 **Outstanding gate:** Alexyu's own eyes on `J2-D` **in Pencil** (not the thumbnail). Everything mechanically verifiable here is verified; the screen's rendered content was taken from the inline agent's itemised report, not independently re-read, because re-reading it from this session is precisely the Pencil-MCP cost the split was designed to avoid.

### Discovery Triage

- **① expand-scope-in-place → AC #7.** (pre-recorded at drafting) The three Epic 2 `.pen` copy revisions, absorbed per Alexyu's 2026-07-27 ruling.
- **① expand-scope-in-place → AC #5.** (pre-recorded at drafting) The `searching` = `--warning` inconsistency; ruled **re-tint**, executed by 7b.
- **① expand-scope-in-place → AC #10 (added at implementation 2026-08-03).** The 轉錄 over-promise exists verbatim on F1-D-v2 / F1-M-v2, which architecture V2 ①'s inventory missed. Absorbed rather than deferred: same settled ruling, one-word fix, and the story's own "don't pay the non-deterministic re-render twice" logic. AC #6's expected PNG count moved 3 → 5 in the same edit so the guard stays exact.
- **① expand-scope-in-place → `scripts/export-pen-screenshots.py` app-path fix.** `Pencil.app` → `Pen.app` (v1.2.2) broke the script outright; the story cannot satisfy AC #6 without it, so it is in-scope by necessity. Fixed as a candidate probe rather than a second hardcoded path, so the next rename degrades to a clear message instead of a broken run.
- **③ backlog-with-carry-forward-link → `backlog-pen-spec-screen-readable-export`.** Spec screens export at ≤400px, leaving text-dense spec sheets unreadable in the committed PNG (affects `j1-d` and now `j2-d`). Filed at discovery with a bidirectional link to this story; candidate fixes are a scale parameter on `get_screenshot` or an `export_html` sidecar for `flow-j-specs` only.
- **Not a discovery — checked and cleared:** `UNVRU`'s 前往設定 (live link, 11 real settings pages) and Flow G's 轉錄 strings (correctly describe ASR).

### File List

| File | Change |
|---|---|
| `ux-design.pen` | **modified** — NEW spec screen `J2-D` (node `ZpQaw`, 1240×2435 at x=18380/y=24900, caption node `cFBCj`) with the five sections J2-1…J2-5; copy revisions on `S9Rbrq` (`TXYYF`), `f6ZxY` (`r9CdQk`/`dDOH6`, `UcEr3` deleted, `iQb2i`/`LMH8J`/`QGf46` added inside new frame `i4HF1a`), `r1EY9` (`X7exGq`), `JkdfH` (`qR6hi`) |
| `scripts/export-pen-screenshots.py` | **modified** — `SCREENS` += `"ZpQaw": ("flow-j-specs", "j2-d")` (AC #1); `MCP_BIN_CANDIDATES` + `resolve_mcp_bin()` for the `Pencil.app` → `Pen.app` rename, with an error message that names every probed path |
| `_bmad-output/screenshots/flow-j-specs/j2-d.png` | **new** — the spec screen (AC #1) |
| `_bmad-output/screenshots/flow-f-subtitle-v2/f2-d-v2.png` | **modified** — AC #7 revision 1 |
| `_bmad-output/screenshots/flow-f-subtitle-v2/f5-d-v2.png` | **modified** — AC #7 revisions 2 + 3 |
| `_bmad-output/screenshots/flow-f-subtitle-v2/f1-d-v2.png` · `f1-m-v2.png` | **modified** — AC #10 (same 轉錄 fix) |
| `_bmad-output/planning-artifacts/ux-design-specification.md` | **modified** — AC #9: `StatusBadge` enumeration at `:1086` + pointer to `j2-d.png` (one line; no table duplicated) |
| `_bmad-output/planning-artifacts/implementation-readiness-report-subtitle-pipeline.md` | **modified** — § 5 open action + § Recommendations item 2 closed (the three `.pen` revisions shipped here) |
| `_bmad-output/implementation-artifacts/sprint-status.yaml` | **modified** — `sub-1-7a` → `review`; Epic 2 / `sub-2-1b` blocked-by notes cleared; `backlog-pen-spec-screen-readable-export` filed |

- **Did this story discover any work outside its current scope?**
  - If **NO** beyond the two pre-recorded items: state `N/A — no further out-of-scope work discovered`.
  - **① expand-scope-in-place → AC #7.** The three Epic 2 `.pen` copy revisions were pre-existing open work (IR report 2026-07-26 § 5, "⚠️ Open"; tracked in `sprint-status.yaml` as Epic 2's blocked-by note). Alexyu ruled 2026-07-27 to absorb them into this story's Pencil session rather than run a second `.pen` change — the added **AC #7** is what tracks them, per lane ①'s requirement that absorbed work gets its own AC.
  - **① expand-scope-in-place → AC #5.** The pre-existing `searching` = `--warning` + spin inconsistency (vs accent-reserved-for-in-progress) is ruled on here; any resulting code change is carried by sub-1-7b's AC #3. Not a deferred discovery.

### File List
