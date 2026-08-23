// Design ref: ux-design.pen Screen J6-D (zMYsL)
/**
 * .nfo localization entry: button → confirmation → inline result (Story 9R-13b).
 *
 * The whole component exists to make ONE promise legible before it is kept:
 * for a TV show this OVERWRITES a file the user may have curated by hand. The
 * backend enforces that with a `confirm_replace` gate; this is the human half
 * of that gate, so the flag is never sent until someone has read the dialog.
 *
 * Movies are additive and could technically skip confirmation — Sally ruled
 * they must NOT (9R-UX, 2026-08-21): a movie run still spends LLM budget, and
 * the 2026-08-19「花錢須同意」ruling means a paid action needs an explicit yes.
 * The movie dialog therefore says two things: your file is safe, AND this costs.
 *
 * Result is an inline pill that REPLACES the button (RequestButton.tsx's
 * `role="status"` + `aria-live` vocabulary) rather than a floating toast —
 * there is no shared Toast component in ui/, and inventing one here would be a
 * second system.
 */
import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useMutation } from '@tanstack/react-query';
import {
  Languages,
  Check,
  ShieldCheck,
  Sparkles,
  TriangleAlert,
  KeyRound,
  Loader2,
} from 'lucide-react';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '../ui/Dialog';
import {
  nfoLocalizerService,
  isSeriesResult,
  NFO_ERROR_CODES,
  NfoLocalizeApiError,
  type NfoLocalizeResult,
  type NfoSeriesLocalizeResult,
} from '../../services/nfoLocalizerService';
import { cn } from '@/lib/utils';

export interface NfoLocalizeActionProps {
  mediaType: 'movie' | 'tv';
  id: string;
  /** Rendered only when the row has a file on disk — nowhere to put a .nfo otherwise. */
  hasFilePath: boolean;
  className?: string;
}

// CR H1: success is keyed on the backend's `replaced`, NOT on media type.
// A movie whose two nfo slots were both occupied IS replaced (backup taken,
// nfo_localizer_service.go:238); a TV show with no tvshow.nfo yet is NOT
// (nothing to back up, :591). Deriving the pill from mediaType told the user
// a backup existed when it did not — the bugfix-j "lying status" class.
type Outcome =
  | { kind: 'ok'; replaced: boolean }
  | { kind: 'batch'; succeeded: number; skipped: number; failed: number }
  | { kind: 'disabled' }
  | { kind: 'error'; message: string };

const pillBase = 'inline-flex h-10 items-center gap-2 rounded-full px-4 text-[13px] font-semibold';

