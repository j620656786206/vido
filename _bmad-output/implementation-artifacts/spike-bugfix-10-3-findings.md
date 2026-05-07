# Spike Findings — bugfix-10-3 Skeleton Flicker on Load

**Story:** `bugfix-10-3-skeleton-flicker-on-load.md`
**Date:** 2026-05-07
**Methodology:** Playwright-based DOM probe (user-authorized substitute for React DevTools Profiler — see story Halt-and-Decide gate at Task 1.5)
**Probe artifacts:**

- `spike-bugfix-10-3-render-snapshots.spec.ts` — coarse 50-450 ms snapshot poller of testid counts
- `spike-bugfix-10-3-mutation-log.spec.ts` — frame-level MutationObserver log of every tracked testid mount/unmount in the 0–2500 ms window
- `spike-bugfix-10-3-{dev,prod}-snapshots.json` — raw count snapshots for both environments
- `spike-bugfix-10-3-{dev,prod}-mutations.json` — raw mutation events (41 events each)

**Probe environment:**

- Dev: `pnpm nx serve web` on port 4200 (React 18 dev build, `<StrictMode>` active per `apps/web/src/main.tsx:11`)
- Prod: `pnpm nx run web:build` then `vite preview --port 4201` (React 18 production build, StrictMode is a no-op in prod runtime)
- Backend: `cd apps/api && go run ./cmd/api` on port 8080 (only used to satisfy the Playwright `globalSetup` `/setup/status` precondition; the homepage itself uses fully mocked endpoints with deterministic 100 ms delays)

---

## ⚖️ One-line verdict

**Bucket A — `dev-mode-only` artifact (high confidence).** The production preview build does NOT exhibit any skeleton re-mount, skeleton-after-final, or other DOM-level flicker pattern. DOM mutations in dev and prod are identical in count and shape; only the absolute timing differs (Vite dev startup adds ~284 ms before first render).

---

## Per-component render-count table

(Counts come from the `MutationObserver` probe over a 2.5 s observation window after `page.goto('/')` with all 6 endpoints stubbed at 100 ms.)

| testid | DEV mount | DEV unmount | PROD mount | PROD unmount | Δ (DEV-PROD) | Verdict |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| `hero-banner-skeleton` | 1 | 0\* | 1 | 0\* | 0 | normal — skeleton replaced via reconciliation, parent node reused |
| `hero-banner` | 0\* | 0\* | 0\* | 0\* | 0 | normal — same parent reused (see methodology note below) |
| `explore-blocks-loading` | 1 | 1 | 1 | 1 | 0 | clean once-only lifecycle |
| `explore-blocks-list` | 1 | 0 | 1 | 0 | 0 | clean once-only mount |
| `explore-block-b1` | 1 | 0 | 1 | 0 | 0 | clean once-only mount |
| `explore-block-b2` | 1 | 0 | 1 | 0 | 0 | clean once-only mount |
| `explore-block-b3` | 1 | 0 | 1 | 0 | 0 | clean once-only mount |
| `explore-block-skeleton` | **18** | **12** | **18** | **12** | 0 | per-card (3 blocks × 6 PosterCardSkeleton) — 2 blocks unmount their 6 skeletons each as content arrives, 3rd block stays lazy below fold |
| `recent-media-loading` | 1 | 0 | 1 | 0 | 0 | normal — final state is the panel with content; loading testid sits inside it |
| `recent-media-panel` | 1 | 0 | 1 | 0 | 0 | clean once-only mount |
| `download-panel` | 1 | 0 | 1 | 0 | 0 | clean once-only mount |
| `download-panel-loading` | 1 | 0 | 1 | 0 | 0 | normal — loading sits inside panel |

\* The 0 mount/unmount on `hero-banner` and the asymmetric 1m/0u on `hero-banner-skeleton` is a probe limitation, not a real anomaly: when React reconciliation swaps the skeleton subtree for the final-state subtree under the SAME parent fiber, the MutationObserver sees only attribute/child mutations on the parent — no top-level mount/unmount. The coarse snapshot probe confirms the testid swap (skeleton count 1 → 0, final count 0 → 1) cleanly, in both modes (see snapshots JSON, t=350 → t=500 in DEV / t=200 → t=350 in PROD).

**Δ column is 0 across the board.** StrictMode does NOT produce any extra DOM mount/unmount events visible to MutationObserver. (StrictMode double-invokes effects and state-initializers at the React fiber layer — those would show up in React DevTools Profiler but not in committed-DOM observers. See "Methodology limitations" below.)

---

## Skeleton transition timeline (snapshot probe, deterministic 100 ms mocks)

