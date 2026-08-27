---
target: homepage
total_score: 21
max_score: 40
na_heuristics: 
p0_count: 1
p1_count: 4
timestamp: 2026-08-26T17-20-30Z
slug: apps-web-src-components-homepage
---
Method: dual-agent (A: design review · B: detector + browser evidence). Parent independently re-verified the P0, the light-theme placeholder contrast, and the mobile divider defect.

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|---|---|---|
| 1 | Visibility of System Status | 3 | Band is excellent and honestly degraded; the in-flight breath is verified real. But CLS 0.3237 — the page visibly jumps on load — and no budget context before the money door. |
| 2 | Match System / Real World | 2 | `<html lang="en">` on a 100%-zh-Hant page; `<title>Web</title>` (untouched Nx default); 「整理中」 says *-ing* for a queued state. |
| 3 | User Control and Freedom | 2 | No skip link (`a[href^="#"]` = 0); 18 mandatory poster tab stops between the band and the next control; the degraded notice can't be dismissed. |
| 4 | Consistency and Standards | 2 | WCAG 2.5.3 Label-in-Name mismatch on all four readout cells; `·` does three unrelated jobs within 400px; `--accent-text #e0be72` vs `--warning-text #e8b04b` at 11px. |
| 5 | Error Prevention | 1 | The 產生字幕 door offers no count, no estimate, no budget before a spend flow — and lands on a screen asserting the opposite of the number printed on it. |
| 6 | Recognition Rather Than Recall | 3 | Band is pure recognition; nothing on the page is unlabelled. But the destination never restates 0/18, so only a user holding it in memory can catch the contradiction. |
| 7 | Flexibility and Efficiency | 2 | Real 44px targets and unmissable focus rings, but no skip link, no per-title generate path, no bulk affordance. |
| 8 | Aesthetic and Minimalist Design | 3 | 夜行/日巡 are genuinely disciplined (contrast verified both themes). Undercut by 400px of reserved-then-empty hero and hash gradients being the loudest thing on screen. |
| 9 | Error Recovery | 2 | The TMDb degraded line is a model. The generate empty state gives an actively false diagnosis; the 失敗 poster badge names no cause. |
| 10 | Help and Documentation | 1 | Zero. No tooltip explains what 繁中字幕覆蓋 counts, what 產生字幕 will do, or what it will cost. |
| **Total** | | **21/40** | **Acceptable — but held down by one P0 that is a positioning failure, not a UX one** |

## Design Specificity Verdict

**Grounded — with one contradiction that undoes the grounding.**

This could not be swapped into another media app unchanged. The band's four words — 繁中字幕 / 今天處理 / 需要注意 / 進行中 — are the four questions *this* product's user has after leaving the machine alone, in that order. The motion licence is verifiable rather than decorative: measured `animationName: none` on every element at 0 in-flight, and exactly one `breathe 2.4s` after mocking `in_flight: 3`. That is 固定詞彙 honoured on the time axis, and almost nobody ships it.

Then the door: the page says **0/18**, one click says **「所有影片都有繁中字幕了」**.

**Deterministic scan**: 11 findings, all `design-system-font-size`. Ten are FALSE POSITIVES — 11px is this repo's established micro-label step (52 occurrences / 31 files) and every 11px node measured clears AA (5.92–9.14:1 dark, 6.5–6.88:1 light); the rule's authority is circular because DESIGN.md was generated from this same source. One is a TRUE POSITIVE: `InFlightBadge.tsx:51` renders the mobile 活動 badge digit at **10px**, below the 11px floor prior rounds adopted.

**Visual overlays**: none. There is no injectable `detect.js` in the skill's scripts directory (only `detect.mjs` and the `live/` helpers), so no overlay was produced. Injection itself was confirmed possible (no CSP block) — the artefact simply does not exist. Fallback signal is the CLI scan plus the measurements below.

## Overall Impression

The band is the best thing this product has built. Four cells, three seconds, every cell a door, and stillness that means something. Then the only gold thing on the page opens onto a sentence that contradicts the number beside it. Everything else here is craft debt; that one is the product's stated reason to exist, broken on its own call to action.

## What's Working