export function NfoLocalizeAction({
  mediaType,
  id,
  hasFilePath,
  className,
}: NfoLocalizeActionProps) {
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const [includeEpisodes, setIncludeEpisodes] = useState(false);
  const [outcome, setOutcome] = useState<Outcome | null>(null);
  const isMovie = mediaType === 'movie';

  const localize = useMutation({
    mutationFn: async (): Promise<NfoLocalizeResult | NfoSeriesLocalizeResult> =>
      isMovie
        ? nfoLocalizerService.localizeMovieNfo(id)
        : // 🔴 `true` is passed ONLY from here — the dialog's confirm button.
          // Never default it at the service layer: that would turn a deliberate
          // user decision into a constant and defeat the backend's file guard.
          nfoLocalizerService.localizeSeriesNfo(id, {
            confirmReplace: true,
            includeEpisodes,
          }),
    onSuccess: (result) => {
      setOpen(false);
      if (isSeriesResult(result)) {
        setOutcome({
          kind: 'batch',
          succeeded: result.succeeded,
          skipped: result.skipped,
          failed: result.failed,
        });
        return;
      }
      setOutcome({ kind: 'ok', replaced: result.replaced });
    },
    onError: (error: unknown) => {
      setOpen(false);
      const code = error instanceof NfoLocalizeApiError ? error.code : '';
      if (code === NFO_ERROR_CODES.disabled) {
        setOutcome({ kind: 'disabled' });
        return;
      }
      // Rule 3: the user sees zh-TW. The backend's 500 message is English
      // ("Failed to localize metadata") — map the known codes here and only
      // fall through to the raw text for a code this surface does not know.
      const message =
        code === NFO_ERROR_CODES.missingPath
          ? '請先掃描媒體庫'
          : code === NFO_ERROR_CODES.failed
            ? '在地化失敗，請查看伺服器記錄'
            : code === NFO_ERROR_CODES.notConfirmed
              ? '未經確認的請求被拒絕（這是程式錯誤，請回報）'
              : error instanceof Error
                ? error.message
                : '在地化失敗';
      setOutcome({ kind: 'error', message });
    },
  });

  if (!hasFilePath) return null;

  if (outcome) {
    return (
      <ResultPill
        outcome={outcome}
        onGoToKeys={() => navigate({ to: '/settings/keys' })}
        // CR H2: an error pill with no way back stranded the user until a page
        // reload. Clearing the outcome brings the button back.
        onRetry={() => setOutcome(null)}
      />
    );
  }

  const handleOpenChange = (next: boolean) => {
    if (!next && localize.isPending) return;
    if (next) {
      // CR M1: the per-episode option is cost-sensitive (one LLM call per
      // episode). Every opening starts from Sally's ruled default — unchecked —
      // rather than remembering a tick from a dialog the user cancelled.
      setIncludeEpisodes(false);
    }
    setOpen(next);
  };

  return (
    // CR H3: the opener MUST be a DialogTrigger. Radix's modal content restores
    // focus on close to `context.triggerRef` and NOTHING else (react-dialog
    // dist/index.mjs:148) — a plain <button> leaves that ref null, so Escape
    // dropped keyboard users onto <body>. Tested: focus returns to the trigger.
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <button
          type="button"
          data-testid="action-localize-nfo"
          className={cn(
            'flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-secondary)] px-4 text-sm font-medium text-[var(--text-primary)] transition-colors hover:bg-[var(--bg-tertiary)]',
            className
          )}
        >
          <Languages className="h-4 w-4" aria-hidden="true" />
          在地化資訊
        </button>
      </DialogTrigger>

      {/* Radix gives focus trap, initial focus, focus restore and Escape for free. */}
      <DialogContent data-testid="nfo-localize-dialog">
        <DialogHeader>
          <DialogTitle>將資訊在地化為繁體中文</DialogTitle>
          <DialogDescription>
            {isMovie
              ? 'Vido 會用 AI 把片名、劇情與角色名翻成繁體中文，寫成播放器讀得到的 .nfo 檔。'
              : 'Vido 會用 AI 把劇名、簡介與角色名翻成繁體中文。'}
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-4 py-2">
          {isMovie ? (
            <Note tone="success" icon={ShieldCheck}>
              不會覆寫你現有的 .nfo —— 會寫進另一個播放器同樣認得的檔名
            </Note>
          ) : (
            <Note tone="warning" icon={TriangleAlert} align="start">
              <span className="font-semibold">
                影集只有一個檔名可用，這會覆寫現有的 tvshow.nfo。
              </span>
              <span className="font-normal">
                原始檔會先備份成 tvshow.nfo.orig；之後再執行也不會覆蓋這份備份。
              </span>
            </Note>
          )}

          <Note tone="info" icon={Sparkles}>
            會使用 AI 翻譯額度
          </Note>

          {!isMovie && (
            // Unchecked by default (Sally, 9R-UX): a checked box would let a
            // first attempt cost 24x. "Do one, then decide" is the cheap path.
            <div className="flex items-start gap-3 pt-1">
              <input
                id="nfo-include-episodes"
                type="checkbox"
                checked={includeEpisodes}
                onChange={(e) => setIncludeEpisodes(e.target.checked)}
                data-testid="nfo-include-episodes"
                aria-describedby="nfo-include-episodes-cost"
                className="mt-0.5 size-[18px] shrink-0 cursor-pointer rounded-[4px] accent-[var(--accent-primary)]"
              />
              <div className="flex flex-col gap-1">
                <label
                  htmlFor="nfo-include-episodes"
                  className="cursor-pointer text-sm font-semibold text-[var(--text-primary)]"
                >
                  連同每一集的集名與劇情
                </label>
                {/* Tied to the input via aria-describedby so the cost warning is
                      announced with the control, not stranded as loose text. */}
                <span id="nfo-include-episodes-cost" className="text-xs text-[var(--text-muted)]">
                  每一集各翻譯一次，額度用量會明顯增加。
                </span>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <button
            type="button"
            onClick={() => setOpen(false)}
            disabled={localize.isPending}
            className="flex h-10 items-center justify-center rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-[18px] text-sm font-semibold text-[var(--text-primary)] disabled:opacity-60"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => localize.mutate()}
            disabled={localize.isPending}
            data-testid="nfo-confirm"
            className="flex h-10 items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-[18px] text-sm font-semibold text-[var(--text-on-accent)] disabled:opacity-60"
          >
            {localize.isPending && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
            {/* Never a vague 「確定」 — the TV button names both actions. */}
            {isMovie ? '開始在地化' : '備份並覆寫'}
          </button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function Note({
  tone,
  icon: Icon,
  align = 'center',
  children,
}: {
  tone: 'success' | 'info' | 'warning';
  icon: typeof ShieldCheck;
  align?: 'center' | 'start';
  children: React.ReactNode;
}) {
  const tones = {
    success: 'bg-[var(--success-tint)] text-[var(--success)]',
    info: 'bg-[var(--info-tint)] text-[var(--info)]',
    warning: 'bg-[var(--warning-tint)] text-[var(--warning)]',
  } as const;
  return (
    <div
      className={cn(
        'flex gap-2 rounded-[var(--radius-md)] p-3 text-[13px] font-medium',
        align === 'start' ? 'items-start' : 'items-center',
        tones[tone]
      )}
    >
      <Icon className="mt-0.5 h-[18px] w-[18px] shrink-0" aria-hidden="true" />
      <span className="flex flex-col gap-1">{children}</span>
    </div>
  );
}

function ResultPill({
  outcome,
  onGoToKeys,
  onRetry,
}: {
  outcome: Outcome;
  onGoToKeys: () => void;
  onRetry: () => void;
}) {
  const common = { role: 'status' as const, 'aria-live': 'polite' as const };

  if (outcome.kind === 'disabled') {
    return (
      <span
        {...common}
        data-testid="nfo-result-disabled"
        className={cn(pillBase, 'bg-[var(--error-tint)] text-[var(--error)]')}
      >
        <KeyRound className="h-[18px] w-[18px]" aria-hidden="true" />
        尚未設定翻譯服務 ·{' '}
        <button type="button" onClick={onGoToKeys} className="underline underline-offset-2">
          前往設定
        </button>
      </span>
    );
  }

  if (outcome.kind === 'error') {
    return (
      <span
        {...common}
        data-testid="nfo-result-error"
        className={cn(pillBase, 'bg-[var(--error-tint)] text-[var(--error)]')}
      >
        <TriangleAlert className="h-[18px] w-[18px]" aria-hidden="true" />
        {outcome.message} ·{' '}
        <button
          type="button"
          onClick={onRetry}
          data-testid="nfo-retry"
          className="underline underline-offset-2"
        >
          重試
        </button>
      </span>
    );
  }

  if (outcome.kind === 'batch') {
    // `skipped` = the database knows the episode but its video file is not on
    // disk, so there is nowhere to put the .nfo. Not a failure — hence a
    // separate word from 失敗.
    const parts = [`${outcome.succeeded} 集完成`];
    if (outcome.skipped > 0) parts.push(`${outcome.skipped} 集略過`);
    if (outcome.failed > 0) parts.push(`${outcome.failed} 集失敗`);
    const clean = outcome.skipped === 0 && outcome.failed === 0;
    return (
      <span
        {...common}
        data-testid="nfo-result-batch"
        className={cn(
          pillBase,
          clean
            ? 'bg-[var(--success-tint)] text-[var(--success)]'
            : 'bg-[var(--warning-tint)] text-[var(--warning)]'
        )}
      >
        {clean ? (
          <Check className="h-[18px] w-[18px]" aria-hidden="true" />
        ) : (
          <TriangleAlert className="h-[18px] w-[18px]" aria-hidden="true" />
        )}
        影集資訊已更新 · {parts.join('、')}
      </span>
    );
  }

  return (
    <span
      {...common}
      data-testid="nfo-result-ok"
      className={cn(pillBase, 'bg-[var(--success-tint)] text-[var(--success)]')}
    >
      <Check className="h-[18px] w-[18px]" aria-hidden="true" />
      {outcome.replaced ? '已覆寫，原檔已備份為 .nfo.orig' : '已寫入繁中資訊'}
    </span>
  );
}