DEV — `hero-banner-skeleton`:

```
t=  50ms:  0     ← homepage not yet committed (Vite dev startup)
t= 100ms:  0
t= 200ms:  0
t= 350ms:  1     ← first commit, skeleton appears
t= 500ms:  0     ← data arrived, final committed
t= 800ms:  0
t=1200ms:  0
t=1800ms:  0
t=netidle: 0
```

PROD — `hero-banner-skeleton`:

```
t=  50ms:  1     ← prod build commits faster
t= 100ms:  1
t= 200ms:  1
t= 350ms:  0     ← data arrived, final committed
t= 500ms:  0
t= 800ms:  0
t=1200ms:  0
t=1800ms:  0
t=netidle: 0
```

Both sequences are **monotonically non-increasing** (the AC #2 NO-FLICKER invariant). No re-appearance after the final state lands. Same shape for `explore-blocks-loading`, `recent-media-loading`. (`download-panel` testid is shared by skeleton + final wrapper, so it stays at 1 throughout — no signal from this testid alone.)

---

## Per-transition findings (per AC #1 baseline requirement)

For each transition observed in the probe, the offending component + render trigger + dev/prod count comparison.

| Transition | Component | File:Line | Dev ms | Prod ms | Re-mount? | Need fix? |
| --- | --- | --- | ---: | ---: | --- | --- |
| skeleton → final hero | `HeroBanner` | `apps/web/src/components/homepage/HeroBanner.tsx:169-195` | 350→500 | 50→350 | **No** | No |
| skeleton → list (outer→inner) | `ExploreBlocksList` | `apps/web/src/components/homepage/ExploreBlocksList.tsx:46-59` | 342→536 | 134→252 | **No** | No (placeholder hands off cleanly under deterministic mocks) |
| per-block skeleton → real cards | `ExploreBlock` × 2 (b1, b2) | `apps/web/src/components/homepage/ExploreBlock.tsx:34-74` | 536→650 | 252→386 | **No** | No |
| b3 lazy block (below-fold) | `ExploreBlock` (b3) | same | n/a (stays skeleton through observation window) | n/a | normal lazy behavior | No |
| recent panel skeleton → content | `RecentMediaPanel` | `apps/web/src/components/dashboard/RecentMediaPanel.tsx:65-85` | not measurable as separate (`recent-media-loading` lives inside `recent-media-panel`) | — | **No** | No |
| download panel | `DownloadPanel` | `apps/web/src/components/dashboard/DownloadPanel.tsx:33-103` | shared testid | — | **No** | No |

Zero transitions exhibit a re-mount, re-appearance after final, or skeleton-after-final pattern in EITHER mode.

---

## Why the user perceives flicker (working hypotheses)

The user's original sprint-status report ("homepage skeletons repeatedly re-mount/flicker during initial load") was almost certainly observed in `pnpm nx serve web` (dev). The most likely contributors, ranked by likelihood under the spike evidence:

1. **(Highest) React 18 StrictMode double-invocation** — `<StrictMode>` at `apps/web/src/main.tsx:11` causes React to intentionally double-invoke component bodies, state initializers (`useState(initialValue)`), and `useEffect`/`useLayoutEffect` setup+cleanup in dev. The COMMITTED DOM still updates only once (which is why MutationObserver shows identical counts in dev and prod), but the fiber-level double-render can cause a visible double-paint of pre-commit components (e.g. `useInViewport`'s `setState(false)` → re-run effect → `setState(true)` cascade) that a careful eye sees as "the skeleton flashed twice". This is by-design and stripped from prod runtime — the user's only fix is to verify in `web:preview` (which this spike just did) and not chase the dev artifact.

2. **(Medium) `useInViewport` `useState(disabled)` initial-state race** — even outside StrictMode, when an above-the-fold lazy block (`disabled=false`) mounts, the initial state is `false` (skeleton shown), then the `IntersectionObserver` callback fires asynchronously and sets it to `true` (real content). That's a TWO-RENDER lifecycle even in prod for blocks that are visible at mount. My probe shows no DOM-level duplicate mount of `explore-block-skeleton` per block — the per-card array unmounts cleanly when data arrives — but the inner `<ExploreBlockSkeleton>` element itself does live for 1 React render before the data path takes over. This is what AC #3 wants to fix. The "flicker" from this is sub-frame and would not be perceptible to most users — but the wasted render is real.

3. **(Low) `ExploreBlocksList` outer → inner placeholder swap** — `ExploreBlocksList.tsx:46-54` renders a `min-h-[200px]` placeholder while the block-list query loads, then `ExploreBlocksListInner` (line 62) mounts when data arrives. My probe shows clean handoff (no measurable layout collapse window under deterministic mocks). Could behave differently under variable real-network latency where the block-list and per-block content queries arrive in different microtasks. AC #4 is a defensive `min-h` wrapper.

4. **(Very low) `HeroBanner` skeleton-before-empty short-circuit** — `HeroBanner.tsx:169-181` returns the skeleton (line 170-178) BEFORE checking `isError || !hasItems` (line 181). Only matters when initial trending data is empty (TMDb degraded, blocked region, etc.). Not exercised by the probe (mocks return data). AC #5 reorders the checks to short-circuit empty before skeleton — defensive only.

5. **(Verified non-issue) `tmdbIds` ownership recompute gate at `ExploreBlocksList.tsx:97-99`** — the existing `anyEnabledInflight` guard works correctly. AC #8 only adds an assertion, no fix needed.

---

## Recommended fix strategy summary

Per the AC #1 Bucket-A path:

- **Task 7.1 path:** add a "Known dev-mode artifacts" paragraph to `project-context.md` documenting the StrictMode double-mount + linking to this spike doc. Skip Tasks 2–6 (mark `~~[skipped — Bucket A]~~`). Story → done.

But there's a defensible alternative the spike data also supports — a **Bucket A+ path** that the user may want to consider:

- **Bucket A + opportunistic AC #3/#4 fix:** while the spike does NOT prove a perceptible prod flicker, the candidate triggers #2 and #3 (useInViewport initial-state race, outer→inner placeholder swap) ARE real wasted-render situations that AC #3/#4 would clean up. The cost is ~30 min of code + ~1 hr of tests. The benefit is removing two well-understood micro-anti-patterns from the codebase with a regression test (AC #6) that protects future contributors. AC #5 (HeroBanner short-circuit reorder) is a 5-line defensive change with negligible risk. AC #7 (E2E proof) becomes a permanent guard.

This is a halt-and-decide call per AC #1 / Task 1.5 — see the dev agent's question to the user.

---

## Methodology limitations (calibration for future similar spikes)

1. **MutationObserver vs React fiber:** The probe sees committed-DOM mutations only. StrictMode-induced re-renders that don't change the committed DOM are invisible. To catch those, would need a second probe with React DevTools `Profiler` API (`<Profiler onRender={...} />`) wrapped around suspect components. Not done here because it requires source modification of `App.tsx`.
2. **Deterministic mocks vs variable real-world latency:** All 6 endpoints respond at exactly 100 ms. Real-world TMDb/local-API latency is variable (50–500 ms typical). Cascading fetches with non-uniform timing can produce micro-flickers a deterministic probe wouldn't see. Mitigation: re-run with `--delay-min 50 --delay-max 500` if Bucket B/C is still suspected.
3. **No real TMDb image loads:** image.tmdb.org is stubbed with a 1×1 PNG. Real backdrop images can cause CLS / re-layout flickers as they arrive — but those are CSS-layout flickers, not React-render flickers, and out-of-scope for AC #2.
4. **Single browser engine (Chromium):** Probe runs only against `chromium`. Behavior in `webkit` (Safari, including iOS) may differ. Not mitigated here because the user's report didn't specify browser.
5. **Above-the-fold viewport assumption:** Default Playwright viewport (1280×720) places block #3 below the fold (lazy). With a much taller viewport, all 3 blocks would be eager — `explore-block-skeleton` lifecycle would differ. The b3 6-skeleton residue at end of observation is normal lazy-load behavior, not a bug.

---

## Files written by the spike (added to story File List)

- `tests/e2e/spike-bugfix-10-3-render-snapshots.spec.ts` — temporary
- `tests/e2e/spike-bugfix-10-3-mutation-log.spec.ts` — temporary
- `_bmad-output/implementation-artifacts/spike-bugfix-10-3-dev-snapshots.json`
- `_bmad-output/implementation-artifacts/spike-bugfix-10-3-prod-snapshots.json`
- `_bmad-output/implementation-artifacts/spike-bugfix-10-3-dev-mutations.json`
- `_bmad-output/implementation-artifacts/spike-bugfix-10-3-prod-mutations.json`
- `_bmad-output/implementation-artifacts/spike-bugfix-10-3-findings.md` — this file

The two spike spec files will be DELETED on Bucket A path (Task 7.1 closeout) since they're not the AC #7 deliverable. They'll be EVOLVED into `tests/e2e/homepage-no-flicker.spec.ts` if the user picks the Bucket A+ path.
