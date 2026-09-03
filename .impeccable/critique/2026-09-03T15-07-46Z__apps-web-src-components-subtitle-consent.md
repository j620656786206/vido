---
target: F15 產生字幕 consent dialog (apps/web/src/components/subtitle/consent)
total_score: 20
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 4
timestamp: 2026-09-03T15-07-46Z
slug: apps-web-src-components-subtitle-consent
---
Method: dual-agent (A: design review sub-agent · B: detector sub-agent; both relaunched once after a 529 overload)

# Critique — F15「產生字幕」cost-consent dialog (`apps/web/src/components/subtitle/consent`)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|---|---|---|
| 1 | Visibility of System Status | 3 | F14 progress / live region / spinner all present; the budget "cut line" inside the list is invisible |
| 2 | Match System / Real World | 2 | Titles are release filenames; `runtimeKnown=false` REPLACES the route explanation with「片長未知」(`CandidateListPanel.tsx:58-61`) |
| 3 | User Control and Freedom | 2 | Close + reopen resets selection (`GenerationConsentView.tsx:129-132` re-seeds); only escape from 2399 rows is 清除選取 |
| 4 | Consistency and Standards | 3 | Chips are a VIEW filter, 全選 acts on the WHOLE list, yet both sit at the same visual level |
| 5 | Error Prevention | 2 | ASR never pre-selected, budget validation, F16 confirm are good; but filter→全選 selects 2399 incl. hidden rows (`handleToggleAll` :277-281) |
| 6 | Recognition Rather Than Recall | 1 | 38×54 empty grey square per row (:86-89), filename titles, no search — zero recognition cues across 2399 rows |
| 7 | Flexibility and Efficiency | 2 | 整劇/整季 group toggles are a real accelerator; no search, sort, shift-range, keyboard |
| 8 | Aesthetic and Minimalist Design | 2 | 2399 dead poster placeholders; same amount restated 3×; 8 controls above the list |
| 9 | Error Recovery | 2 | `startError` renders at the BOTTOM of the scroll region (:402-406) — invisible when 開始 fails |
| 10 | Help and Documentation | 1 | 抽取 / 語音辨識 never explained; which key gets charged never shown; no settings link |
| **Total** | | **20/40** | **Acceptable (lower edge)** |

## Design Specificity Verdict

**LLM assessment:** Authored for this product — but only for a 142-row world. The vocabulary (抽取／語音辨識), verbatim `usd()`, soft-ceiling copy (預計／約, never「絕不超過」), amber = "you asked, it won't happen" are unmistakably Vido. The ROW itself (checkbox–thumb–title–badge–price) is a generic cart row, and the `.pen` was drawn at「候選 142 部」. PRODUCT.md records the real library as 55 movies + 2406 episodes; the design made no decision for 17× that scale (no search, no sort, movies flat).

**Deterministic scan:** exit 2, 2 findings, both real: `design-system-font-size` — `text-[10px]` on the「僅翻譯費」and「付費」chip markers (`CandidateListPanel.tsx:269`, `:274`). DESIGN.md's ramp has no 10px step and states the 12px floor for any text a user reads. No false positives. Nothing else in the directory flagged.

**Visual overlays:** not available — no DOM-mutation browser tool in this session and the screen needs the live NAS library to render. Fallback signal = the owner's production screenshot.

**Where A and B agree:** A's minor observation (10px/11px used on content, not chrome) is exactly what B flagged. B caught nothing A missed; A's structural findings are beyond the detector's reach.

## Overall Impression

The money logic is honest and well-built; the LIST it sits on is unusable at real scale. Every row has lost its identity (no poster, no runtime, filename titles), so the user is asked to consent to $13.92 for 696 things they cannot recognise. The single biggest opportunity: give rows back their identity, then make 2399 of them navigable.

## What's Working

1. **三處金額同源 + 逐分呈現** (`consentSelection.ts:61-100`) — summary, footer and confirm can never disagree; `≈` honestly marks estimated rows.
2. **Default = cheapest set, ASR never pre-selected** (:27-29) + WYSIWYG budget —「花錢的事先問」is enforced in the data flow, not just the copy.
3. **Empty state refuses to lie** (`ConsentEmptyState.tsx`) — the green tick has to be earned.

## Priority Issues

