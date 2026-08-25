# Refactor: pay the token debt — warning family (amber + yellow + orange)

Status: ready-for-review

> Slice 3 of `disc-2026-08-settings-token-bypass-retrofit`. 63 literals across
> three hues in 21 files → zero. Third slice, third confirmation of the same
> defect class — and this time the GATE itself had the bug.

## The token, as usual

`--warning` (#f59e0b) used as TEXT measures **4.31:1 on its own tint over
--bg-tertiary** — sub-AA, same class as `--error-text` (slice 1) and
`--accent-text` (slice 2). New `--warning-text: #fbbf24` = the amber-400 that
**10 call sites were already hardcoding**; worst case 5.54:1. Also
`--warning-pressed: #d97706`, completing the pressed trio.

## The gate had a blind spot, and falsification caught it

Setting `--warning-text` to the failing #f59e0b did NOT turn the gate red. The
gate only tested plain surfaces — but semantic text usually sits on its own
tint (an error message inside an error banner), and #f59e0b passes every plain
surface while failing the tinted one. The gate now composites each family tint
over --bg-tertiary and tests all four `*-text` tokens against it. Re-falsified:
the old value now fails exactly one test, the new tinted-surface case.

## Mapping

| literal group | → |
|---|---|
| `text-{amber,yellow,orange}-{200,300,400}` | `--warning-text` |
| `text-amber-500` (icon-solids) | `--warning` |
| `bg-*-{800,900}[/x]`, `bg-*-{400,500}/x` | `--warning-tint` |
| `bg-{yellow,orange}-400` (dots) | `--warning` |
| `bg-amber-{500,600}` + `hover:bg-amber-500` (buttons) | `--warning` + `hover:--warning-pressed` |
| `border-*` (alpha preserved) | `--warning` |

One deliberate normalisation: the two warning buttons (RestoreConfirmDialog,
LearnPatternPrompt) rested on amber-600 and LIGHTENED on hover — backwards from
the accent/error convention where hover darkens. They now rest on `--warning`
and darken to `--warning-pressed`. Their white labels were sub-AA before and
after; that belongs to the standing white-on-fill question (same as
`--text-on-accent`), filed, not smuggled into this slice.

## Verification

- Gate: 26 cases (now including 4 tinted-surface ones), falsified.
- Full suite green except the 16 pre-existing `src/eslint-rules` failures.
- Visual churn: 1 darwin baseline regenerated (`degradation-service-health-banner`
  — the yellow-900/20 ground is now `--warning-tint`); matching `-linux` deleted
  for CI bootstrap.

## Remaining

green 29 · emerald 25 · purple 6 → then the exemption-free lint rule + border
removal on tinted surfaces.
