// Design ref: ux-design.pen Screen F20-D-v2 (D7MOm)
/**
 * F20 產生字幕．空狀態 (sub-4-3 AC #5).
 *
 * ⚠️ THIS SCREEN USED TO LIE, and it took a homepage critique to catch it.
 *
 * The phase that renders it is「0 listable candidates」— `listableCandidates()`
 * keeps only `route === 'extract' | 'asr'`, so an empty list means *nothing is
 * generatable*. That covers TWO different truths:
 *
 *   (a) the analysis produced records and every one was already covered
 *       → 「所有影片都有繁中字幕了」 is TRUE
 *   (b) the analysis produced NOTHING generatable at all
 *       → that sentence is FALSE, and it was being printed anyway
 *
 * Case (b) is not hypothetical. ASR is movies-only today (PRODUCT.md: 影集入口
 * 開發中), so a **series-only library** — an ordinary setup — analyses, yields
 * zero candidates, and used to be told every film already had subtitles while
 * having none. Verified on the seeded env: `analyzed 15 / total 15 /
 * candidates [] / skipped_count 0` against a library of 18 titles with 0
 * covered. Three numbers, and the one sentence on screen matched none of them.
 *
 * PRODUCT.md calls this failure mode fatal —「無人值守＝沒人發現它在騙你」—
 * and the false claim wore GREEN, which 固定詞彙 reserves for 正在發生.
 *
 * So the green check is now EARNED, not default: it appears only for (a). Case
 * (b) reports what was actually measured and makes no claim about coverage.
 */
import { SquareCheck, SearchX } from 'lucide-react';

export interface ConsentEmptyStateProps {
  /**
   * TRUE only when the analysis returned candidate records and every one of
   * them was filtered out as already-covered. Defaults to FALSE: an unknown
   * reason must never render as「everything is done」, because that is the
   * direction that lies.
   */
  allCovered?: boolean;
  /** How many titles the analysis actually looked at, for the honest readout. */
  analyzed?: number;
}

export function ConsentEmptyState({ allCovered = false, analyzed }: ConsentEmptyStateProps) {
  return (
    <div
      data-testid="consent-empty-state"
      data-empty-reason={allCovered ? 'all-covered' : 'no-candidates'}
      className="flex flex-1 flex-col items-center justify-center gap-4 px-6 py-14 text-center"
    >
      <span
        aria-hidden="true"
        className="flex h-16 w-16 items-center justify-center rounded-full bg-[var(--bg-tertiary)]"
      >
        {allCovered ? (
          <SquareCheck className="h-8 w-8 text-[var(--success-text)]" />
        ) : (
          // Muted, not green. Nothing was completed — nothing was found.
          <SearchX className="h-8 w-8 text-[var(--text-muted)]" />
        )}
      </span>

      {allCovered ? (
        <>
          <p className="text-base font-semibold text-[var(--text-primary)]">
            所有影片都有繁中字幕了
          </p>
          <p className="text-[13px] text-[var(--text-secondary)]">
            掃描到新影片時，這裡會列出可產生字幕的項目。
          </p>
        </>
      ) : (
        <>
          <p className="text-base font-semibold text-[var(--text-primary)]">
            沒有找到可以產生字幕的項目
          </p>
          <p className="max-w-sm text-[13px] text-[var(--text-secondary)]">
            {analyzed !== undefined && analyzed > 0
              ? `分析了 ${analyzed.toLocaleString()} 部，沒有一部可以抽取內嵌字幕或用語音辨識產生。`
              : '這次分析沒有可以抽取內嵌字幕或用語音辨識產生的項目。'}
            {/* Stated as context, not as a diagnosis — this component cannot
                know the library's composition, only what the analysis returned.
                It is the most common reason a non-empty library lands here. */}
            影集的語音辨識還在開發中。
          </p>
        </>
      )}
    </div>
  );
}
