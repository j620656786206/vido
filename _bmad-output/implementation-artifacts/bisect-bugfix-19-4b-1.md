# Bisect Verdict — bugfix-19-4b-1: `/test/gallery` Max-Update-Depth Warnings

**Date:** 2026-05-18 · **Probe by:** Amelia /dev-story · **Story:** [bugfix-19-4b-1](./bugfix-19-4b-1-gallery-max-update-depth-warnings.md)

## Verdict: **Bucket A — Single Offender**

The offender is the fixture **`parse-floating-parse-progress-card`**, which renders `FloatingParseProgressCard` → consumes `useParseProgress(taskId, { ...inline options... })`. The loop lives in **`apps/web/src/components/parse/useParseProgress.ts`** (lines 225 / 240 in Vite's dev-transformed module — source-level lines 314-316 + 344-354, see Root Cause below).

## Evidence

### Phase A — multi-fixture browse (`/test/gallery`, no filter)

- **Dev (port 4200, StrictMode on): 265 warnings** in 3 s settle.
- Captured JS stacks (`new Error().stack` via console.error wrapper) — every stack with a non-noise frame pointed to **`/src/components/parse/useParseProgress.ts:225` and `:240`**.
- ZERO captured frames pointed to Sally's narrative-suspect files (`ExploreBlocksList.tsx`, `ExploreBlock.tsx`, `RecentMediaPanel.tsx`, `LibraryGrid.tsx`).

### Phase B — per-fixture isolation (targeted: 1 suspect + 4 Sally-prime + 1 control)

| Fixture id | warnCount | Top frames | Verdict |
|---|---:|---|---|
| `parse-floating-parse-progress-card` | **95** | useParseProgress.ts:225, :240 | 🔴 **OFFENDER** |
| `ui-button` | 39 | *(empty — bleed-through)* | ⚠️ noise¹ |
| `homepage-explore-blocks-list` | 0 | — | ✅ clean |
| `homepage-explore-block` | 0 | — | ✅ clean |
| `dashboard-recent-media-panel` | 0 | — | ✅ clean |
| `library-library-grid` | 0 | — | ✅ clean |

¹ `ui-button` has no effects/SSE; the 39 warnings are bleed-through from the prior parse fixture's unmount cleanup (its `componentFrames` is empty — no useParseProgress.ts frame fires during ui-button's own settle window). Confirms the offender is in the parse fixture, not ui-button.

### Why we short-circuited the full 123-walk

The Phase A stacks were unanimous (only `useParseProgress.ts` frames). Sally's four prime-suspect fixtures each return 0 warnings in single-fixture mode → her narrative-based shortlist is **wrong**. The empirical bisect points squarely at `useParseProgress`. Walking the remaining ~117 fixtures (none of which import `useParseProgress`) would be O(n) effort to confirm "still zero" — bugfix-10-3 spike precedent: stop once the offender is unambiguous. Full per-id JSON: `bisect-bugfix-19-4b-1-dev.json`.

### Bucket D ruled out (preview probe limitation)

`apps/web/src/routes/test/gallery.tsx:90-97` gates the route on `!import.meta.env.PROD` — the gallery returns "Access Denied" in any prod build, so the preview server (port 4201) cannot render fixtures and the `nx run web:preview` Phase A probe is structurally impossible without modifying the gate. Rather than relax the gate, Bucket D was ruled out **structurally**: the loop is callback-prop-identity drift (inline `{ onConnected, onParseStarted, … }` literal in `FloatingParseProgressCard.tsx:54` recreated each render → `handleEvent` identity churns → `connect` identity churns → useEffect re-fires → `setProgress(NEW obj)` triggers re-render → repeat). This chain does not require StrictMode's double-invocation; it is a real prod-mode loop pattern that React 18's depth limiter would surface in prod the moment a `FloatingParseProgressCard` is mounted with no SSE backend reachable.

## Root Cause (for Task 2 scoping)

