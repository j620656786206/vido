// Design ref: ux-design.pen Screen F1-D-v2 (r1EY9) + Screen F1-M-v2 (JkdfH)
/**
 * 畫面字幕 — the 現有字幕 row for Chinese burned into an episode's picture, and
 * its 判斷錯了？ menu (disc-2026-11-burned-in-subs-c2; Sally's decisions in
 * disc-2026-11-burned-in-subs-c-design). No frame of its own — Pencil went paid —
 * so it refers to the 現有字幕 section of F1 it extends.
 *
 * Same row as every other 現有字幕 row (pill · source · detail); the detail is
 * a sentence here, not a file name, so it is not set in Mono. The menu holds
 * the two corrections the backend offers for a whole season:
 *  - 「這季沒有畫面字幕」 → PUT /subtitles/burned-in/season {value: none}, free;
 *  - 「重新檢查這季」 → POST /subtitles/burned-in/check (~US$0.002, said under
 *    the option — DESIGN.md「會花錢的動作要有記號」).
 * Either refetches the season list, which is where the dialog reads the
 * episode from; a season corrected to none simply loses this row. There is
 * deliberately no 「改成有」 (design: no entry until someone needs it).
 *
 * The menu is the Radix DropdownMenu the download actions use, modal={false}
 * for the same reason (a modal menu inside the dialog leaves pointer-events
 * off for a frame).
 */
import { useState } from 'react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { CircleAlert, Loader2 } from 'lucide-react';
import { cn } from '../../lib/utils';
import { detailKeys } from '../../hooks/useMediaDetails';
import { subtitleService } from '../../services/subtitleService';

/** The BurnedInSubtitle values that mean "Chinese is burned in". */
export const BURNED_IN_CHINESE = new Set(['zh-Hans', 'zh-Hant', 'zh']);

/** The detail line under the 畫面字幕 source — the design's words. */
export const BURNED_IN_DETAIL = '燒在畫面上，不能關掉';

/** The generate button's extra line when the picture already has Chinese. */
export const BURNED_IN_GENERATE_NOTE = '畫面上已經有中文字幕，再生成的字幕會跟它疊在一起';

const MENU_ITEM =
  'flex min-h-11 cursor-default select-none flex-col items-start justify-center gap-0.5 rounded-[var(--radius-sm)] px-3 py-2 text-sm font-medium text-[var(--text-primary)] outline-none data-[highlighted]:bg-[var(--accent-subtle)] data-[disabled]:opacity-50';

export interface BurnedInSubtitleRowProps {
  /** The language pill, from the dialog's own descriptor (簡中／繁中／中文). */
  label: string;
  pillClass: string;
  /** Which season the corrections apply to. Absent → the row shows, the menu
   *  does not (nothing to address). */
  seriesId?: string;
  seasonNumber?: number;
}

export function BurnedInSubtitleRow({
  label,
  pillClass,
  seriesId,
  seasonNumber,
}: BurnedInSubtitleRowProps) {
  const queryClient = useQueryClient();
  const [undecided, setUndecided] = useState(false);
  const canCorrect = !!seriesId && seasonNumber !== undefined;
  const target = { seriesId: seriesId ?? '', season: seasonNumber ?? 0 };

  const refreshSeason = () =>
    queryClient.invalidateQueries({
      queryKey: detailKeys.seasonEpisodes(target.seriesId, target.season),
    });

  const clear = useMutation({
    mutationFn: () => subtitleService.clearBurnedInSeason(target),
    onSuccess: () => refreshSeason(),
  });
  const recheck = useMutation({
    mutationFn: () => subtitleService.checkBurnedInSeason(target),
    onSuccess: (res) => {
      setUndecided(res.result === null);
      return refreshSeason();
    },
  });

  const busy = clear.isPending || recheck.isPending;
  const failure = clear.error
    ? `沒辦法改成沒有畫面字幕：${clear.error.message}`
    : recheck.error
      ? `沒辦法重新檢查：${recheck.error.message}`
      : null;

  const run = (which: 'clear' | 'recheck') => {
    setUndecided(false);
    clear.reset();
    recheck.reset();
    if (which === 'clear') clear.mutate();
    else recheck.mutate();
  };

  return (
    <div className="flex flex-col gap-1">
      <div
        data-testid="subtitle-track-burned-in"
        className="flex items-center gap-3 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-3.5 py-3 max-sm:gap-2.5 max-sm:px-3 max-sm:py-2.5"
      >
        <span className={cn('shrink-0 rounded-full px-2.5 py-1 text-xs font-medium', pillClass)}>
          {label}
        </span>
        <span className="shrink-0 text-xs text-[var(--text-secondary)]">畫面字幕</span>
        <span
          data-testid="subtitle-track-detail-burned-in"
          className="min-w-0 flex-1 truncate text-xs text-[var(--text-muted)]"
          title={BURNED_IN_DETAIL}
        >
          {BURNED_IN_DETAIL}
        </span>
        {canCorrect && (
          <DropdownMenu.Root modal={false}>
            <DropdownMenu.Trigger asChild>
              <button
                type="button"
                data-testid="burned-in-correct"
                disabled={busy}
                aria-busy={busy || undefined}
                className="flex min-h-11 shrink-0 items-center gap-1.5 rounded-[var(--radius-sm)] px-2 text-xs font-medium text-[var(--text-secondary)] transition-colors hover:bg-[var(--bg-primary)] hover:text-[var(--text-primary)] disabled:cursor-default disabled:hover:bg-transparent"
              >
                {busy && <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />}
                {recheck.isPending ? '檢查中…' : clear.isPending ? '更新中…' : '判斷錯了？'}
              </button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content
                align="end"
                sideOffset={4}
                className="z-50 min-w-56 rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-tertiary)] p-1.5 shadow-[var(--shadow-lg)]"
              >
                <DropdownMenu.Item
                  data-testid="burned-in-clear"
                  onSelect={() => run('clear')}
                  className={MENU_ITEM}
                >
                  這季沒有畫面字幕
                </DropdownMenu.Item>
                <DropdownMenu.Item
                  data-testid="burned-in-recheck"
                  onSelect={() => run('recheck')}
                  className={MENU_ITEM}
                >
                  重新檢查這季
                  <span className="text-xs font-normal text-[var(--text-muted)]">
                    會花約 US$0.002
                  </span>
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        )}
      </div>
      {failure && (
        <div
          data-testid="burned-in-error"
          role="alert"
          className="flex items-center gap-2 rounded-[var(--radius-md)] bg-[var(--error-tint)] px-3 py-2"
        >
          <CircleAlert className="h-4 w-4 shrink-0 text-[var(--error-text)]" aria-hidden="true" />
          <p className="flex-1 text-xs text-[var(--error-text)]">{failure}</p>
        </div>
      )}
      {undecided && (
        <p data-testid="burned-in-undecided" className="px-1 text-xs text-[var(--text-muted)]">
          這次沒看出結果，先維持原本的判斷
        </p>
      )}
    </div>
  );
}
