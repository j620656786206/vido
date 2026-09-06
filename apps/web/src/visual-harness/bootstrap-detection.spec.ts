/**
 * Tests for the bootstrap-detection parser (story bugfix-19-9).
 *
 * The parser consumes Playwright verify-only stdout and classifies each
 * failure into one of three buckets so the Visual Regression / Main workflow
 * can decide between (a) re-run verify-only, (b) trigger incremental bootstrap
 * PR for missing -linux baselines, or (c) fail loudly on real pixel diffs.
 *
 * Spec cases enumerated by story bugfix-19-9 AC #4 (a–h).
 *
 * Location rationale (deviation from story Task 1 sub-task path):
 *   The story Task 1 sub-task said `tests/visual/bootstrap-detection.{mjs,spec.ts}`,
 *   but `nx test web`'s vitest scope is `apps/web/{src,tests}/**` (per
 *   `apps/web/vite.config.mts` test.include). A spec at `tests/visual/` would
 *   never run under the project's vitest. Co-located both files under
 *   `apps/web/src/visual-harness/` preserves Rule 9 (test co-location) AND
 *   keeps them inside the web vitest scope. The workflow step 8 invokes the
 *   helper via `node apps/web/src/visual-harness/bootstrap-detection.mjs`.
 *   Deviation documented in story Completion Notes.
 *
 * @see _bmad-output/implementation-artifacts/bugfix-19-9-bootstrap-not-incremental.md AC #4
 * @see .github/workflows/visual-regression.yml step 8 (Check Linux baseline freshness)
 */
import { describe, it, expect } from 'vitest';
import { detectMissingBaselines } from './bootstrap-detection.mjs';