1. **The breath is a measurement, not an ornament — and it is enforced.** With `in_flight: 0` there is not one running animation anywhere on the page; mocking the count to 3 starts exactly one. This makes *stillness* informative.
2. **The TMDb degraded line is a model of 降級不失效.** It names the scope (3 blocks), names the cause with both possibilities rather than guessing, and hands over the door — and replaces the section instead of leaving three broken shelves.
3. **Two-theme contrast discipline that actually holds.** Readout band, every text node, composited backgrounds: 5.92–12.85:1 dark, 6.5–13.21:1 light. Zero failures. 日巡 is a re-derived palette, not a tint.

## Priority Issues

### [P0] The 產生字幕 door contradicts the number printed on it

**What**: `readout-coverage` reads 「繁中字幕 · 產生字幕 / 0/18」 and navigates to the generate dialog's empty state: a green check above 「所有影片都有繁中字幕了」.

Verified against the live API, not inferred:
```
GET /subtitles/generation-candidates
→ analyzed: 15, total: 15, candidates: [], skipped_count: 0
```
Library is 15 movies + 3 series = 18. Three quantities, none of which support that sentence: the homepage counts **18** titles with **0** covered; the analysis looked at **15** (series are skipped — ASR is movies-only per PRODUCT.md); it found **0** generatable.

**Why it matters**: This is the failure mode PRODUCT.md names as fatal —「無人值守＝沒人發現它在騙你」. And it is not a seed artifact: because the analysis skips series entirely, a **series-only library** — an ordinary setup — analyses 0, finds 0 candidates, and is told 「所有影片都有繁中字幕了」 while having none. The false claim also wears **green**, which 固定詞彙 reserves for 正在發生.

`ConsentEmptyState.tsx`'s own header states its premise: "every listable candidate already has a zh-Hant subtitle." The condition that renders it is merely "the list is empty" — which also covers "nothing was listable."

**Fix**: (a) the empty state must report what was found, not assert completion — 「這次分析沒有找到可產生的項目」 plus the reason (分析後 0 部符合／尚未分析／來源無法取得), and drop the green; (b) the door must not open onto a different quantity than the one on it — gate the action on the candidate count, or open the dialog pre-filtered to the titles the band counted.

**Suggested command**: `/impeccable clarify`

### [P1] 日巡: poster placeholder initials measure 1.45:1

**What**: The hash gradient is a fixed dark colour that does NOT swap by theme; the initial uses `--text-primary`, which does. Measured in 日巡: **F 1.45 / 1.75**, H 2.07 / 2.99, U 2.18 / 2.96, S 2.39 / 2.81 — every one below the 3:1 large-text floor. In 夜行 the same cards pass (4.29–8.83:1).

**Why it matters**: PRODUCT.md names contrast a hard requirement, with precedent (`--text-disabled` at 3.55:1 was ruled unusable). These are the homepage's largest elements, and they are exactly the un-matched files a new library is full of.

**Fix**: Derive the placeholder from theme tokens instead of a raw hash colour, or keep the letter on `--text-on-scrim` (light in both themes) and darken the light-theme stops.

**Suggested command**: `/impeccable audit`

### [P1] The mobile readout band draws half a divider

**What**: At 390px the band is `grid` / `179px 179px` with `max-md:divide-y`. Tailwind 4 compiles that to `> :not(:last-child){border-bottom}`, so in a 2-column grid: coverage 1px, processed 1px, **attention 1px**, inflight **0px**. The third cell draws a 179px rule *inside the last row* with no counterpart under 進行中 — a half-width line hanging off the band's bottom edge. Both assessments flagged it independently; measured by the parent.

**Why it matters**: The band is the product's signature. On the surface PRODUCT.md calls 同等重要, its 2×2 grid reads as an L of stray lines.

**Fix**: Explicit borders on cells 2/3/4, or `gap-px` over a `--border-subtle` background.

**Suggested command**: `/impeccable layout`

### [P1] CLS 0.3237 — the skeleton reserves 400px for a hero that never arrives

**What**: Measured on a normal load: shifts 0.2139 @127ms, 0.0492 @721ms, 0.0606 @728ms. Cause: `hero-banner-skeleton` reserves 400px at y=180 while loading, but HeroBanner renders **nothing** in the TMDb-degraded state — `home-recently-added` snaps y=652 → y=185 (467px), docHeight 1184 → 1642.

**Why it matters**: Good CLS is ≤0.1; this is 3× that. The user's thumb is over the readout band when the page jumps 467px.

