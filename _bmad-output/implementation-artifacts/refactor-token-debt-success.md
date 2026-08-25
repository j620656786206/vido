# Refactor: pay the token debt — success + purple (the last colour slice)

Status: ready-for-review

> Slice 4 of `disc-2026-08-settings-token-bypass-retrofit`. 60 green/emerald/
> purple literals in 27 files → zero. **This closes the colour migration**: all
> 231 literals across all families are now tokens. What remains is the lint
> rule and the tinted-surface border removal.

## Fourth slice, fourth confirmation

`--success` (#22c55e) as TEXT: **4.04:1** on its own tint over `--bg-tertiary`.
New `--success-text: #4ade80` (green-400) — worst case 5.28:1, in the gate,
falsified.

Hue call worth recording: the most-hardcoded literal was `text-emerald-400`
(10×), which PASSES AA (4.78) — but emerald is a different hue from the
`--success` green. The token keeps the family hue (green-400) rather than
canonising the drift; the emerald call sites fold back into the green family.
That IS a visible shift on downloads/notifications surfaces, and it is the
point: one green means "success", not two.

## Purple folded into info

Purple survived in exactly two production sites: StatusIcon's 檢查中 text and
MetadataSourceBadge's AI ground. TechBadge's audio-purple already went to the
info family in the blue slice; these follow. 檢查中 (a transient in-progress
state) and AI 解析 are informational, not success/warning/error — `--info-text`
/ `--info-tint` are the right slots, and purple leaves the palette entirely.

## Verification

- Gate: 30 cases (5 *-text tokens × plain+tinted), falsified.
- Full suite green except the 16 pre-existing `src/eslint-rules` failures.
- Visual churn: 1 darwin baseline (`settings-connection-test-result` — its
  green-900/30 ground is now `--success-tint`, its text `--success-text`).
  `-linux` deleted for CI bootstrap. Note its bright `--success` border is the
  tinted-surface-border violation, deliberately left for the final pass so the
  red/green variants change together.

## Remaining (the endgame)

1. `no-hardcoded-palette` lint rule — now exemption-free by construction.
2. Border removal on tinted surfaces (the 色調優先 violation), all families at
   once so no component ships half-converted.