describe('detectMissingBaselines', () => {
  // (a) Canonical case from the actual failed run #26557906757 that originated this story.
  // 6 missing -linux baselines for library-recently-added/{recent,stale}/{default,hover,focus}.
  it('(a) 6 missing -linux baselines from real run #26557906757 → bootstrapNeeded=true', () => {
    const log = [
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/recent/default-visual-linux.png, writing actual.",
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/recent/hover-visual-linux.png, writing actual.",
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/recent/focus-visual-linux.png, writing actual.",
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/stale/default-visual-linux.png, writing actual.",
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/stale/hover-visual-linux.png, writing actual.",
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/stale/focus-visual-linux.png, writing actual.",
    ].join('\n');
    const result = detectMissingBaselines(log);
    expect(result.missingPaths).toHaveLength(6);
    expect(result.pixelDiffs).toEqual([]);
    expect(result.other).toEqual([]);
    expect(result.bootstrapNeeded).toBe(true);
    // Normalized relative paths (stripped of /home/runner/work/vido/vido/ prefix).
    expect(result.missingPaths[0]).toBe(
      'tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/recent/default-visual-linux.png'
    );
    expect(result.missingPaths[5]).toBe(
      'tests/visual/components.visual.spec.ts-snapshots/components/library-recently-added/stale/focus-visual-linux.png'
    );
  });

  // (b) Mixed missing + pixel-diff → bootstrapNeeded MUST be false (human review required;
  // auto-blessing a real pixel-diff would defeat the Rule 22 visual-regression gate).
  it('(b) 1 missing + 1 pixel-diff → bootstrapNeeded=false (conservative)', () => {
    const log = [
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/foo/default-visual-linux.png, writing actual.",
      'Error: Screenshot comparison failed:',
      '  Expected: tests/visual/components.visual.spec.ts-snapshots/components/bar/default-visual-linux.png',
      '  Received: test-results/bar-actual.png',
      '  Expected an image 1280x800, received 1280x800. 12345 pixels (ratio 0.012 of all image pixels) are different.',
    ].join('\n');
    const result = detectMissingBaselines(log);
    expect(result.missingPaths.length).toBe(1);
    expect(result.pixelDiffs.length).toBeGreaterThan(0);
    expect(result.bootstrapNeeded).toBe(false);
  });

  // (c) Clean Playwright output → all empty, bootstrapNeeded=false.
  it('(c) clean Playwright output → all empty, bootstrapNeeded=false', () => {
    const log = 'Running 1 test using 1 worker\n  1 passed (1.2m)\n';
    const result = detectMissingBaselines(log);
    expect(result.missingPaths).toEqual([]);
    expect(result.pixelDiffs).toEqual([]);
    expect(result.other).toEqual([]);
    expect(result.bootstrapNeeded).toBe(false);
  });

  // (d) -darwin missing-snapshot line → NOT eligible for Linux bootstrap (auto-bootstrap
  // is Linux-CI-only per the 19-5 + 19-4b platform-suffix decision).
  it('(d) -darwin missing → NOT counted as Linux-bootstrap-eligible (classified as other)', () => {
    const log =
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/foo/default-visual-darwin.png, writing actual.";
    const result = detectMissingBaselines(log);
    expect(result.missingPaths).toEqual([]);
    expect(result.other.length).toBeGreaterThan(0);
    expect(result.bootstrapNeeded).toBe(false);
  });

  // (e) Snapshot path outside the visual-harness snapshot dir → other.
  // Defensive: only the visual-harness suite is in bootstrap scope.
  it('(e) snapshot path outside visual-harness scope → classified as other', () => {
    const log =
      "Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/e2e/some-other-test-linux.png, writing actual.";
    const result = detectMissingBaselines(log);
    expect(result.missingPaths).toEqual([]);
    expect(result.other.length).toBeGreaterThan(0);
    expect(result.bootstrapNeeded).toBe(false);
  });

  // (f) Pixel-diff failure that references *-actual.png attachment must NOT be
  // confused with the missing-baseline `writing actual.` pattern. Real Playwright
  // output emits both in failed-comparison stanzas; the parser MUST distinguish
  // by the surrounding `doesn't exist` (missing) vs `Screenshot comparison failed`
  // (diff) markers.
  it('(f) pixel-diff stanza with actual.png attachment NOT confused with missing', () => {
    const log = [
      'Error: Screenshot comparison failed at tests/visual/components.visual.spec.ts-snapshots/components/foo-linux.png',
      '  Expected: tests/visual/components.visual.spec.ts-snapshots/components/foo-linux.png',
      '  Received: test-results/foo-actual.png',
      '  Expected an image 1280x800, received 1280x800. 9999 pixels (ratio 0.010 of all image pixels) are different.',
    ].join('\n');
    const result = detectMissingBaselines(log);
    expect(result.missingPaths).toEqual([]);
    expect(result.pixelDiffs.length).toBeGreaterThan(0);
    expect(result.bootstrapNeeded).toBe(false);
  });

  // (g) Multiple pixel-diffs, zero missing → bootstrapNeeded=false.
  it('(g) multiple pixel-diffs, zero missing → bootstrapNeeded=false', () => {
    const log = [
      'Error: Screenshot comparison failed at .../foo-visual-linux.png',
      '  4000 pixels (ratio 0.004 of all image pixels) are different.',
      'Error: Screenshot comparison failed at .../bar-visual-linux.png',
      '  6000 pixels (ratio 0.006 of all image pixels) are different.',
    ].join('\n');
    const result = detectMissingBaselines(log);
    expect(result.missingPaths).toEqual([]);
    expect(result.pixelDiffs.length).toBeGreaterThanOrEqual(2);
    expect(result.bootstrapNeeded).toBe(false);
  });

  // (h) Empty input → defensive: all empty, bootstrapNeeded=false.
  it('(h) empty input → all empty, bootstrapNeeded=false', () => {
    const result = detectMissingBaselines('');
    expect(result.missingPaths).toEqual([]);
    expect(result.pixelDiffs).toEqual([]);
    expect(result.other).toEqual([]);
    expect(result.bootstrapNeeded).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// Playwright 1.58 wording (2026-09-06). The repo ran 1.58 while the parser only
// knew the ≤ 1.4x `Screenshot comparison failed` header: three real diffs on
// PR #392's first CI round were invisible to it. Below is one of those stanzas
// VERBATIM (SGR colour codes included — the tee'd probe log has them), so a
// future Playwright bump that changes the wording again turns this red.
// ---------------------------------------------------------------------------
const PW158_DIFF_STANZA =
  '    Error: \u001b[2mexpect(\u001b[22m\u001b[31mlocator\u001b[39m\u001b[2m).\u001b[22mtoHaveScreenshot\u001b[2m(\u001b[22m\u001b[32mexpected\u001b[39m\u001b[2m)\u001b[22m failed\n\n    Locator: locator(\'section[data-gallery-id="generation-consent/list"]\').locator(\'[data-gallery-state]\').first()\n      6362 pixels (ratio 0.01 of all image pixels) are different.\n\n      Snapshot: components/generation-consent/list/default.png\n\n    Call log:\n    \u001b[2m  - Expect "soft toHaveScreenshot(components/generation-consent/list/default.png)" with timeout 10000ms\u001b[22m\n    \u001b[2m    - verifying given screenshot expectation\u001b[22m\n    \u001b[2m  - waiting for locator(\'section[data-gallery-id="generation-consent/list"]\').locator(\'[data-gallery-state]\').first()\u001b[22m\n    \u001b[2m    - locator resolved to <div class="inline-block" data-gallery-state="default">…</div>\u001b[22m\n    \u001b[2m  - taking element screenshot\u001b[22m\n    \u001b[2m    - disabled all CSS animations\u001b[22m\n    \u001b[2m  - waiting for fonts to load...\u001b[22m\n    \u001b[2m  - fonts loaded\u001b[22m\n    \u001b[2m  - attempting scroll into view action\u001b[22m\n    \u001b[2m    - waiting for element to be stable\u001b[22m\n    \u001b[2m  - 6362 pixels (ratio 0.01 of all image pixels) are different.\u001b[22m\n    \u001b[2m  - waiting 100ms before taking screenshot\u001b[22m\n    \u001b[2m  - waiting for locator(\'section[data-gallery-id="generation-consent/list"]\').locator(\'[data-gallery-state]\').first()\u001b[22m\n    \u001b[2m    - locator resolved to <div class="inline-block" data-gallery-state="default">…</div>\u001b[22m\n    \u001b[2m  - taking element screenshot\u001b[22m\n    \u001b[2m    - disabled all CSS animations\u001b[22m\n    \u001b[2m  - waiting for fonts to load...\u001b[22m\n    \u001b[2m  - fonts loaded\u001b[22m\n    \u001b[2m  - attempting scroll into view action\u001b[22m\n    \u001b[2m    - waiting for element to be stable\u001b[22m\n    \u001b[2m  - captured a stable screenshot\u001b[22m\n    \u001b[2m  - 6362 pixels (ratio 0.01 of all image pixels) are different.\u001b[22m\n\n\n      292 |             await expect.soft(page).toHaveScreenshot([\'components\', id, `${state}.png`]);\n      293 |           } else {\n    > 294 |             await expect.soft(stateDiv).toHaveScreenshot([\'components\', id, `${state}.png`]);\n          |                                         ^\n      295 |           }\n      296 |         }\n      297 |       }\n        at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts:294:41\n\n    attachment #1: components/generation-consent/list/default (image/png) ──────────────────────────\n    Expected: tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/list/default-visual-linux.png\n    Received: test-results/components.visual--visual--8935b-over-focus-open-bucket-4-4--visual/components/generation-consent/list/default-actual.png';

const PW158_MISSING_LINE =
  "    Error: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/generation-consent/list-mobile/default-visual-linux.png, writing actual.";

const PW158_SIZE_MISMATCH_STANZA = [
  '    Error: \u001b[2mexpect(\u001b[22m\u001b[31mlocator\u001b[39m\u001b[2m).\u001b[22mtoHaveScreenshot\u001b[2m(\u001b[22m\u001b[32mexpected\u001b[39m\u001b[2m)\u001b[22m failed',
  '',
  '      Expected an image 480px by 835px, received 480px by 851px.',
  '',
  '    attachment #1: components/media-media-detail-panel/focus (image/png) ───',
].join('\n');

describe('detectMissingBaselines — Playwright ≥ 1.5x wording', () => {
  it('(i) 1.58 pixel-diff stanza (verbatim, with ANSI) → 1 pixelDiff, bootstrapNeeded=false', () => {
    const r = detectMissingBaselines(PW158_DIFF_STANZA);
    expect(r.pixelDiffs).toHaveLength(1);
    expect(r.pixelDiffs[0]).toMatch(
      /^6362 pixels \(ratio 0\.01 of all image pixels\) are different\.$/
    );
    expect(r.missingPaths).toEqual([]);
    expect(r.other).toEqual([]);
    expect(r.bootstrapNeeded).toBe(false);
  });

  it('(j) 1.58: the dimmed call-log echoes of the count line are NOT extra diffs', () => {
    const r = detectMissingBaselines(PW158_DIFF_STANZA);
    // The stanza carries the count line three times (once real, twice as `- …` echoes).
    expect(PW158_DIFF_STANZA.match(/pixels \(ratio/g)).toHaveLength(3);
    expect(r.pixelDiffs).toHaveLength(1);
  });

  it('(k) 1.58: 1 missing -linux + 1 real diff → bootstrapNeeded=false (the 2026-09-06 near miss)', () => {
    const r = detectMissingBaselines(PW158_MISSING_LINE + '\n' + PW158_DIFF_STANZA);
    expect(r.missingPaths).toHaveLength(1);
    expect(r.pixelDiffs).toHaveLength(1);
    expect(r.bootstrapNeeded).toBe(false);
  });

  it('(l) 1.58: image size mismatch counts as a real diff', () => {
    const r = detectMissingBaselines(PW158_SIZE_MISMATCH_STANZA);
    expect(r.pixelDiffs).toHaveLength(1);
    expect(r.bootstrapNeeded).toBe(false);
  });

  it('(m) an `Error:` stanza the parser does not recognise blocks auto-bootstrap', () => {
    const unknown = [
      PW158_MISSING_LINE,
      '    Error: \u001b[31msome future Playwright wording we have never seen\u001b[39m',
      '      1 sprocket (ratio 0.42 of all sprockets) is bent.',
    ].join('\n');
    const r = detectMissingBaselines(unknown);
    expect(r.missingPaths).toHaveLength(1);
    expect(r.pixelDiffs).toEqual([]);
    expect(r.other).toHaveLength(1);
    expect(r.other[0]).toMatch(/^UNCLASSIFIED: 1 `Error:` stanza/);
    expect(r.bootstrapNeeded).toBe(false);
  });

  it('(n) ANSI codes around a missing-baseline line do not break the path capture', () => {
    const coloured =
      "    \u001b[31mError: A snapshot doesn't exist at /home/runner/work/vido/vido/tests/visual/components.visual.spec.ts-snapshots/components/x/default-visual-linux.png, writing actual.\u001b[39m";
    const r = detectMissingBaselines(coloured);
    expect(r.missingPaths).toEqual([
      'tests/visual/components.visual.spec.ts-snapshots/components/x/default-visual-linux.png',
    ]);
    expect(r.bootstrapNeeded).toBe(true);
  });
});
