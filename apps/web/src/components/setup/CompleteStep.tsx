// Design ref: ux-design.pen Screen N5-D (CWh3E)
import { CircleCheck } from 'lucide-react';
import type { SetupLibraryEntry } from '../../services/setupService';
import type { StepProps } from './SetupWizard';
import { languageLabel } from './setupLanguages';
import { StepNav } from './StepNav';

const TYPE_LABELS: Record<SetupLibraryEntry['contentType'], string> = {
  movie: '電影',
  series: '影集',
};

/** 「2 個（電影・影集）」— how many libraries, and which kinds, each named once. */
function librarySummary(libraries: SetupLibraryEntry[] | undefined): string | null {
  const filled = (libraries ?? []).filter((lib) => lib.path.trim());
  if (filled.length === 0) return null;
  const kinds = (['movie', 'series'] as const)
    .filter((type) => filled.some((lib) => lib.contentType === type))
    .map((type) => TYPE_LABELS[type]);
  return `${filled.length} 個（${kinds.join('・')}）`;
}

export function CompleteStep({ data, onNext, onBack, isSubmitting }: StepProps) {
  const libraries = librarySummary(data.libraries);
  const rows = [
    { label: '語言', value: languageLabel(data.language || 'zh-TW'), set: true },
    { label: '媒體資料夾', value: libraries ?? '未設定', set: libraries !== null },
    { label: 'qBittorrent', value: data.qbtUrl ? '已設定' : '未設定', set: !!data.qbtUrl },
    { label: 'TMDb 金鑰', value: data.tmdbApiKey ? '已設定' : '未設定', set: !!data.tmdbApiKey },
    {
      label: 'Claude 金鑰',
      value: data.claudeApiKey ? '已設定' : '未設定',
      set: !!data.claudeApiKey,
    },
  ];

  return (
    <div className="flex flex-col gap-4" data-testid="complete-step">
      <div className="flex flex-col items-center gap-2">
        <div className="flex h-14 w-14 items-center justify-center rounded-full bg-[var(--success-tint)]">
          <CircleCheck className="size-6.5 text-[var(--success-text)]" aria-hidden="true" />
        </div>
        <h2 className="text-lg font-semibold text-[var(--text-primary)]">設定完成！</h2>
        <p className="text-sm text-[var(--text-secondary)]">以下是您的設定摘要。</p>
      </div>

      <dl className="rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-secondary)]">
        {rows.map((row) => (
          <div key={row.label} className="flex items-center gap-3 px-4 py-3 text-sm">
            <dt className="w-24 shrink-0 text-[var(--text-secondary)] sm:w-40">{row.label}</dt>
            <dd
              className={`min-w-0 font-medium ${
                row.set ? 'text-[var(--text-primary)]' : 'text-[var(--text-muted)]'
              }`}
            >
              {row.value}
            </dd>
          </div>
        ))}
      </dl>

      <p className="text-xs text-[var(--text-muted)]">未設定的項目之後都可以在「設定」裡補上。</p>

      <StepNav
        onNext={onNext}
        onBack={onBack}
        backDisabled={isSubmitting}
        nextDisabled={isSubmitting}
        nextLabel={isSubmitting ? '儲存中...' : '完成設定'}
        nextTestId="finish-button"
      />
    </div>
  );
}
