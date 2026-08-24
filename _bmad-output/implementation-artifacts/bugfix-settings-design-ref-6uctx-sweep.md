# Story bugfix: settings `6UCtX` Design-ref Sweep

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a developer relying on Rule 21 traceability headers,
I want every `// Design ref:` header that cites `6UCtX` to point at the screen frame that actually shows that component,
so that design-verification steps (dev workflow Step 9, Sally's MCP reviews, future drift sweeps) compare implementations against the RIGHT design instead of the qBittorrent connection page.

## Background — why this exists (Rule 24 ③ origin)

- Filed at discovery time by `9R-10b-m4-unsupported-state-frontend` create-story (2026-08-21).
- `9R-UX-auto-generation-toggle-design` field-verified: **`6UCtX` (frame `C4-D`) is the 連線設定 (qBittorrent) settings tab** — host/username/password form. Its sidebar has 7 items and **no 媒體庫掃描**. The real library-scan screens are `KvZSc` (E1-D) / `uABWl` (E1-M).
- Root cause located (2026-08-24, Bob, via `_bmad-output/audit/drift-sweep-2026-05.md`): story 19-8's sweep **blanket-upgraded every settings screen-section placeholder to `6UCtX`** and recorded them all as `exact` (`screen-section-upgraded-to-6UCtX` rows, audit doc lines ~109–188). One systematic decision, not 17 independent mistakes.
- Precedent (single-file fix): `bugfix-libraryeditmodal-wrong-design-ref` — fixed by 9R-10b Task 5; `LibraryEditModal.tsx:1` now reads `// Design ref: ux-design.pen Screen E5-D (hUVYm) · E5-M (P0P82x) · J4-D (sPzZT) · J5-D (alrIw)` — that is the header format to follow.
- **Why per-file judgment, not global replace:** for some files (the qBT connection form itself) `6UCtX` is CORRECT. A `sed` sweep would break the correct ones.

## Acceptance Criteria

1. Every file under `apps/web/src/` whose leading Rule 21 header cites `6UCtX` (17 files as of 2026-08-24 — see mapping table) has been individually verified against the live `.pen` file via **read-only** Pencil MCP (`Get` on the candidate frame; confirm the component's UI actually appears there), and its header updated per the mapping table (or per a documented correction to the table if verification disagrees).
2. After the sweep, `grep -rn "Screen 10 Settings Desktop" apps/web/src` returns **0 hits** — the pre-2026-06-05 legacy screen name is gone from all 6UCtX-citing headers (files where `6UCtX` is genuinely correct are relabeled to the current frame name `C4-D`, not left stale).
3. Files that genuinely render on the connection-settings screen keep `6UCtX` with the corrected label (`Screen C4-D (6UCtX) · C4-M (2H4OM)` where the mobile frame applies equally).
4. Library-scan components (`MediaLibraryManager.tsx`, `LibraryCard.tsx`) point to `Screen E1-D (KvZSc) · E1-M (uABWl)`; `LibraryCard.tsx` **retains** its existing `· J4-D (sPzZT) · J5-D (alrIw)` spec refs.
5. Components whose settings tab has **no screen frame in the `.pen` at all** (快取管理 / 系統日誌 / 服務狀態 / 匯出/匯入 exist only as sidebar items — verified 2026-08-24) use the honest design-coverage-gap form: `// Design ref: ux-design.pen — no current screen frame; {tab} tab was never given a frame — rides the designed settings shell (Screen C4-D, 6UCtX)`, mirroring the `ApiKeysForm.tsx` precedent. `ApiKeysForm.tsx`'s own stale parenthetical `(Screen 10 Settings Desktop, 6UCtX)` is updated to `(Screen C4-D, 6UCtX)` in the same pass.
6. Every rewritten header still satisfies `local/implements-pen-node-id` (`pnpm nx lint web` green) and remains the file's **leading** comment (before any import).
7. An addendum section is appended to `_bmad-output/audit/drift-sweep-2026-05.md` (do NOT rewrite the original 19-8 rows — it is a durable record) titled `## Addendum 2026-08 — 6UCtX correction (bugfix-settings-design-ref-6uctx-sweep)`, containing the final per-file mapping table so future re-scans grep against corrected truth.
8. Comment-only + docs change: **zero** rendering/behavior diff, **zero** `.pen` modification (MCP reads only — so the CLAUDE.md screenshot-regen workflow is NOT triggered), zero visual-baseline changes, `pnpm nx test web` untouched-green, `pnpm run format:check` green, tsc error count = main baseline.

## Mapping Table (pre-verified by Bob 2026-08-24 via Pencil MCP — dev MUST re-confirm each row per AC #1)

Ground truth established this session: `6UCtX`/C4-D = 連線設定 (qBT form: 主機位址/使用者名稱/密碼/測試連線/儲存設定); `2H4OM`/C4-M = its mobile twin; `uhAKd`/C5-D = 備份與還原; `KvZSc`/E1-D + `uABWl`/E1-M = 媒體庫掃描 (媒體資料夾/新增資料夾/掃描排程/掃描媒體庫). A full-document text scan found 快取管理/系統日誌/服務狀態/匯出/匯入/效能監控 appear **only as sidebar items** in C4/C5/E1 — no dedicated frames exist.

| # | File (`apps/web/src/components/`) | Verdict | New header |
|---|---|---|---|
| 1 | `settings/QBittorrentForm.tsx` | ✅ ID correct, label stale | `// Design ref: ux-design.pen Screen C4-D (6UCtX) · C4-M (2H4OM)` |
| 2 | `settings/ConnectionTestResult.tsx` | ✅ ID correct (測試連線 lives on C4) | same as #1 |
| 3 | `settings/MediaLibraryManager.tsx` | ❌ wrong screen | `// Design ref: ux-design.pen Screen E1-D (KvZSc) · E1-M (uABWl)` |
| 4 | `settings/LibraryCard.tsx` | ❌ wrong screen (keep J refs) | `// Design ref: ux-design.pen Screen E1-D (KvZSc) · E1-M (uABWl) · J4-D (sPzZT) · J5-D (alrIw)` |
| 5 | `settings/CacheManagement.tsx` | ❌ no frame exists | gap form, `{tab}` = 快取管理 (AC #5) |
| 6 | `settings/CacheTypeCard.tsx` | ❌ no frame exists | gap form, `{tab}` = 快取管理 |
| 7 | `settings/LogsViewer.tsx` | ❌ no frame exists | gap form, `{tab}` = 系統日誌 |
| 8 | `settings/LogEntry.tsx` | ❌ no frame exists | gap form, `{tab}` = 系統日誌 |
| 9 | `settings/LogFilters.tsx` | ❌ no frame exists | gap form, `{tab}` = 系統日誌 |
| 10 | `settings/ServiceStatusDashboard.tsx` | ❌ no frame exists | gap form, `{tab}` = 服務狀態 |
| 11 | `settings/ServiceStatusCard.tsx` | ❌ no frame exists | gap form, `{tab}` = 服務狀態 |
| 12 | `settings/MetadataExport.tsx` | ❌ no frame exists (the 匯出中繼資料 hits in B2/B5 are the detail-page context-menu item, a different component) | gap form, `{tab}` = 匯出/匯入 |
| 13 | `settings/ExploreBlocksSettings.tsx` | ❌ wrong screen — real refs already sit in its inner doc-comment (`HP-5 (Y5XvRv)`, legacy name) | `// Design ref: ux-design.pen Screen H5-D (Y5XvRv) · H3 (Paqlk)`; reconcile the inner doc-comment's `Screen HP-5` to `Screen H5-D` so the file agrees with itself |
| 14 | `learning/LearnedPatternsSettings.tsx` | ❌ no frame — learning postdates the design (19-8 classed `learning/LearnPatternPrompt` as design-gap; this file should match) | `// Design ref: ux-design.pen — no current screen frame; learning feature postdates the .pen design (epic-19-8 sweep finding)` |
| 15 | `health/QBStatusIndicator.tsx` | ⚠️ judgment: Story 4.6 status indicator — check where it actually renders (header? settings?). If it IS the qBT status inside 連線設定, `6UCtX` may be right; if it is the sidebar footer status, `Implements: Component/SidebarFooterStatus (PrmQG)` fits better | dev decides from render location + `.pen` read; record in addendum |
| 16 | `health/ConnectionHistoryPanel.tsx` | ⚠️ judgment: side-panel (SidePanel) for connection history — C4-D shows **no** history panel. Likely gap form (`connection-history side panel postdates the .pen design`) unless a frame is found | dev decides; record in addendum |
| 17 | `settings/ApiKeysForm.tsx` | ✅ already the honest gap form — only the parenthetical is stale | update `(Screen 10 Settings Desktop, 6UCtX)` → `(Screen C4-D, 6UCtX)` |

## Tasks / Subtasks

- [x] Task 1 — Re-verify the mapping table (AC: #1)
  - [x] 1.1 Pencil.app must be running with `ux-design.pen`; use Pencil MCP `execute` with **read-only** `Get`/`Print` (never `Insert`/`Update`/`Delete`)
  - [x] 1.2 For rows 1–14 + 17: spot-confirm the target frame's content matches the component (the ground-truth dump in this story is from 2026-08-24; re-check cheaply, don't re-derive)
  - [x] 1.3 For rows 15–16: read the component's render location in the app code, then confirm/deny against C4-D and `Component/SidebarFooterStatus` (`PrmQG`); write the verdict down
- [x] Task 2 — Apply the 17 header edits (AC: #2–#5)
  - [x] 2.1 Keep every header as the file's first line (leading-comment requirement, story 19-3 AC #2)
  - [x] 2.2 Do NOT touch any header that does not cite `6UCtX` (see Out of Scope)
- [x] Task 3 — Audit-doc addendum (AC: #7)
  - [x] 3.1 Append the final table (incl. rows 15–16 verdicts) to `_bmad-output/audit/drift-sweep-2026-05.md` as a new addendum section; leave 19-8's original rows untouched
- [x] Task 4 — Gates (AC: #6, #8)
  - [x] 4.1 `grep -rn "Screen 10 Settings Desktop" apps/web/src` → 0 hits
  - [x] 4.2 `grep -rln 6UCtX apps/web/src` → only rows 1, 2, 17 (plus 15 if verdict keeps it)
  - [x] 4.3 `pnpm nx lint web` green (every new header must match `local/implements-pen-node-id`'s `DESIGN_REF_RE`)
  - [x] 4.4 `pnpm nx test web` green, `pnpm run format:check` green, tsc count = main baseline
  - [x] 4.5 `git diff` shows comment/docs lines only — zero logic lines touched

### Review Follow-ups (AI)

- [x] [AI-Review][MEDIUM] `settings/SettingsLayout.tsx` claimed `<utility — no .pen counterpart>` while rendering the very settings shell (連線設定/快取管理/… sidebar) that C4-D/C5-D/E1-D draw — and that this story's 8 new gap-form headers cite as "the designed settings shell". False exemption, same defect family as the sweep. Fixed → `Screen C4-D (6UCtX) · C4-M (2H4OM)` [SettingsLayout.tsx:1]
- [x] [AI-Review][LOW] Same-directory, same-node-ID label divergence created by this story's own edits: `ScannerSettings.tsx` still said `Screen H1 Settings Scanner Desktop (KvZSc)` next to the new `E1-D (KvZSc)`, and `ExploreBlockEditModal.tsx` still said `Screen HP-3 Block CRUD Modal (Paqlk)` next to the new `H3 (Paqlk)`. Normalized ONLY these two (node IDs this story already re-labeled); the general out-of-scope carve-out (uhAKd/KNI8F/…) stands. [ScannerSettings.tsx:1, ExploreBlockEditModal.tsx:1]

## Dev Notes

- **ESLint grammar you must satisfy** (`apps/web/src/eslint-rules/implements-pen-node-id.js`):
  - Screen form: `Design ref: ux-design.pen Screen {anything} ({nodeId})` — nodeId is letters/digits only; the `·`-joined multi-ref reads as one line and passes because the regex is `Screen\s+.+\s+\([A-Za-z0-9]+\)` anchored at the END — the LAST parenthesized token must be a bare node ID.
  - Gap form: `Design ref: ux-design.pen — no current screen frame; {reason}` — the em-dash and semicolon are load-bearing.
  - Marker must be a **leading** comment (before the first import).
- **No `.pen` writes.** This story reads the design; it never edits it. Therefore the CLAUDE.md "regenerate screenshots after .pen modification" workflow does NOT trigger. If you find yourself wanting to fix the design, that is a new discovery — Rule 24 it, don't do it here.
- **Do not "fix" `settings/LibraryEditModal.tsx`** — it is already correct (9R-10b Task 5) and is the format precedent.
- **Header format precedent:** `Screen E5-D (hUVYm) · E5-M (P0P82x) · J4-D (sPzZT) · J5-D (alrIw)` (current post-2026-06-05 frame-code naming, `·` joiner). Match it.
- **Why row 12's B2/B5 hits are red herrings:** 匯出中繼資料 in B2-D/B5-D/B2-M/B5-M is the poster-card / detail context-menu ITEM (flow B), not the settings 匯出/匯入 tab UI that `MetadataExport.tsx` renders.
- Rule 7 / Rule 10 / Rule 20 contract stamps: **N/A** — zero API surface touched. Cross-stack split check: FE 4 tasks / BE 0 → no split.

### Project Structure Notes

- All 17 files live under `apps/web/src/components/{settings,health,learning}/` — comment line 1 only.
- One docs file: `_bmad-output/audit/drift-sweep-2026-05.md` (append-only addendum).

### Time-dependent visual coverage

- N/A — no wall-clock-reading components touched (comment-only edits; zero rendering changes).

### Out of Scope (do NOT expand)

- Legacy-but-**correct-ID** headers elsewhere (e.g. `Screen 11 Backup Management Desktop (uhAKd)`, `Screen 1 Library Grid Desktop (KNI8F)`, `Screen AS-1 …`): stale names, right IDs, zero traceability harm. Sweeping them is a separate cosmetic story if anyone ever cares — grep scope here is `6UCtX` citations only, plus the `Screen 10 Settings Desktop` string those same files carry.
- Any `.pen` design change (e.g. "we should draw the missing cache/logs/service-status frames") — that is Sally's call, via the inline-agent workflow, not this story.

### References

- [Source: sprint-status.yaml `bugfix-settings-design-ref-6uctx-sweep` entry — Rule 24 ③ filing with rationale]
- [Source: _bmad-output/implementation-artifacts/9R-UX-auto-generation-toggle-design.md — Discovery Triage; the field verification that 6UCtX is the qBT page]
- [Source: _bmad-output/audit/drift-sweep-2026-05.md lines ~83, ~109–188 — the 19-8 `screen-section-upgraded-to-6UCtX` blanket rows (root cause)]
- [Source: apps/web/src/eslint-rules/implements-pen-node-id.js — accepted marker grammar]
- [Source: scripts/export-pen-screenshots.py `SCREENS` dict — canonical frame-code ↔ node-ID registry]
- [Source: project-context.md Rule 21 (traceability), Rule 24 (discovery triage)]

## Dev Agent Record

### Agent Model Used

claude-fable-5 (Claude Fable 5) — Bob/SM context continuing directly into dev-story per Alexyu's option-1 pick (2026-08-24).

### Debug Log References

- RED baseline: `grep -rln 6UCtX apps/web/src` = 17 files; `grep -rln "Screen 10 Settings Desktop" apps/web/src` = 17 files.
- GREEN: legacy-name grep = 0 hits; `6UCtX` citers = 11 files (see deviation note in Completion Notes).
- Gates: `pnpm nx lint web` 0 errors (117 pre-existing jsx-a11y warnings, retro-11-AI1b batch — 0 introduced); `pnpm run format:check` clean; `pnpm nx test web` green; `pnpm nx test api` green; `tsc --noEmit` (apps/web) 0 errors; no orphaned vitest workers.

### Completion Notes List

- All 17 headers rewritten per the mapping table; `git diff -- apps/web/src` = 17 files / 18 comment lines, zero logic lines (ExploreBlocksSettings has 2 lines: top header + inner doc-comment `HP-5`→`H5-D` reconcile).
- **Rows 15–16 verdicts (Task 1.3):** both `health/QBStatusIndicator.tsx` and `health/ConnectionHistoryPanel.tsx` → design-coverage gap form. Basis: (a) C4-D's full text dump shows only the qBT connection form — no status-indicator or history-panel UI; (b) neither component has a live render site anymore (dashboard-era Story 4.6; referenced only by `routes/test/-gallery.fixtures.tsx` and a prose comment in `HomeBrowseV2.tsx` — kept for visual-baseline gallery); (c) NOT `Component/SidebarFooterStatus (PrmQG)` — that is implemented by `shell/SidebarFooter.tsx`. Recorded in the audit addendum.
- **⚠️ Deviation (Task 4.2 vs AC #5):** Task 4.2 expected only rows 1/2/17 to still cite `6UCtX`, but AC #5's mandated gap-form wording itself cites the shell exemplar `(Screen C4-D, 6UCtX)` — the two were internally inconsistent as written. The AC governs: final `6UCtX` citers = 11 files (2 connection true-refs + 8 gap forms + ApiKeysForm), every citation now correct in meaning (either the actual screen or an explicitly-labeled shell exemplar). No header claims `6UCtX` shows a component it does not show — which is the defect this story exists to remove.
- 🔗 AC Drift: FOUND — story 19-8 (done, frozen) recorded these 17 files as `6UCtX`/`exact` in `_bmad-output/audit/drift-sweep-2026-05.md`; this story supersedes those rows via an append-only addendum (AC #7), original rows retained as the durable 19-8 record. No runtime behavior changed.
- 📎 Contract Stamps: NONE in this story (no `[@contract-vN]` stamps; comment/docs only — no wire contract defined or bumped). Upstream grammar consumed unchanged: confirmed against 19-3 `[@contract-v3]` (`local/implements-pen-node-id` accepted-marker grammar incl. the 19-8 `// Design ref:` forms) — all 17 rewritten headers pass the live ESLint rule; no grammar widening needed.
- 🎭 A11y Pre-Flight: PASS (17 files touched, comment lines only — 0 jsx-a11y warnings introduced by this story; 117 pre-existing project-wide warnings belong to retro-11-AI1b, scope not expanded).
- 🎨 UX Verification: SKIPPED — no UI changes (comment-only; zero rendering diff, zero `.pen` writes, so the CLAUDE.md screenshot-regen workflow is not triggered).
- Rule 23: N/A — no wall-clock-reading component logic touched.
- **CR 2026-08-24 (adversarial self-review, same session — noted as such):** 0 High / 2 Medium / 3 Low. Fixed: M1 SettingsLayout false utility-exemption; L1 same-node-ID label normalize (ScannerSettings, ExploreBlockEditModal). Accepted-as-documented: M2 = the Task 4.2 vs AC #5 contradiction (already recorded above; AC governs). Observations, no action: L2 = `Screen 10 Settings Desktop` string survives in `_bmad-output/audit/*` by design (durable 19-8 record); L3 = the two orphaned health components are already tracked by the ux3 plan (ux3-1-4 Rule-24 acknowledged trade; data reachable via ux3-0-4 sidebar status strip). Final `6UCtX` citers = 12 files (11 + SettingsLayout), every citation correct in meaning. Post-fix gates: lint web 0 errors, format:check green, diff still 100% comment lines. 🔒 Rule 7 Wire Format: N/A (no Go error-code files in scope). 🔒 Rule 20 Contract Bump: N/A (no stamp bumps). 🔒 Rule 25 Mega-line: N/A (project-context.md untouched). Git vs File List discrepancies: 0.

### Discovery Triage

- **Did this story discover any work outside its current scope?**
  - Draft-time candidate resolved: the legacy-era screen NAMES with correct IDs (e.g. `Screen 11 Backup Management Desktop (uhAKd)`, `Screen 1 Library Grid Desktop (KNI8F)`) — **N/A, no entry filed**. Rationale: stale display labels over CORRECT node IDs; traceability (the ID) is intact, ESLint passes, no verification step can be misled. Purely cosmetic — does not meet the Rule 24 "real work" bar. Documented in the audit addendum's out-of-scope note so the assessment is durable.
  - No other out-of-scope work discovered during implementation.

### File List

- apps/web/src/components/settings/QBittorrentForm.tsx
- apps/web/src/components/settings/ConnectionTestResult.tsx
- apps/web/src/components/settings/MediaLibraryManager.tsx
- apps/web/src/components/settings/LibraryCard.tsx
- apps/web/src/components/settings/CacheManagement.tsx
- apps/web/src/components/settings/CacheTypeCard.tsx
- apps/web/src/components/settings/LogsViewer.tsx
- apps/web/src/components/settings/LogEntry.tsx
- apps/web/src/components/settings/LogFilters.tsx
- apps/web/src/components/settings/ServiceStatusDashboard.tsx
- apps/web/src/components/settings/ServiceStatusCard.tsx
- apps/web/src/components/settings/MetadataExport.tsx
- apps/web/src/components/settings/ExploreBlocksSettings.tsx
- apps/web/src/components/learning/LearnedPatternsSettings.tsx
- apps/web/src/components/health/QBStatusIndicator.tsx
- apps/web/src/components/health/ConnectionHistoryPanel.tsx
- apps/web/src/components/settings/ApiKeysForm.tsx
- _bmad-output/audit/drift-sweep-2026-05.md (append-only addendum)
- _bmad-output/implementation-artifacts/sprint-status.yaml
- _bmad-output/implementation-artifacts/bugfix-settings-design-ref-6uctx-sweep.md
- apps/web/src/components/settings/SettingsLayout.tsx (CR fix M1)
- apps/web/src/components/settings/ScannerSettings.tsx (CR fix L1)
- apps/web/src/components/settings/ExploreBlockEditModal.tsx (CR fix L1)

## Change Log

| Date | Change |
|------|--------|
| 2026-08-24 | Tasks 1–2: verified 17-file mapping (rows 15–16 ruled design-gap: no live render site, C4-D shows no such UI, not PrmQG) and applied all header corrections — 18 comment lines, 0 logic lines. |
| 2026-08-24 | Task 3: appended `## Addendum 2026-08 — 6UCtX correction` to `drift-sweep-2026-05.md` (19-8 rows retained; corrected table is the new grep target). |
| 2026-08-24 | CR (adversarial): 0H/2M/3L — fixed SettingsLayout false `<utility>` exemption (it IS the designed settings shell → `C4-D (6UCtX) · C4-M (2H4OM)`) + normalized ScannerSettings/ExploreBlockEditModal labels on node IDs this story re-labeled; M2 accepted-as-documented, L2/L3 observations. Lint+format re-green. Story → done. |
| 2026-08-24 | Task 4: gates green (legacy-name grep 0; lint 0 errors; format clean; web+api suites green; tsc 0). Deviation Task 4.2 vs AC #5 documented — AC governs, 11 files legitimately still cite 6UCtX. |
