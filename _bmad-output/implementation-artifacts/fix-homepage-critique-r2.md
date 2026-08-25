# Fix: homepage critique R2 — the last contrast nest, the honest carousel, the naturalized transplant

Status: ready-for-review

> Critique R2 scored 25/40 (↑20). Alexyu rulings:「全部收」+ two vocabulary
> rulings recorded in DESIGN.md / sprint-status. All five R1 fixes were
> independently re-verified by both assessors before this batch.

## P0 — the LAST nest of the token-debt class

`AvailabilityBadge` 已有 wore `text-white` on the jade fill — measured
**2.17:1**, independently re-verified. Only visible in hover/owned states,
which is why four token-debt waves and two critiques missed it. Fixed with
`--text-on-accent` (8.36:1); same sweep took PosterCard's 新增 badge, the
metadataSource badge (opaque fill + ink), and RequestButton's raw
`--success`/`--info` texts → `*-text`.

## P1 ×3

- **Keyboard focus-pause parity**: rotation flipped the focused slide to
  `inert`, throwing focus to `<body>` (measured 8.6s). `onFocusCapture` /
  `onBlurCapture` now pause exactly like hover. Spec falsified (handlers
  removed → red).
- **⚖️ Vocabulary ruling — pending = QUEUED = amber**: the chip and the
  poster badge dressed the same items in green and amber. Ruled amber; the
  chip now matches the badge exactly（整理中, warning tint/text）. Supersedes
  R1's brief green — recorded in the code comment so it is not re-litigated.
- **The lying ▶ overlay removed**: a big play affordance promised playback
  the product does not have. The card's hover scale + title tint already say
  「可以點」without claiming「可以播」.

## P2 — naturalizing the Epic-10 transplant

- Hero content adopts the sibling gutter recipe (`mx-auto max-w-7xl px-4
  sm:px-6`) — left edge 288→**264**, all three sections share one line
  (live-measured).
- 最近新增 row gains the explore rows' chevron + edge-scrim grammar (one
  page, one scroll language; the clipped first card finally has an
  affordance).
- 查看更多 removed until Epic 11's filters let it keep its promise.
- Trailer empty state gets a 前往詳情頁 door + `--overlay-scrim` token
  (was a dead-end box on hardcoded `bg-black/85`).
- Section headings unified to Title (18px/600) — 20px was off-scale.
- **⚖️ Pill ruling**: poster-overlay micro-elements may wear pills
  (amendment written into DESIGN.md Shapes); hero CTA BUTTONS return to 8px.
- Quickies: CW slot's border-on-tint dropped (色調優先); `sr-only` h1 首頁
  (outline began at H2).

## Filed, not fixed

`disc-2026-08-home-hero-fails-silently`, `disc-2026-08-home-keyboard-cost`
(pairs with the rails skip-link item), `disc-2026-08-home-mobile-meta-parity`.

## Verification

- Suites: homepage/media/library/requests 753/753; full lint 0 errors; tsc
  clean. Focus-pause spec falsified live.
- Live on rebuilt seeded env: left edges 264/264/264, chip amber 整理中 · 3,
  h1 present, chevrons render, see-more gone.
- Visual churn exactly the 7 touched surfaces; darwin regenerated, 15
  `-linux` deleted for the incremental bootstrap.
