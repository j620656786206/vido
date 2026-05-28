/**
 * Bootstrap-detection parser (story bugfix-19-9 AC #4).
 *
 * Consumes Playwright verify-only stdout and classifies each failure into one
 * of three buckets so the Visual Regression / Main workflow can decide:
 *
 *   - **missing-baseline** — Playwright emitted
 *     `Error: A snapshot doesn't exist at <path>, writing actual.` for `<path>`
 *     ending in `-linux.png` AND under the visual-harness snapshot dir
 *     → fixture is in scope for incremental bootstrap.
 *   - **pixel-diff** — Playwright emitted `Screenshot comparison failed`
 *     against an existing baseline → REAL regression; the workflow MUST NOT
 *     auto-bootstrap (would defeat the Rule 22 visual-regression gate).
 *   - **other** — any other failure class (missing snapshot for the wrong
 *     platform / outside scope, test infra error, etc.) → fail the job for
 *     human review.
 *
 * Output a `bootstrapNeeded` flag that is `true` ONLY if
 * `missingPaths.length > 0 && pixelDiffs.length === 0 && other.length === 0`.
 * Conservative classification: unknown patterns fall into `other`, never
 * into `missingPaths`. False-negative cost (manual PR like #11) is low;
 * false-positive cost (auto-bless a real regression) is HIGH.
 *
 * CLI entry: when invoked as `node bootstrap-detection.mjs`, reads stdin
 * (the verify-only probe's stdout/stderr) and writes the classification
 * outputs to `$GITHUB_OUTPUT` if set, plus a human-readable summary to
 * stdout. Always exits 0 — the workflow's downstream `if:` conditionals
 * decide what to do based on the emitted outputs.
 *
 * @see _bmad-output/implementation-artifacts/bugfix-19-9-bootstrap-not-incremental.md AC #4
 * @see .github/workflows/visual-regression.yml step 8 (Check Linux baseline freshness)
 */

'use strict';

// Match Playwright's missing-baseline emission. Captured group is the absolute path.
// Playwright emits this for EVERY missing snapshot during verify-only runs.
const MISSING_RE = /A snapshot doesn't exist at (.+?), writing actual\./;

// Match Playwright's pixel-diff failure header line. One match per failed comparison.
const PIXEL_DIFF_HEADER_RE = /Screenshot comparison failed/i;

// Visual-harness snapshot dir — only files under this prefix are bootstrap-eligible.
const VISUAL_SNAPSHOT_PREFIX = 'tests/visual/components.visual.spec.ts-snapshots/';

/**
 * Classify a Playwright verify-only log into missing-baseline / pixel-diff / other.
 *
 * @param {string} playwrightOutput - stdout + stderr from `pnpm run test:visual`.
 * @returns {{
 *   missingPaths: string[],
 *   pixelDiffs: string[],
 *   other: string[],
 *   bootstrapNeeded: boolean,
 * }}
 */
export function detectMissingBaselines(playwrightOutput) {
  const missingPaths = [];
  const pixelDiffs = [];
  const other = [];

  if (typeof playwrightOutput !== 'string' || playwrightOutput.length === 0) {
    return { missingPaths, pixelDiffs, other, bootstrapNeeded: false };
  }

  const lines = playwrightOutput.split('\n');
  for (const line of lines) {
    // 1. Missing-baseline match first (more specific pattern; `writing actual.` is
    //    unique to Playwright's missing-snapshot path and would never appear in a
    //    pixel-diff stanza).
    const missing = line.match(MISSING_RE);
    if (missing) {
      const fullPath = missing[1];
      // Normalize: strip absolute prefix, keep relative to repo root.
      const idx = fullPath.indexOf(VISUAL_SNAPSHOT_PREFIX);
      const relPath = idx >= 0 ? fullPath.slice(idx) : fullPath;

      // Scope guard: only -linux baselines under the visual-harness snapshot dir
      // are eligible for auto-bootstrap. Anything else falls into `other`
      // (defensive — the workflow's incremental path commits ONLY Linux baselines).
      const inScope = relPath.startsWith(VISUAL_SNAPSHOT_PREFIX) && relPath.endsWith('-linux.png');
      if (inScope) {
        missingPaths.push(relPath);
      } else {
        other.push(line);
      }
      continue;
    }

    // 2. Pixel-diff failure header — count each stanza once.
    //    Conservative: if Playwright ever changes the literal "Screenshot comparison
    //    failed" wording, this branch silently misses real diffs → `bootstrapNeeded`
    //    could mis-classify a diff scenario as `bootstrap-needed`. AC #4 spec case
    //    (f) + (g) pin this exact wording so the spec catches such Playwright bumps
    //    in CI before the workflow can mis-fire in production.
    if (PIXEL_DIFF_HEADER_RE.test(line)) {
      pixelDiffs.push(line);
      continue;
    }

    // 3. Lines that match neither pattern are ignored (most of Playwright's output
    //    is test-progress / banner / passing-test lines — not classifiable failures).
  }

  const bootstrapNeeded =
    missingPaths.length > 0 && pixelDiffs.length === 0 && other.length === 0;

  return { missingPaths, pixelDiffs, other, bootstrapNeeded };
}

// CLI entry — only fires when this file is invoked directly (not when imported by the spec).
if (import.meta.url === `file://${process.argv[1]}`) {
  const fs = await import('node:fs');

  // Read entire stdin synchronously. The workflow pipes the verify-only probe's
  // captured log file content here; size is bounded by Playwright's output for
  // ~123 visual fixtures (well under tens of MB even with diff details).
  const input = fs.readFileSync(0, 'utf-8');
  const result = detectMissingBaselines(input);

  // Emit GitHub Actions step outputs. `missing_paths` uses the heredoc syntax
  // (`<<EOF ... EOF`) so newline-joined paths survive YAML escape rules.
  const ghOutput = process.env.GITHUB_OUTPUT;
  if (ghOutput) {
    const outputLines = [
      `bootstrap_needed=${result.bootstrapNeeded}`,
      `missing_count=${result.missingPaths.length}`,
      `pixel_diff_count=${result.pixelDiffs.length}`,
      `other_count=${result.other.length}`,
      `missing_paths<<BOOTSTRAP_DETECTION_EOF`,
      ...result.missingPaths,
      `BOOTSTRAP_DETECTION_EOF`,
    ];
    fs.appendFileSync(ghOutput, outputLines.join('\n') + '\n');
  }

  // Human-readable summary for the workflow's run log.
  console.log(`bootstrap-detection summary:`);
  console.log(`  bootstrap_needed = ${result.bootstrapNeeded}`);
  console.log(`  missing -linux baselines: ${result.missingPaths.length}`);
  console.log(`  pixel diffs on existing baselines: ${result.pixelDiffs.length}`);
  console.log(`  other (out-of-scope or non-Linux): ${result.other.length}`);
  if (result.missingPaths.length > 0) {
    console.log(`  missing paths:`);
    for (const p of result.missingPaths) {
      console.log(`    - ${p}`);
    }
  }
  // Always exit 0 — the workflow uses step outputs to decide next action, not the
  // exit code of this script. The verify-only probe upstream may have exited 1
  // (because Playwright found missing snapshots / diffs), but THIS script's job
  // is just to classify, never to fail.
  process.exit(0);
}