`apps/web/src/components/parse/useParseProgress.ts:344-354`:

```ts
useEffect(() => {
  if (taskId) { initializeProgress(); connect(); }
  return () => disconnect();
}, [taskId, connect, disconnect, initializeProgress]);
```

`connect` (line 227) has deps `[taskId, handleEvent, onError, autoReconnect, reconnectDelay, maxReconnectAttempts]`. `handleEvent` (line 116) has deps `[taskId, onConnected, onParseStarted, onStepStarted, onStepCompleted, onStepFailed, onParseCompleted, onParseFailed]`. The `on*` callbacks are destructured from `options` (line 71-83) — but `options` in `FloatingParseProgressCard.tsx:54` is a **fresh object literal per render**, so every `on*` reference is new → `handleEvent` is new → `connect` is new → useEffect deps change every render → cleanup runs → effect body runs → `initializeProgress()` calls `setProgress(NEW object)` (line 100-108) which never bails out → re-render → ∞.

## Recommended Fix Form (AC #3 priority order)

- **(a) `useCallback` lift at call site** — wrap each `on*` lambda in `FloatingParseProgressCard.tsx:54-72` in `useCallback`. **Light, correct, fits the AC #3-(a) priority slot.** However, requires 6+ `useCallback` wraps in every `useParseProgress` consumer (currently only `FloatingParseProgressCard`, but future consumers would need the same discipline).
- **(d) `useMemo`-stable refs INSIDE `useParseProgress`** — stash the callbacks in `useRef`s that update each render and read them via a stable `handleEvent` (no callback in deps). **Most robust; protects all future consumers; matches the "stable function with mutable ref" idiom.** Slightly larger diff. Recommended.
- **(c) Drop `connect`/`disconnect`/`initializeProgress` from the line-345 useEffect deps with a `// rationale:` comment** — works, but loses ESLint's `react-hooks/exhaustive-deps` signal. Acceptable only if combined with (d) to make the dropped deps genuinely stable.
- **NOT acceptable:** changing `useState` initial values to bail out; commenting the fixture out; adding render-count guards.

## Probe Reproducibility

- **Committable Playwright spec:** `tests/e2e/bisect-bugfix-19-4b-1.spec.ts` (full 123-fixture walk). Run via:
  ```bash
  BISECT_MODE=dev BASE_URL=http://localhost:4200 \
    pnpm exec playwright test tests/e2e/bisect-bugfix-19-4b-1.spec.ts --project=chromium
  ```
  Outputs JSON to `bisect-bugfix-19-4b-1-${MODE}.json`. The spec re-attaches a `console.error` wrapper via `addInitScript` to capture `new Error().stack` (React 18's max-update-depth warning has no component-stack arg, so the JS call-site stack is the only signal). Kept post-fix as the AC #2 re-verification tool.
- **Targeted one-shot used for this verdict:** `/tmp/probe-targeted.mjs` (inline `node -e`-style script, 6 fixtures × dev+preview). Phase B for the full 123-walk stalled in `page.evaluate` drain after ~5 min on multiple runs — the targeted form short-circuits to the answer in ~30 s.

## Side-Discovery (out of scope, file follow-up)

The `?manifest=1` mode at `gallery.tsx:85` is **broken** as of the 19-4b Task 6 CR L2 fix. TanStack Router auto-parses `?manifest=1` to NUMBER 1 (default JSON-based search parser), but `validateSearch` checks `search.manifest === '1'` (STRING). The check fails → `manifest: undefined` → router strips the param → manifest mode never activates → `/test/gallery?manifest=1` renders the full gallery instead of the id list. The visual spec (`tests/visual/components.visual.spec.ts:93-101`) silently falls through to the gallery DOM scrape, so the bug was not caught. **NOT this story's scope** (AC #7 — `Don't add features beyond what the task requires`); recommend filing `bugfix-19-4b-2-gallery-manifest-mode-regression` or rolling into the next 19-4b touch.
