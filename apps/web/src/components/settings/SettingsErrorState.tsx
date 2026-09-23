// Design ref: ux-design.pen Screen C16-D (uYGBU)
// 整頁載入失敗：OUqh7 圖示・eTUqu 標題・M5NMY 說明・wywZ6 重試
import { CircleAlert, RefreshCw } from 'lucide-react';

interface SettingsErrorStateProps {
  title: string;
  description: string;
  onRetry: () => void;
  isRetrying?: boolean;
  testId?: string;
}

/**
 * The one "this whole page could not load" state every settings tab shares.
 *
 * Five tabs used to hand-write this — each printing `error.message` (backend
 * English, verbatim) with no way to try again. This component takes a title
 * and a sentence the page writes in plain words, and deliberately has NO prop
 * that could carry a raw error: keeping the backend's text off the page is the
 * reason it exists.
 */
export function SettingsErrorState({
  title,
  description,
  onRetry,
  isRetrying = false,
  testId,
}: SettingsErrorStateProps) {
  return (
    <div
      data-testid={testId}
      className="flex flex-col items-center justify-center gap-3 py-16 text-center"
    >
      {/* Only the message is the live region. The button sits outside it, so
          flipping 重試 → 重試中… does not re-announce the whole alert. */}
      <div role="alert" className="flex flex-col items-center gap-3">
        <div
          data-testid="settings-error-state-icon"
          className="flex size-16 items-center justify-center rounded-full bg-[var(--error-tint)]"
        >
          <CircleAlert className="size-7 text-[var(--error-text)]" aria-hidden="true" />
        </div>
        {/* C16-D is one column at a flat 12px rhythm — icon, title, sentence, button. */}
        <p className="text-lg font-bold text-[var(--text-primary)] sm:text-xl">{title}</p>
        <p className="max-w-[440px] text-sm text-[var(--text-secondary)]">{description}</p>
      </div>
      {/* aria-disabled, not disabled: a disabled button drops keyboard focus to
          <body> the instant Enter is pressed, and the user loses their place. */}
      <button
        type="button"
        onClick={() => {
          if (!isRetrying) onRetry();
        }}
        aria-disabled={isRetrying || undefined}
        className="inline-flex min-h-11 items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-semibold text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-secondary)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] aria-disabled:cursor-not-allowed aria-disabled:text-[var(--text-muted)] aria-disabled:hover:bg-[var(--bg-tertiary)]"
      >
        <RefreshCw
          className={isRetrying ? 'size-4 motion-safe:animate-spin' : 'size-4'}
          aria-hidden="true"
        />
        {isRetrying ? '重試中…' : '重試'}
      </button>
    </div>
  );
}
