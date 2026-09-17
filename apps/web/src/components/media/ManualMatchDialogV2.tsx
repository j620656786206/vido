// Design ref: ux-design.pen Screen B12p-D (GnBxR) + Screen B12p-M (xb6fV)
/**
 * 手動選片 — pick the right TMDb work for a library item the scanner could not
 * match (dsr-2b-b AC #4). Applying writes it onto the row as the user's own
 * choice (dsr-2b-a AC #1): title, poster, overview replace what is there, and
 * automatic matching never overwrites it again — the confirm line says so.
 *
 * Locked on purpose: TMDb only (it is the only source that can be applied) and
 * the item's own type (movie ids and TV ids are two numbering systems). The v1
 * `manual-search/ManualSearchDialog` keeps its pickers for the dev-only
 * /test/manual-search page, whose e2e is bound to that DOM.
 *
 * Mobile renders the same Radix dialog as a bottom sheet (B12p-M), like
 * ManageSubtitleDialogV2.
 */
import { useEffect, useRef, useState } from 'react';
import { Loader2, Search } from 'lucide-react';
import { useDebouncedCallback } from 'use-debounce';
import { Dialog, DialogContent, DialogTitle } from '../ui/Dialog';
import { cn } from '../../lib/utils';
import { useApplyMetadata, useManualSearch } from '../../hooks/useManualSearch';
import type { ManualSearchResultItem } from '../../services/metadata';
import { filenameToGradient } from './ColorPlaceholder';
import { fallbackInitial } from '../../utils/fallbackInitial';

export interface ManualMatchDialogV2Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mediaId: string;
  mediaType: 'movie' | 'series';
  /** Already cleaned for search (see utils/cleanFilenameForSearch). */
  initialQuery: string;
}

export function ManualMatchDialogV2({ open, onOpenChange, ...rest }: ManualMatchDialogV2Props) {
  // Radix returns focus to a DialogTrigger; this dialog is opened by a plain
  // button elsewhere on the page, so remember what had focus and go back to it
  // (when it still exists — after a successful apply the opener is gone).
  const openerRef = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (open && document.activeElement instanceof HTMLElement) {
      openerRef.current = document.activeElement;
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        data-testid="manual-match-dialog"
        aria-describedby={undefined}
        onCloseAutoFocus={(event) => {
          const opener = openerRef.current;
          if (opener?.isConnected) {
            event.preventDefault();
            opener.focus();
          }
        }}
        className={cn(
          'flex max-h-[85vh] flex-col gap-0 overflow-hidden p-0',
          // Mobile: bottom sheet (B12p-M). Desktop: centered dialog (B12p-D).
          'bottom-0 left-0 right-0 top-auto w-full max-w-none translate-x-0 translate-y-0 rounded-b-none rounded-t-[var(--radius-xl)]',
          'sm:bottom-auto sm:left-1/2 sm:right-auto sm:top-1/2 sm:w-[calc(100vw-4rem)] sm:max-w-2xl sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-[var(--radius-lg)]'
        )}
      >
        {/* Mounted only while open (Radix renders no content when closed), so a
            closed dialog never searches. */}
        <ManualMatchBody onClose={() => onOpenChange(false)} {...rest} />
      </DialogContent>
    </Dialog>
  );
}

function applyErrorCopy(error: unknown): { sentence: string; code?: string } {
  const { status, code } = (error ?? {}) as { status?: number; code?: string };
  if (code === 'ENRICHMENT_ALREADY_RUNNING' || status === 409) {
    return { sentence: '媒體庫正在比對其他檔案，請稍後再試。' };
  }
  if (code === 'TMDB_NOT_FOUND') {
    return { sentence: 'TMDb 上找不到這部作品，請換一筆。' };
  }
  return { sentence: '套用失敗，請稍後再試。', code };
}