**Fix**: The skeleton must reserve only what the resolved state will occupy — decide the hero's presence before reserving, or give the degraded state something honest to occupy the space (see the P2 below).

**Suggested command**: `/impeccable layout`

### [P1] All four readout cells fail WCAG 2.5.3 Label in Name

**What**: Visible text vs accessible name:

| cell | visible | aria-label |
|---|---|---|
| coverage | 繁中字幕 · 產生字幕 0/18 | 繁中字幕覆蓋 0 / 18 部，前往產生字幕 |
| processed | 今天處理 0 部 | 今天處理**了** 0 部，前往活動中心 |
| attention | **需要注意** 一切正常 | 一切正常，前往活動中心 |
| inflight | 進行中 0 個任務 | 0 個任務進行中，前往活動中心 |

**Why it matters**: `readout-attention` is the worst: its visible label word 需要注意 appears **nowhere** in the accessible name, so voice control cannot activate the cell by what the user can see. The others reorder or insert characters.

**Fix**: Start each accessible name with the visible label verbatim, then append the context.

**Suggested command**: `/impeccable audit`

## Persona Red Flags

**Jordan (first-timer)** — the sharpest failure. Fresh scan, opens `/`, reads `0/18`, `0 部`, **一切正常**, `0 個任務`. The reassurance cell says relax; the coverage cell says nothing works. He clicks the one gold thing — 11px, the smallest text on the page — and gets **「所有影片都有繁中字幕了」** under a green check. No sentence anywhere tells him what happens next, what it costs, or how long it takes.

**Sam (accessibility / keyboard / screen reader)** — credit first: 39 tab stops, no trap, nothing unreachable, every focus indicator ≥5.92:1, nothing unlabelled. Real failures: (1) `<html lang="en">` means every 繁體中文 string is announced with English pronunciation rules; (2) Label-in-Name mismatch on all four cells; (3) 18 poster tab stops with no skip link; (4) the primary CTA is signalled by colour alone (no underline, no glyph) — WCAG 1.4.1.

**Casey (distracted, one-handed, mobile)** — the band's tap targets are right (whole cell, 179×61.5). But the distinction she needs is an 11px hue shift she will never catch at arm's length in a warm palette, the band's own grid draws a broken line under it, and the page jumps 467px while she is reading it.

## Minor Observations

- `InFlightBadge.tsx:51` renders the mobile 活動 badge digit at **10px** — the one true positive from the detector, and it came from the motion pass.
- Three tap targets under 44px: `vido 首頁` 36.6×28, `整理中 · 3` 80.2×24, `前往連線設定` 80.5×20. All clear WCAG 2.5.2's 24×24 (the third is inline text, plausibly exempt) but miss the 44px the rest of the page holds.
- `整理中 · 3` is a row-scoped count that navigates to `/activity` — a global destination — and overflows its own box (scrollWidth 84 / clientWidth 80).
- Card titles clip mid-token with no ellipsis (`scrollWidth 213 / clientWidth 140`), so truncation is indistinguishable from a filename that really ends there.
- 缺字幕 renders in the quietest treatment on the card and appears 15 times. The one thing this product exists to fix is styled as incidental metadata.
- **Regression verified fixed**: the 2026-08-25 shell round recorded the MobileTabBar active label at 3.58:1. It now measures 9.14:1.

## Questions to Consider

1. **Why is 需要注意 allowed to say 「一切正常」 while 繁中字幕 says 0/18?** If coverage is the product's reason to exist, 0% coverage *is* the thing needing attention. Either 需要注意 means "無人值守管線的例外" — then say so in the label — or 「一切正常」 is the page's second untrue statement.
2. **The band decided 0 是資訊. Are three zeros in a four-cell row still information, or noise wearing information's clothes?** A returning user reads the band in three seconds. A first-run user reads it as "nothing here."
3. **When the accent moved from 訊號藍 to gold, who re-checked the 固定詞彙?** Blue and amber were unmistakable. Gold (#e0be72) and amber (#e8b04b) at 11px are not.
4. **The 產生字幕 door exists because the toast destroyed itself in ten seconds. Right diagnosis — but is an 11px suffix a door, or a link that happens to be there?** What the page may actually need is a first-fold statement with a real button, in the 400px the hero currently reserves and never uses.
