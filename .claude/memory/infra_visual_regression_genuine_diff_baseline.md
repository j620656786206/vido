---
name: infra-visual-regression-genuine-diff-baseline
description: How to update -linux visual baselines when a PR both changes existing baselines AND adds new fixtures (fail-fast trick)
metadata: 
  node_type: memory
  type: reference
  originSessionId: 8daeb841-64e0-4d9a-99ef-dd0fc5896987
---

vido's `Visual Regression / PR` job (verify-only, ubuntu) uses a single mega-test
(`tests/visual/components.visual.spec.ts`) that loops fixtures with HARD
`toHaveScreenshot`. Two failure classes behave differently:

- **Missing baseline** → writes the actual and CONTINUES → one CI run emits ALL
  missing fixtures' `-actual.png` at once (this is the `infra-vr-pr-bootstrap-gap`
  case; 13-1b precedent = commit those ubuntu actuals as `-linux`).
- **Existing baseline pixel-diff** → throws HARD → aborts the whole mega-test at
  the FIRST diffing fixture (array order in `-gallery.fixtures.tsx` decides which),
  so only ONE actual is emitted and later fixtures never render.

**The trap:** a PR that BOTH changes an existing `-linux` baseline (genuine
change, e.g. a relabel) AND adds new `-darwin`-only fixtures will fail-fast on the
diff before the new fixtures render — you never get their actuals.

**The fix (two CI rounds):** `git rm` the stale/diffing `-linux` PNGs so they
become MISSING instead of diffing. Now nothing fail-fasts → one CI run emits ALL
needed actuals (converted-to-missing + genuinely-new) in a single pass. Download
the run's `visual-regression-diffs-pr-<runid>` artifact, map each
`components/{id}/{state}-actual.png` → `components/{id}/{state}-visual-linux.png`,
commit, push → green. Verify a content-rich actual visually before committing
(these become the source of truth). Never generate `-linux` locally (darwin box).

**This is the reliable path for ANY genuine `-linux` diff, incl. a SINGLE
one-fixture pixel change** (confirmed 2026-07-07, PR #151, story 13-3b: a
`text-[13px]`→`text-xs` shift on `request-row/downloading` alone). NOTE: the
`/ship` skill claims the `Visual Regression` workflow "auto-opens a
`chore(visual): bootstrap …` PR" — that did NOT fire here (no bootstrap PR ever
appeared; `gh pr list --search "bootstrap linux baselines"` empty). Do not wait
for it; do the `git rm` → download-actual → commit-as-`-linux` cycle yourself.
Also regenerate the matching `-darwin` locally (`npx playwright test
--project=visual --update-snapshots --grep @story-19-4`) but stage ONLY the truly
changed fixture — a full update run re-renders ~27 darwin PNGs with byte noise;
`git checkout -- tests/visual` the rest.

Related: `-actual.png` base token + project `visual` + platform → baseline
`{token}-visual-{darwin|linux}.png`. See [[project-gh-account-for-prs]] — the gh
active account drifts to `alexyu-tvbs` and must be switched to `j620656786206`
before EVERY PR create/merge (it re-drifts between calls).