**[P1] Rows have lost their identity (poster / runtime / title)**
- Why it matters: consent requires recognition. 2399 grey squares + `[bitsearch.to] Predator…` + 100%「片長未知」means the user cannot tell what they are paying for.
- Root cause (verified): backend `GenerationCandidate` has no poster field (`generation_candidates.go:93-118`); the estimator reads only TMDb `runtime` (`:689`) while ffprobe's `DurationSeconds` (`ffprobe_service.go:33`) is probed at scan (`enrichment_service.go:470 applyFFprobeTechInfo`) but never persisted or used; unmatched rows carry the raw filename as `Title`.
- Fix: BE — add `poster_path` (episodes inherit the series poster) to the candidate envelope; persist `duration_seconds` in `applyFFprobeTechInfo` and estimate from `coalesce(duration, tmdb runtime, 45)`; unmatched rows send a cleaned parsed title +「未匹配」flag. FE — `routeSubtitle` shows route AND「≈ 45 分」side by side instead of replacing; poster via `getImageUrl(path,'w92')` (`lib/image.ts:19`).
- Suggested command: `/impeccable clarify` (FE copy/labels) — backend half is a story.

**[P1] 2399 rows are not operable**
- Why: no search, no sort, no virtualization, movies flat. Mobile must complete the task (PRODUCT.md); today the only paths are "accept default" or "give up".
- Fix: sticky search box (title + filename, debounced) above the list; sort (route / cost / title / 未匹配 first); `@tanstack/react-virtual` (already a dependency); collapse series groups by default with count + subtotal in the header; movies grouped 未匹配／已匹配.
- Suggested command: `/impeccable optimize` + `/impeccable distill`

**[P1] Filter + 全選 = money trap**
- Why: chips filter the VIEW, 全選 selects the WHOLE list (:45-47 admits it). Filter to 需語音辨識, tick 全選 → expected ~1700 ASR, got 2399 incl. hidden rows; total jumps.
- Fix: 全選 acts on the visible set and reads「選取顯示的 N 部」; group header pills show route composition ALWAYS, not only after selection (:191-199).
- Suggested command: `/impeccable harden`

**[P1] Start failure is buried under the list**
- Why: `startError` is the last child of the scroll region (:402-406) — a 502 on 開始 is unseen.
- Fix: move it into the sticky footer area with the over-budget banner.
- Suggested command: `/impeccable harden`

**[P2] Cut line invisible, false precision, sub-floor type**
- Why:「約 251 部後暫停」cannot be mapped to the list; every row is `≈` but the total `$13.92` is not; 10px badges violate the documented floor.
- Fix: draw a「上限後暫停」divider before row 252 and dim rows after it; when all rows are estimates show `≈` (or a range) on the total; badges → 12px.
- Suggested command: `/impeccable clarify` + `/impeccable polish`

## Persona Red Flags

- **Alex (Power User):** no search, no shift-range, no keyboard accelerators; "just do Predator" = scroll 2399 rows.
- **Riley (Stress Tester):** filter+全選 trap; empty `preselectedIds` intersection silently falls back to default (:132); close/reopen loses selection.
- **Casey (Distracted Mobile):** 85vh sheet holding 2399 rows, drag handle decorative; at 375px the title has ~150px →「[bitsearch.to] Pre…」; returning after an interruption starts from the top.
- **The NAS owner who just pasted an Anthropic key (project persona):** never told WHICH key is charged (`selfHostedAsr` is in the summary, not the UI); 僅翻譯費 assumes they know translation costs; 抽取／語音辨識 unexplained; the first thing they see is an amber over-budget banner.

## Minor Observations

- Green (僅翻譯費) / amber (付費) used as cost categories — tension with the fixed vocabulary (green = happening now); the `.pen` does the same.
- Budget input `w-16` cannot hold `$100.00`.
- Mobile rows keep the route badge the design removed;「未知影集」sinks to the bottom with no explanation.
- Dialog unmounts on start with no "where to look next".

## Questions to Consider

1. The product principle is「移除操作者」— this screen turns the user into an operator of 2399 rows. Should consent be a POLICY (「每次 $5，先做可抽取」) with the list as an audit view?
2. When 100% of runtimes are the 45-minute guess, is `$13.92` an estimate or a fabrication? Is the honest version a range?
3.「花錢的事先問」asked HOW MUCH — why does it never ask WHOSE money?