function ManualMatchBody({
  mediaId,
  mediaType,
  initialQuery,
  onClose,
}: Omit<ManualMatchDialogV2Props, 'open' | 'onOpenChange'> & { onClose: () => void }) {
  const [query, setQuery] = useState(initialQuery);
  const [debouncedQuery, setDebouncedQuery] = useState(initialQuery);
  const [selected, setSelected] = useState<ManualSearchResultItem | null>(null);
  const debounce = useDebouncedCallback((value: string) => setDebouncedQuery(value), 300);
  const resultType = mediaType === 'series' ? 'tv' : 'movie';

  const search = useManualSearch({ query: debouncedQuery, mediaType: resultType, source: 'tmdb' });
  const apply = useApplyMetadata();

  // A new query invalidates the pick — its row may no longer be listed — and any
  // error about the previous pick.
  const resetApply = apply.reset;
  useEffect(() => {
    setSelected(null);
    resetApply();
  }, [debouncedQuery, resetApply]);

  const pick = (item: ManualSearchResultItem) => {
    setSelected(item);
    apply.reset();
  };

  const results = search.data?.results ?? [];
  const noun = mediaType === 'series' ? '影集' : '電影';

  const handleApply = async () => {
    if (!selected) return;
    try {
      await apply.mutateAsync({
        mediaId,
        mediaType,
        selectedItem: { id: selected.id, source: selected.source, mediaType: selected.mediaType },
      });
      onClose();
    } catch {
      // Shown below from apply.error.
    }
  };

  const errorCopy = apply.error ? applyErrorCopy(apply.error) : null;
  const selectedName = selected ? selected.titleZhTw || selected.title : '';

  return (
    <>
      <div className="flex h-14 shrink-0 items-center border-b border-[var(--border-subtle)] pl-6 pr-12">
        <DialogTitle className="text-base font-semibold">手動選片</DialogTitle>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-4 sm:p-6">
        <div className="relative">
          <Search
            className="absolute left-3 top-1/2 h-5 w-5 -translate-y-1/2 text-[var(--text-muted)]"
            aria-hidden="true"
          />
          <input
            type="search"
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              debounce(e.target.value);
            }}
            aria-label={`在 TMDb 搜尋${noun}`}
            placeholder={`輸入${noun}名稱`}
            className="min-h-[44px] w-full rounded-[var(--radius-md)] border border-[var(--border-subtle)] bg-[var(--bg-primary)] py-2 pl-10 pr-4 [&::-webkit-search-cancel-button]:appearance-none text-[var(--text-primary)] placeholder-[var(--text-muted)] focus:border-transparent focus:outline-none focus:ring-2 focus:ring-[var(--accent-primary)]"
          />
        </div>

        <p className="text-sm text-[var(--text-secondary)]">
          TMDb 上的{noun}
          {search.data ? ` · ${results.length} 筆` : ''}
        </p>

        {search.isLoading ? (
          <p aria-busy="true" className="flex items-center gap-2 text-sm text-[var(--text-muted)]">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            搜尋中…
          </p>
        ) : search.isError ? (
          <p className="text-sm text-[var(--error-text)]">搜尋暫時無法使用，請稍後再試。</p>
        ) : search.data && results.length === 0 ? (
          <p className="text-sm text-[var(--text-secondary)]">找不到符合的作品，換個關鍵字試試。</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {results.map((item) => (
              <li key={item.id}>
                <ResultRow
                  item={item}
                  selected={selected?.id === item.id}
                  onSelect={() => pick(item)}
                />
              </li>
            ))}
          </ul>
        )}
      </div>

      <div className="flex shrink-0 flex-col gap-3 border-t border-[var(--border-subtle)] p-4 sm:flex-row sm:items-center sm:px-6">
        {selected && (
          <p
            data-testid="manual-match-confirm"
            className="flex-1 text-sm text-[var(--text-secondary)]"
          >
            套用「{selectedName}
            {selected.year ? `（${selected.year}）` : ''}
            」？這會用它的片名、海報、簡介取代目前的資料，之後自動比對不會再改它。
          </p>
        )}
        <div className="flex flex-wrap items-center justify-end gap-2 sm:ml-auto">
          {errorCopy && !apply.isPending && (
            <div
              role="alert"
              className="flex w-full flex-wrap items-center gap-2 text-sm text-[var(--error-text)] sm:w-auto"
            >
              <span>{errorCopy.sentence}</span>
              {errorCopy.code ? (
                <span
                  data-testid="manual-match-error-code"
                  className="rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-[11px] text-[var(--text-muted)]"
                >
                  {errorCopy.code}
                </span>
              ) : null}
            </div>
          )}
          <button
            type="button"
            onClick={onClose}
            className="min-h-[44px] rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-opacity hover:opacity-90"
          >
            取消
          </button>
          {selected && (
            <button
              type="button"
              onClick={handleApply}
              disabled={apply.isPending}
              data-testid="manual-match-apply"
              className="inline-flex min-h-[44px] items-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] disabled:cursor-wait disabled:opacity-70"
            >
              {apply.isPending && <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />}
              {apply.isPending ? '套用中…' : '確認套用'}
            </button>
          )}
        </div>
      </div>
    </>
  );
}

function ResultRow({
  item,
  selected,
  onSelect,
}: {
  item: ManualSearchResultItem;
  selected: boolean;
  onSelect: () => void;
}) {
  const [posterFailed, setPosterFailed] = useState(false);
  const name = item.titleZhTw || item.title;
  const [from, to] = filenameToGradient(name);

  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      data-testid="manual-match-result"
      className={cn(
        'flex w-full items-center gap-3 rounded-[var(--radius-md)] p-3 text-left transition-colors',
        selected
          ? 'bg-[var(--accent-subtle)] ring-2 ring-inset ring-[var(--accent-primary)]'
          : 'bg-[var(--bg-tertiary)] hover:bg-[var(--bg-primary)]'
      )}
    >
      <div className="aspect-[2/3] w-10 shrink-0 overflow-hidden rounded-[var(--radius-sm)]">
        {item.posterUrl && !posterFailed ? (
          <img
            src={item.posterUrl}
            alt=""
            loading="lazy"
            onError={() => setPosterFailed(true)}
            className="h-full w-full object-cover"
          />
        ) : (
          <div
            data-testid="manual-match-poster-fallback"
            aria-hidden="true"
            className="flex h-full w-full items-center justify-center text-sm font-bold text-[var(--text-on-scrim)]"
            style={{ backgroundImage: `linear-gradient(135deg, ${from}, ${to})` }}
          >
            {fallbackInitial(name)}
          </div>
        )}
      </div>
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium text-[var(--text-primary)]">{name}</p>
        <p className="truncate text-sm text-[var(--text-secondary)]">
          {[item.title !== name ? item.title : null, item.year].filter(Boolean).join(' · ')}
        </p>
      </div>
    </button>
  );
}
