# Fix: homepage critique R1 — the invisible numbers learn to speak, the carousel learns to stop

Status: ready-for-review

> Closes all five priority issues from the first homepage critique (20/40,
> dual-agent 2026-08-25). Alexyu ruling:「全部五個」; the homepage-identity
> question is filed as `disc-2026-08-home-identity-hero`, not decided here.

## P0#1 — the rating number was invisible (1.1:1)

`PosterCardV2` put `--text-on-accent` (ink cut for GOLD fills) on the 70%
black scrim — 1.07–1.20:1, confirmed independently by both assessors. Now
宣紙白 `--text-primary`: **live-measured 15.67:1**. The fallback tile's
initial letter had the same ink over a hash-derived gradient (2.4:1 worst
case); `filenameToGradient` lightness is now clamped dark (26%/32%, was
35%/45%) so light text clears 3:1 on every hue — and the tiles finally sit
inside the 夜行 grounds instead of signal-era brights.

## P0#2 — poster badges wore sub-AA colours on arbitrary artwork

`libraryStatus.ts` TINT map used raw `--success/--warning/--info` as text
(styles.css itself documents 4.31:1 worst case) and the 12% tint composited
over ANY poster art (1.58:1 on a light hash tile). Text now wears the
gate-verified `*-text` variants, and the poster badge gets an opaque
`--bg-secondary` underlay so the composite is deterministic. Live-measured
整理中: **7.17:1**.

## P1#1 — degradation stops lying about configured blocks

Explore blocks the user configured used to skeleton-then-vanish on content
500 —「不存在＝你沒要求」. Now one quiet page-level amber line
(`explore-degraded-notice`):「N 個探索區塊的內容目前無法載入（TMDb 未設定或
無法連線）＋ 前往連線設定」— the 固定詞彙 amber scenario rendered honestly.
Live-verified on the seeded env (3 blocks accounted for).

## P1#2 — carousel accessibility triple

- **Pause control** (WCAG 2.2.2): explicit pause/play button — hover-pause
  was the ONLY stop mechanism and touch devices never got one. User pause
  survives mouse-leave (spec'd).
- **44px dots**: the 8px dot is now a decorative span inside a 44px-tall
  button.
- **ARIA structure**: the slide was `role="link"` WRAPPING a button and a
  Link (forbidden). Container is now a plain pointer-convenience surface;
  the accessible path is the real title `<Link>` + 查看詳情 + 觀看預告片.

## P2 — one card language, one visual world

- Old `PosterCard` aligned with V2's recipe: 12px radius, rating bottom-LEFT
  in scrim + mono + 宣紙白 (was `--warning`-coloured bottom-right), 2-line
  CJK title grid, `text-white` hardcodes → tokens, 🎬 emoji fallback →
  lucide `<Film>`.
- Type chip (`電影/影集`) prop-gated `showTypeBadge`; single-type homepage
  rows pass false (was ×8 repeated noise under a heading that already says
  熱門影集).
- HeroBanner speaks 夜行: grounds/gradient off raw black onto
  `--bg-primary`, CTAs off hardcoded white onto solid gold accent
  (pressable = solid accent, the grammar), white focus ring → `--focus-ring`,
  title capped at the 36px Display ceiling (was 48px), star stops wearing
  `--warning` as decoration.

## Verification

- Unit: homepage + media + library + utils — 818/818. Hero spec reshaped
  (slide role removed; new red→green pause-button test).
- Live probe on rebuilt seeded env: the three numbers above + notice render.
- Visual churn exactly the touched surfaces (hero, explore ×2, both card
  systems, 3 grids): darwin regenerated, 22 `-linux` deleted for bootstrap.
