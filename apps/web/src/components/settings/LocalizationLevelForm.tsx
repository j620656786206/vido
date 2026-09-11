// Design ref: ux-design.pen Screen C9-D (NR3zK) · C9-M (AWYm0)
/**
 * 在地化程度 — the sub-7-4 taste dial for AI subtitles.
 *
 * Three levels, one radio card. This is a matter of TASTE (party-mode
 * 2026-09-03: some people hate seeing 全聯 in an American sitcom), so it is
 * the user's setting — never a score-driven default. Each option carries one
 * sentence and one example, because the difference between the levels is
 * easier to SEE than to describe.
 *
 * Honest about precedence like ApiKeysForm: a level that came from the
 * SUBTITLE_LOCALIZATION_LEVEL env var says so, and saving here overrides it.
 */
import { AlertTriangle, Check, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import {
  useSaveSubtitleLocalization,
  useSubtitleLocalization,
} from '../../hooks/useSubtitleLocalization';
import type { LocalizationLevel } from '../../services/subtitleLocalizationService';

interface LevelSpec {
  id: LocalizationLevel;
  label: string;
  description: string;
  example: string;
}

export const LEVEL_SPECS: LevelSpec[] = [
  {
    id: 'literal',
    label: '直譯',
    description: '照原文翻，不把故事世界搬到台灣。超市就是超市，感恩節就是感恩節。',
    example: '「I grabbed milk at the grocery store」→「我在超市買了牛奶」',
  },
  {
    id: 'standard',
    label: '台灣用語（預設）',
    description: '用台灣的詞和語氣（影片、品質、超商），但不拿本地品牌或俚語替換泛稱。',
    example: '「I grabbed milk at the grocery store」→「我去超市買了牛奶」',
  },
  {
    id: 'ott',
    label: 'OTT 風格',
    description:
      'Netflix、Apple TV+ 台灣字幕的口吻：日常場景的泛稱可以換成台灣人會講的字，專有品牌名絕不亂換。',
    example: '「I grabbed milk at the grocery store」→「我去全聯買了牛奶」',
  },
];

export function LocalizationLevelForm() {
  const { data, isLoading, isError, error } = useSubtitleLocalization();
  const save = useSaveSubtitleLocalization();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12" data-testid="localization-loading">
        <Loader2 className="h-6 w-6 animate-spin text-[var(--text-secondary)]" />
      </div>
    );
  }

  const current = data?.level ?? 'standard';
  const fromEnv = data?.source === 'env';

  return (
    <div className="max-w-3xl">
      {isError && (
        <div
          data-testid="localization-load-error"
          role="status"
          aria-live="polite"
          className="mb-6 flex items-start gap-3 rounded-md bg-[var(--error-tint)] p-4"
        >
          <AlertTriangle
            className="mt-0.5 h-4 w-4 shrink-0 text-[var(--error-text)]"
            aria-hidden="true"
          />
          <p className="text-sm text-[var(--error-text)]">
            無法讀取在地化程度{error?.message ? `：${error.message}` : ''}
          </p>
        </div>
      )}

      <div className="rounded-lg border border-[var(--border-subtle)] bg-[var(--bg-secondary)]/50 p-6">
        <fieldset disabled={save.isPending}>
          <legend className="mb-1 text-base font-semibold text-[var(--text-primary)]">
            AI 字幕的在地化程度
          </legend>
          <p className="mb-4 text-sm text-[var(--text-secondary)]">
            這是口味，不是對錯。選了之後，下一部翻譯的片就會用新的風格；已經翻好的不會動。
          </p>

          {fromEnv && (
            <p
              data-testid="localization-env-note"
              className="mb-4 rounded-md bg-[var(--info-tint)] px-3 py-2 text-xs text-[var(--info-text)]"
            >
              目前的程度來自環境變數 SUBTITLE_LOCALIZATION_LEVEL；在這裡儲存會蓋過它。
            </p>
          )}

          <div className="flex flex-col gap-2" role="radiogroup" aria-label="在地化程度">
            {LEVEL_SPECS.map((spec) => {
              const checked = spec.id === current;
              return (
                <label
                  key={spec.id}
                  data-testid={`localization-option-${spec.id}`}
                  data-selected={checked ? 'true' : 'false'}
                  className={cn(
                    // Grid, not nested flex: jsx-a11y only looks two levels
                    // deep for a label's text, so every text span is a DIRECT
                    // child of the label and the radio sits in column one.
                    'grid cursor-pointer grid-cols-[auto_1fr] gap-x-3 rounded-[var(--radius-sm)] border px-3 py-3 transition-colors',
                    checked
                      ? 'border-[var(--accent-primary)] bg-[var(--bg-tertiary)]'
                      : 'border-transparent hover:bg-[var(--bg-tertiary)]',
                    save.isPending && 'cursor-not-allowed opacity-60'
                  )}
                >
                  <input
                    type="radio"
                    name="subtitle-localization-level"
                    value={spec.id}
                    checked={checked}
                    onChange={() => save.mutate(spec.id)}
                    className="col-start-1 row-start-1 mt-1 h-4 w-4 shrink-0 accent-[var(--accent-primary)] disabled:cursor-not-allowed"
                  />
                  <span className="col-start-2 row-start-1 flex items-center gap-2 text-sm font-medium text-[var(--text-primary)]">
                    {spec.label}
                    {checked && data?.source === 'settings' && (
                      <Check
                        className="h-3.5 w-3.5 text-[var(--success-text)]"
                        aria-hidden="true"
                      />
                    )}
                  </span>
                  <span className="col-start-2 row-start-2 mt-0.5 text-sm text-[var(--text-secondary)]">
                    {spec.description}
                  </span>
                  <span className="col-start-2 row-start-3 mt-1 text-xs text-[var(--text-muted)]">
                    例：{spec.example}
                  </span>
                </label>
              );
            })}
          </div>

          {save.isError && (
            <p
              data-testid="localization-save-error"
              role="status"
              aria-live="polite"
              className="mt-4 text-sm text-[var(--error-text)]"
            >
              儲存失敗{save.error?.message ? `：${save.error.message}` : ''}
            </p>
          )}
          {save.isPending && (
            <p className="mt-4 flex items-center gap-2 text-xs text-[var(--text-muted)]">
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
              儲存中…
            </p>
          )}
        </fieldset>
      </div>
    </div>
  );
}
