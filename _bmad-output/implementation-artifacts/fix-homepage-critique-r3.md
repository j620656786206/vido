# Fix: homepage critique R3 — the badge learns its own vocabulary, the fake door closes

Status: ready-for-review

> R3 scored 27/40 (20→25→27). Rulings:「全收」;「CW 板 Epic 17 前不渲染」;
> two questions filed (explore-owned-sea, hero-autoplay-alien).

## P1#1 — AvailabilityBadge remake (three diseases, one element)

已有 wore SOLID running-green — but ownership is a static FACT, and 固定詞彙
reserves green for happening-now; it was the page's most frequent status
colour (12×). It now wears the neutral scrim pill (a classification, like
the type badge). 已請求 is genuine amber grammar → warning tint over an
opaque backing (the V2 recipe). The whole badge class moves 10px→11px
(detector floor), incl. PosterCard's 新增 (also de-greened — same ruling)
and metadataSource. And the `role="status" aria-live` per badge is gone —
a resolving grid used to fire 12 simultaneous「已有」announcements at SR
users.

## P1#2 — ⚖️ the fake door closes

`ContinueWatchingSlot` is unmounted: 「連接 Plex / Jellyfin 後顯示」promised
a connection flow that exists nowhere in the app.不出現＝你沒要求 is the
honest state until Epic 17; the component survives for that day. First fold
now opens directly on the user's own shelf.

## P2 — degraded theater + hero floors

- TMDb-backed queries (trending, explore content ×2 paths) go `retry:false`:
  on a LAN the realistic failure is unconfigured/down TMDb, which a retry
  never fixes — skeleton-then-collapse shrinks from ~4s to ~1 round trip.
- Hero dots sit on a scrim pill (bare dots measured 1.87:1 on arbitrary
  stills; Shapes amendment makes the pill lawful); inactive dots /50→/70;
  meta row --text-secondary→--text-primary (3.20:1 on white stills).

## P3 + minors

Chip touch target ≥44px (after-inset, visual pill unchanged); TrailerModal
close 32→44px; rotation cross-fade no longer double-exposes CJK titles
(outgoing 300ms < incoming 700ms); hero 片名 h2→p (a movie name was
masquerading as a section heading); fallback initial skips leading symbols
(「[FanSub] 未知電影」rendered a giant「[」); PosterCard meta line joins V2's
mono (比較才用等寬).

## Verification

- Units 733/733 across homepage/library/media (+ the AvailabilityBadge spec
  rewritten around the remake); lint 0 errors; tsc clean.
- e2e swept THIS time: availability-badges + poster-card-hover + hero-banner
  + homepage-layout run locally — 36/36 on chromium.
- Visual churn exactly the touched surfaces (8); darwin regenerated, 15
  `-linux` deleted for the incremental bootstrap.
