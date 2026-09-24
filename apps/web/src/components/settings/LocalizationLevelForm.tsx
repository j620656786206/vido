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
import { useEffect, useRef } from 'react';
import { AlertTriangle, Loader2 } from 'lucide-react';
import { RadioDot } from '../ui/RadioDot';
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
  // A choice made while a save is in flight is remembered, not dropped: arrow
  // keys move focus (and the user's intent) immediately, and the last choice
  // is the one that must win (dsr-3d CR L1).
  const queued = useRef<LocalizationLevel | null>(null);
  const choose = (level: LocalizationLevel) => {
    if (save.isPending) queued.current = level;
    else save.mutate(level);
  };
  useEffect(() => {
    if (save.isPending || !queued.current) return;
    const next = queued.current;
    queued.current = null;
    if (next !== data?.level) save.mutate(next);
  }, [save, data?.level]);

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

      {/* No outer card (C9): the three options are the cards. The legend and
          the「這是口味，不是對錯」line stay — they are the only words on this
          page saying there is no right answer (C9 now draws them too). The
          promise「已經翻好的不會動」was checked against the pipeline (dsr-3d
          Task 1): the level is read once per item when it starts and rides the
          segment-cache key; saving only writes the setting. */}
      <div data-testid="localization-form">
        {/* Not `disabled` while saving: that blurred the radio the user had
            just moved to with the arrow keys, dropping keyboard focus on
            <body> (dsr-3d e2e). A change made meanwhile is queued (see `choose`). */}
        <fieldset aria-busy={save.isPending || undefined}>
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

          <div className="flex flex-col gap-3" role="radiogroup" aria-label="在地化程度">
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
                    // child of the label and the drawn radio sits in column one.
                    'grid cursor-pointer grid-cols-[auto_1fr] gap-x-3 rounded-[var(--radius-lg)] border p-4 transition-colors',
                    checked
                      ? 'border-[var(--accent-primary)] bg-[var(--accent-subtle)]'
                      : 'border-[var(--border-subtle)] bg-[var(--bg-secondary)] hover:bg-[var(--bg-tertiary)]',
                    save.isPending && 'cursor-not-allowed opacity-60'
                  )}
                >
                  {/* The native radio does the work (arrow keys, the name group);
                      RadioDot, right after it, draws it via peer-*. */}
                  <input
                    type="radio"
                    name="subtitle-localization-level"
                    value={spec.id}
                    checked={checked}
                    onChange={() => choose(spec.id)}
                    className="peer sr-only"
                  />
                  <RadioDot className="col-start-1 row-start-1" />
                  <span
                    className={cn(
                      'col-start-2 row-start-1 text-sm font-semibold',
                      checked ? 'text-[var(--accent-text)]' : 'text-[var(--text-primary)]'
                    )}
                  >
                    {spec.label}
                  </span>
                  <span className="col-start-2 row-start-2 mt-1 text-xs text-[var(--text-secondary)]">
                    {spec.description}
                  </span>
                  <span className="col-start-2 row-start-3 mt-1 font-mono text-xs text-[var(--text-muted)]">
                    {/* C9 shows the example bare; a screen reader still hears
                        that it is one. */}
                    <span className="sr-only">例：</span>
                    {spec.example}
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
            <p
              role="status"
              className="mt-4 flex items-center gap-2 text-xs text-[var(--text-muted)]"
            >
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
              儲存中…
            </p>
          )}
        </fieldset>
      </div>
    </div>
  );
}
