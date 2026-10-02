// Design ref: ux-design.pen Screen F6c-D-v2 (zv4hT) · F6c-M-v2 (x1uKHq)
/**
 * 匯入結果 — sub-8-1 AC #3. Sits above the glossary list after an import:
 * one summary line (new / same / different) and, for every term both sides
 * have with DIFFERENT renderings, 保留我的 / 改用他的 — per row or all at
 * once. Nothing was overwritten by the import itself; 改用他的 is an ordinary
 * edit, which confirms the term (⚖️ 2026-10-02).
 *
 * A refused import/export shows here too, in the same slot (note ④), never as
 * a toast: the panel stays open and the reason stays readable.
 */
import { CircleAlert } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { GlossaryImportConflict, GlossaryImportResult } from '../../services/glossaryService';

export interface GlossaryImportResultCardProps {
  result?: GlossaryImportResult;
  /** Conflicts still waiting for a decision (the parent drops resolved ones). */
  conflicts?: GlossaryImportConflict[];
  error?: string;
  busy?: boolean;
  onKeep?: (id: string) => void;
  onUseTheirs?: (conflict: GlossaryImportConflict) => void;
  onKeepAll?: () => void;
  onUseAllTheirs?: () => void;
  onDismiss: () => void;
}

const GHOST =
  'flex min-h-[36px] items-center justify-center rounded-[var(--radius-md)] px-3 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-primary)] disabled:opacity-50 max-sm:min-h-[44px]';

/** 「新增 12 個詞（待你確認）· 3 個跟你的一樣，略過 · 2 個跟你的不一樣：」 */
export function importSummary(result: GlossaryImportResult, pending: number): string {
  const parts: string[] = [];
  if (result.imported > 0) parts.push(`新增 ${result.imported} 個詞（待你確認）`);
  if (result.skipped > 0) parts.push(`${result.skipped} 個跟你的一樣，略過`);
  if (result.conflicts.length > 0) {
    parts.push(
      pending > 0
        ? `${result.conflicts.length} 個跟你的不一樣：`
        : `${result.conflicts.length} 個不一樣的都處理好了`
    );
  }
  return parts.length > 0 ? parts.join(' · ') : '檔案裡沒有新的詞';
}

export function GlossaryImportResultCard({
  result,
  conflicts = [],
  error,
  busy = false,
  onKeep,
  onUseTheirs,
  onKeepAll,
  onUseAllTheirs,
  onDismiss,
}: GlossaryImportResultCardProps) {
  if (error) {
    return (
      <div
        role="alert"
        data-testid="glossary-exchange-error"
        className="flex items-start gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] p-3"
      >
        <CircleAlert
          className="mt-0.5 h-4 w-4 shrink-0 text-[var(--error-text)]"
          aria-hidden="true"
        />
        <p className="flex-1 text-sm text-[var(--error-text)]">{error}</p>
        <button
          type="button"
          onClick={onDismiss}
          data-testid="glossary-exchange-dismiss"
          className={GHOST}
        >
          收起
        </button>
      </div>
    );
  }
  if (!result) return null;

  const title = result.title ? `已匯入「${result.title}」的詞彙表` : '已匯入詞彙表';
  return (
    <section
      data-testid="glossary-import-result"
      aria-label="匯入結果"
      className="flex flex-col gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] p-3"
    >
      <div className="flex items-center justify-between gap-2">
        <p className="text-sm font-semibold text-[var(--text-primary)]">{title}</p>
        <button
          type="button"
          onClick={onDismiss}
          data-testid="glossary-import-dismiss"
          className={GHOST}
        >
          收起
        </button>
      </div>
      <p data-testid="glossary-import-summary" className="text-sm text-[var(--text-secondary)]">
        {importSummary(result, conflicts.length)}
      </p>

      {conflicts.length > 0 && (
        <>
          <ul className="flex flex-col gap-1" data-testid="glossary-import-conflicts">
            {conflicts.map((c) => (
              <li
                key={c.id}
                data-testid={`glossary-conflict-${c.id}`}
                className="flex items-center justify-between gap-3 max-sm:flex-col max-sm:items-stretch max-sm:gap-1.5"
              >
                <div className="flex min-w-0 items-baseline gap-3 max-sm:flex-col max-sm:gap-0.5">
                  <span className="font-mono text-sm text-[var(--text-primary)]">{c.termSrc}</span>
                  <span className="text-sm text-[var(--text-secondary)]">
                    你的 <span className="text-[var(--text-primary)]">{c.mine}</span> · 他的{' '}
                    <span className="text-[var(--text-primary)]">{c.theirs}</span>
                  </span>
                </div>
                <div className="flex shrink-0 gap-1 max-sm:grid max-sm:grid-cols-2">
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onKeep?.(c.id)}
                    data-testid={`glossary-conflict-keep-${c.id}`}
                    className={GHOST}
                  >
                    保留我的
                  </button>
                  <button
                    type="button"
                    disabled={busy}
                    onClick={() => onUseTheirs?.(c)}
                    data-testid={`glossary-conflict-theirs-${c.id}`}
                    className={GHOST}
                  >
                    改用他的
                  </button>
                </div>
              </li>
            ))}
          </ul>
          {conflicts.length > 1 && (
            <div className={cn('flex justify-end gap-1 max-sm:grid max-sm:grid-cols-2')}>
              <button
                type="button"
                disabled={busy}
                onClick={onKeepAll}
                data-testid="glossary-conflict-keep-all"
                className={GHOST}
              >
                全部保留我的
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={onUseAllTheirs}
                data-testid="glossary-conflict-theirs-all"
                className={GHOST}
              >
                全部改用他的
              </button>
            </div>
          )}
        </>
      )}
    </section>
  );
}
