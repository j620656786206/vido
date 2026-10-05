// Design ref: ux-design.pen Screen N6-D (GQae8)
/**
 * 匿名使用回報 — wizard step 5 of 6 (infra-optin-usage-report-b2).
 *
 * One yes/no question, off by default, asked before the summary so the answer
 * shows up there. No 跳過: the switch already has a safe default, so skipping
 * and pressing 下一步 would do the same thing (design ruling D3). What is sent
 * and what never is sit side by side, equal height, so the second column — the
 * one people care about — is read without a paragraph in the way (D4).
 */
import { useId } from 'react';
import { cn } from '../../lib/utils';
import type { StepProps } from './SetupWizard';
import { StepNav } from './StepNav';
import { USAGE_REPORT_DOCS_URL } from '../../services/usageReportService';

const SENT = [
  '一個隨機編號（不是從你的機器算出來的）',
  'Vido 版本',
  '最近 7 天 Vido 自己做出的字幕數',
];
const NEVER = ['片名、檔名、資料夾路徑', 'API 金鑰與任何設定', '你看了、想要或下載了什麼'];

function FactBox({ title, lines }: { title: string; lines: string[] }) {
  return (
    <div className="flex flex-col gap-1 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)] p-3">
      <h3 className="text-sm font-semibold text-[var(--text-primary)]">{title}</h3>
      <ul className="m-0 flex list-none flex-col gap-1 p-0">
        {lines.map((line) => (
          <li key={line} className="text-xs text-[var(--text-secondary)]">
            {line}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function UsageReportStep({ data, onUpdate, onNext, onBack }: StepProps) {
  const labelId = useId();
  const on = data.usageReportEnabled === true;

  return (
    <div className="flex flex-col gap-4" data-testid="usage-report-step">
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">匿名使用回報</h2>
      <p className="text-sm text-[var(--text-secondary)]">
        要不要每週告訴維護者「Vido 有在幫你做字幕」？只送幾個數字，預設關閉。
      </p>

      <div className="flex items-center justify-between gap-4">
        <div>
          <span id={labelId} className="block text-sm font-medium text-[var(--text-secondary)]">
            每週送一次匿名計數
          </span>
          <span className="text-xs text-[var(--text-muted)]">
            之後隨時可以在「設定 → 連線設定」改。
          </span>
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={on}
          aria-labelledby={labelId}
          onClick={() => onUpdate({ usageReportEnabled: !on })}
          className="flex size-11 shrink-0 items-center justify-center"
          data-testid="usage-report-step-switch"
        >
          <span
            className={cn(
              'flex h-6 w-11 items-center rounded-full p-1 transition-colors',
              on ? 'bg-[var(--accent-primary)]' : 'bg-[var(--bg-tertiary)]'
            )}
          >
            <span
              className={cn(
                'size-4 rounded-full transition-transform',
                on
                  ? 'translate-x-5 bg-[var(--text-on-accent)]'
                  : 'translate-x-0 bg-[var(--text-muted)]'
              )}
            />
          </span>
        </button>
      </div>

      {/* Grid rows stretch by default: the two boxes are always as tall as the
          taller one, whatever the copy wraps to (design ruling, N6-D). */}
      <div className="grid grid-cols-2 gap-3" data-testid="usage-report-facts">
        <FactBox title="會送" lines={SENT} />
        <FactBox title="絕不送" lines={NEVER} />
      </div>

      <a
        href={USAGE_REPORT_DOCS_URL}
        target="_blank"
        rel="noreferrer"
        className="text-xs text-[var(--accent-text)] hover:underline"
      >
        看完整說明 →
      </a>

      <StepNav onNext={onNext} onBack={onBack} />
    </div>
  );
}
