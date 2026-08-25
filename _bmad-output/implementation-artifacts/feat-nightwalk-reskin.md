# Feat: 夜行 — the wuxia reskin (colour only)

Status: ready-for-review

> Alexyu chose 夜行 over signal blue (2026-08-25) after the side-by-side
> proposal built from his nine reference images. Prerequisite honoured: the
> token debt is fully paid (#291/#292/#294/#295/#298 + baseline chain through
> #302), so this reskin is a styles.css token block edit — not a 61-file hunt.

## Scope: colour ONLY

In: every colour token in `:root`. Out, each a separate later decision:
- Type faces (明體 headings) — typography pass
- Radii (8px → 1px) — 123 call sites, its own debt story
- The white-on-fill literals (`text-white` on semantic fills) — already filed

## The world

Ink-green grounds (石綠墨), 宣紙 warm-white text, GOLD as the one「你在這裡」
colour, cinnabar reserved for faults and destruction, jade for success —
because the ground is green now, the old signal-green cannot carry "success"
against it.

| token | signal blue | 夜行 |
|---|---|---|
| bg-primary/secondary/tertiary | #1b2336/#24304a/#2e3b56 | **#0c1512/#132320/#1b302b** |
| text-primary | #f2f2f2 | **#eae4d6 宣紙白** |
| accent (primary/hover/pressed/text) | blues | **金 #c9a24b/#e0be72/#a8853c/#e0be72** |
| error (+text/pressed) | reds | **朱砂 #c0392b/#e08a76/#9c3a2b** |
| success (+text) | greens | **青玉 #6fbfa8/#8fd3be** |
| warning (+text) | ambers | **赭 #d4763f/#e8b04b** |
| info (+text) | cyan | unchanged (青 already fits) |
| text-on-accent | white | **墨 #14161a** (dark on gold, 7.55:1) |
| text-muted | #a0aabe | **#8fa096** (5.08 worst) |
| text-disabled | #6e7891 | **#5e6e66** (still intentionally sub-AA) |
| focus-ring | blue | gold |
| all 8-digit tints | — | follow their bases |

## The gate as the designer's instrument

Every value was measured BEFORE being written (worst-case across three plain
surfaces AND the family's own tint over --bg-tertiary). The contrast gate —
built across the four debt slices — passed 30/30 on the first run. First
candidate for --text-muted (#7c8a83) failed at 3.87 pre-write and never
touched the codebase; #8fa096 shipped instead.

## Acceptance Criteria

1. Zero non-token colour changes: the diff touches `:root` values, DESIGN.md's
   staleness banner, this story, and baselines. Nothing component-side.
2. styles-contrast gate green (30/30) — plain and tinted surfaces.
3. Live verification against the production build: key readings re-measured in
   the browser (active nav, error banner text, footer readouts) ≥4.5:1.
4. Full unit suite green (16 pre-existing eslint-rules env failures excepted).
5. Visual baselines: ALL darwin regenerated (everything churns by design);
   ALL -linux deleted for one final CI bootstrap. This is the known,
   priced-in cost stated in the proposal.
6. DESIGN.md carries a staleness banner pointing at styles.css as the living
   authority; a full `/impeccable document` rerun is the follow-up once the
   skin settles — not smuggled into this PR.
