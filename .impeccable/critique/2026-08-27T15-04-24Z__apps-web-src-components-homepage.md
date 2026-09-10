---
target: homepage
total_score: 26
max_score: 40
na_heuristics: 
p0_count: 0
p1_count: 1
timestamp: 2026-08-27T15-04-24Z
slug: apps-web-src-components-homepage
---
Method: dual-agent (A: /root/design_review · B: /root/detector_evidence)

## Design Health Score

| # | Heuristic | Score | Key Issue |
|---|---|---:|---|
| 1 | Visibility of System Status | 2 | Home summary error removes the entire readout band. |
| 2 | Match System / Real World | 3 | Readout vocabulary is clear; TMDb remains unexplained. |
| 3 | User Control and Freedom | 3 | Direct readout links and manual hero; no summary recovery. |
| 4 | Consistency and Standards | 4 | Tokens, touch targets, and shelf grammar are consistent. |
| 5 | Error Prevention | 3 | TMDb degrades honestly; primary status has no fallback. |
| 6 | Recognition Rather Than Recall | 3 | Labels help; desktop shelf traversal is hidden until hover. |
| 7 | Flexibility and Efficiency | 2 | No direct status retry or power-user fast path. |
| 8 | Aesthetic and Minimalist Design | 3 | Focused happy state; degraded state is visually hollow. |
| 9 | Error Recovery | 2 | Recent-items error gives only a generic retry. |
| 10 | Help and Documentation | 1 | No contextual explanation of readouts or subtitle states. |
| **Total** | | **26/40** | **Acceptable; significant improvements needed** |

## Design Specificity Verdict

The normal state is strongly authored for Vido: the unattended-subtitle-worker readout, own-library-first ordering, and manual-only hero cannot be transplanted unchanged into a generic media app. The degraded state loses that authority: `HomeReadoutBand` returns `null` for an error/no data, leaving a sparse shell that gives no route back to trust.

The detector found two advisory `design-system-font-size` findings at `HomeReadoutBand.tsx:127,189`, both the deliberate `text-[11px]` micro-label. They conflict with the stale 12px `DESIGN.md` label token, not a functional flaw; retain or document the intentional exception. Browser overlay evidence is unavailable because this runtime reported `No browser is available`.

## What's Working

1. The four-cell readout treats unavailable values honestly and keeps status colours semantically disciplined.
2. `HomeBrowseV2` presents own media before external discovery, matching the returning NAS user's job.
3. The hero is manual, keyboard-aware, and its motion responds to user action rather than inventing activity.

## Priority Issues

### [P1] The trust surface disappears during its own failure

**What:** `HomeReadoutBand` returns `null` on error or missing data; the hero also disappears with unavailable recent data.

**Why it matters:** The homepage is most needed when the background worker or NAS cannot answer, yet it becomes a mostly blank admin shell.

**Fix:** Keep the band mounted with a compact unavailable state, source/timestamp, retry, and a recovery destination.

**Suggested command:** `/impeccable harden`

### [P2] Hero navigation repeats the same decision

**What:** The full hero title link and `查看詳情` both reach the same detail route; the same recent media is immediately repeated below.

**Why it matters:** The first fold asks for a duplicate browsing choice instead of advancing the user's subtitle task.

**Fix:** Retain the hero as context, but use one clear detail target and reserve the second action for a subtitle-specific state when actionable.

**Suggested command:** `/impeccable distill`

### [P2] Recovery language is uneven

**What:** Recent items says only `無法載入，請稍後再試`; the TMDb notice names its scope and provides a settings door.

**Why it matters:** NAS users need to distinguish library/mount failure from an external metadata failure.

**Fix:** Give the recent failure a source-aware reason and the relevant recovery destination.

**Suggested command:** `/impeccable clarify`

### [P2] Shelf traversal is concealed on desktop

**What:** Both shelf chevrons begin at `opacity-0` and appear only on hover or focus.

**Why it matters:** Overflow is easy to miss, so users may never discover later titles.

**Fix:** Show a quiet right-edge continuation cue whenever the row overflows; retain the stronger hover treatment.

**Suggested command:** `/impeccable adapt`

### [P3] The visible page identity evaporates in degraded state

**What:** The page H1 is screen-reader-only; when the band and hero disappear, `最近新增` becomes the apparent page identity.

**Why it matters:** Returning users lose the homepage's operational context.

**Fix:** Add a restrained visible readout/page context label that remains useful even when data is unavailable.

**Suggested command:** `/impeccable clarify`

## Persona Red Flags

- **Alex (power user):** A home-summary failure has no retry, diagnostic, or direct route from the missing band; the hero has two same-destination targets.
- **Jordan (first-timer):** `TMDb 未設定或無法連線` exposes an implementation dependency before explaining its user consequence; recent-data failure gives no diagnosis.
- **Sam (accessibility-dependent):** A summary error removes the labelled readout group; the hero image repeats its adjacent title through `alt={item.title}`.

## Minor Observations

- The 200px explore-loading reserve can read as an unexplained blank void.
- Make the decorative hero image `alt=""` when its adjacent link already announces the title.
- Existing exported Pencil homepage screenshots use the older blue visual world, while current code uses Nightwalk/Daywalk; inspect a happy live-data state before final visual sign-off.
