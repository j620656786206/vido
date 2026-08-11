// Design ref: ux-design.pen Screen F20-D-v2 (D7MOm)
/**
 * F20 產生字幕．空狀態 (sub-4-3 AC #5) — every listable candidate already has
 * a zh-Hant subtitle. Footer offers ONLY 關閉 (no 開始產生 — there is nothing
 * to consent to).
 */
import { SquareCheck } from 'lucide-react';

export function ConsentEmptyState() {
  return (
    <div
      data-testid="consent-empty-state"
      className="flex flex-1 flex-col items-center justify-center gap-4 px-6 py-14 text-center"
    >
      <span
        aria-hidden="true"
        className="flex h-16 w-16 items-center justify-center rounded-full bg-[var(--bg-tertiary)]"
      >
        <SquareCheck className="h-8 w-8 text-[var(--success)]" />
      </span>
      <p className="text-base font-semibold text-[var(--text-primary)]">所有影片都有繁中字幕了</p>
      <p className="text-[13px] text-[var(--text-secondary)]">
        掃描到新影片時，這裡會列出可產生字幕的項目。
      </p>
    </div>
  );
}
