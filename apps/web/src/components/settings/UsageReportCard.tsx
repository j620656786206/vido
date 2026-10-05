// Design ref: ux-design.pen Screen C25-D (SHogC) · C25-M (bG3l3) · C26-D (FeCfY)
/**
 * 匿名使用回報 — infra-optin-usage-report-b1.
 *
 * The last card of 設定 → 連線設定: the one connection Vido makes on its own
 * accord. Off by default. When on, the backend sends one anonymous count a
 * week; this card shows the switch and — once something was sent — the exact
 * body of the last report, verbatim, so the claim "only these numbers" can be
 * checked by reading it (PRD P1-040-3). The payload is shown as the string the
 * server stored: never parsed, never pretty-printed.
 *
 * Four states (C26-D): unavailable (no receiver in this build), off,
 * on-but-never-sent, sent. One more the design does not draw: unavailable but
 * still on (the wizard said yes on a build without a receiver). Nothing is
 * sent then, and the switch stays usable so it can be turned off.
 */
import { useId } from 'react';
import { RefreshCw } from 'lucide-react';
import { cn } from '../../lib/utils';
import { useSetUsageReport, useUsageReport } from '../../hooks/useUsageReport';
import { USAGE_REPORT_DOCS_URL } from '../../services/usageReportService';
import { formatLocalDateTime } from '../../utils/formatLocalDateTime';

const CARD =
  'max-w-3xl rounded-[var(--radius-lg)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-4 md:p-6';

export function UsageReportCard() {
  const titleId = useId();
  const switchLabelId = useId();
  const payloadLabelId = useId();
  const status = useUsageReport();
  const save = useSetUsageReport();

  const header = (
    <header className="mb-6">
      <h2 id={titleId} className="text-base font-semibold text-[var(--text-primary)]">
        匿名使用回報
      </h2>
      <p className="text-xs text-[var(--text-muted)]">
        每週最多一次，把幾個匿名數字送給 Vido 維護者。
      </p>
    </header>
  );

  if (status.isPending) {
    return (
      <section aria-labelledby={titleId} className={CARD} data-testid="usage-report-card">
        {header}
        <p className="text-sm text-[var(--text-muted)]" role="status">
          讀取中…
        </p>
      </section>
    );
  }

  // Only a read that never succeeded hides the card: a failed background
  // refetch keeps showing the last good state.
  if (!status.data) {
    return (
      <section aria-labelledby={titleId} className={CARD} data-testid="usage-report-card">
        {header}
        <div className="flex items-center justify-between gap-4">
          <p className="text-sm text-[var(--text-secondary)]">讀不到匿名使用回報的狀態。</p>
          <button
            type="button"
            onClick={() => status.refetch()}
            className="inline-flex min-h-11 items-center gap-2 rounded-[var(--radius-md)] border border-[var(--border-subtle)] px-3 text-sm text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]"
          >
            <RefreshCw className="size-4" aria-hidden="true" />
            重試
          </button>
        </div>
      </section>
    );
  }

  const { available, enabled, lastSentAt, lastPayload } = status.data;
  // While the PUT is in flight the switch shows where it is going.
  const shownEnabled = save.isPending && save.variables !== undefined ? save.variables : enabled;
  // Unavailable blocks turning it on, never turning it off.
  const canToggle = available || enabled;
  const disabled = !canToggle || save.isPending;
  const hint = available
    ? '預設關閉。關掉之後就不會再送。'
    : enabled
      ? '這個版本沒有設定接收端，不會送出。可以關掉。'
      : '這個版本沒有設定接收端，無法開啟。';

  return (
    <section aria-labelledby={titleId} className={CARD} data-testid="usage-report-card">
      {header}
      <div className="space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <span
              id={switchLabelId}
              className="block text-sm font-medium text-[var(--text-secondary)]"
            >
              每週送一次匿名計數
            </span>
            <span className="text-xs text-[var(--text-muted)]">{hint}</span>
          </div>
          <button
            type="button"
            role="switch"
            aria-checked={shownEnabled}
            aria-labelledby={switchLabelId}
            disabled={disabled}
            onClick={() => save.mutate(!enabled)}
            className={cn(
              'flex size-11 shrink-0 items-center justify-center disabled:cursor-not-allowed',
              !canToggle && 'opacity-40'
            )}
            data-testid="usage-report-switch"
          >
            <span
              className={cn(
                'flex h-6 w-11 items-center rounded-full p-1 transition-colors',
                shownEnabled ? 'bg-[var(--accent-primary)]' : 'bg-[var(--bg-tertiary)]'
              )}
            >
              <span
                className={cn(
                  'size-4 rounded-full transition-transform',
                  shownEnabled
                    ? 'translate-x-5 bg-[var(--text-on-accent)]'
                    : 'translate-x-0 bg-[var(--text-muted)]'
                )}
              />
            </span>
          </button>
        </div>

        {save.isError && (
          <p className="text-xs text-[var(--error-text)]" role="alert">
            沒有存到，請再試一次。
          </p>
        )}

        {(lastSentAt || (available && enabled)) && (
          <div className="flex items-baseline justify-between gap-4">
            <span className="text-sm text-[var(--text-secondary)]">上次送出</span>
            {lastSentAt ? (
              <span className="font-mono text-sm text-[var(--text-primary)]">
                {formatLocalDateTime(lastSentAt)}
              </span>
            ) : (
              <span className="text-right text-sm text-[var(--text-muted)]">
                還沒送過——打開後一小時內會送出第一份
              </span>
            )}
          </div>
        )}

        {lastPayload && (
          <div role="group" aria-labelledby={payloadLabelId} className="space-y-2">
            <span id={payloadLabelId} className="block text-sm text-[var(--text-secondary)]">
              送出的內容（原文）
            </span>
            <pre
              className="m-0 rounded-[var(--radius-md)] bg-[var(--bg-primary)] p-3 font-mono text-xs break-all whitespace-pre-wrap text-[var(--text-secondary)]"
              data-testid="usage-report-payload"
            >
              {lastPayload}
            </pre>
          </div>
        )}

        <a
          href={USAGE_REPORT_DOCS_URL}
          target="_blank"
          rel="noreferrer"
          className="inline-block text-sm text-[var(--accent-text)] hover:underline"
        >
          送什麼、不送什麼 →
        </a>
      </div>
    </section>
  );
}
