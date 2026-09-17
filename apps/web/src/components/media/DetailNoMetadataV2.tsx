// Design ref: ux-design.pen Screen B10p-D (p3qEc) + Screen B11p-D (V56cx) + Screen B10p-M (H2MRl) + Screen B11p-M (idN42)
/**
 * The detail page for a library item that has no metadata yet (dsr-2b-b AC #2 /
 * #3). Replaces nothing — it sits under the hero, above 檔案資訊 — because the
 * file itself is real and its local facts are worth showing.
 *
 * - `failed` (比對失敗): automatic matching ran and found no work. The way back is
 *   picking the right one (手動選片); re-matching is honest about mostly helping
 *   after a temporary TMDb/network failure — AI filename parses are cached, so a
 *   parse that was wrong stays wrong.
 * - `pending` (資料整理中): nothing is necessarily running — matching only runs
 *   after a scan that found changes — so NOTHING here moves until the user asks
 *   for a match (DESIGN.md: a moving element claims work is running).
 *
 * Presentational: the container owns the re-match mutation and passes its
 * state down.
 */
import { Loader2, RefreshCw, Search } from 'lucide-react';

export type NoMetadataVariant = 'failed' | 'pending';

interface DetailNoMetadataV2Props {
  variant: NoMetadataVariant;
  mediaType: 'movie' | 'series';
  onManualMatch: () => void;
  onRematch: () => void;
  /** A re-match is in flight — the only time this block shows motion. */
  rematching: boolean;
  /** The last re-match started from this page ran and still found nothing. */
  lastRematch?: 'still-failed' | null;
  /** The last re-match request failed (not "found nothing" — the request itself). */
  rematchError?: unknown;
}

const primaryButton =
  'inline-flex min-h-[44px] items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--accent-primary)] px-5 text-sm font-medium text-[var(--text-on-accent)] transition-colors hover:bg-[var(--accent-pressed)] aria-disabled:cursor-wait aria-disabled:opacity-70';
const secondaryButton =
  'inline-flex min-h-[44px] items-center justify-center gap-2 rounded-[var(--radius-md)] bg-[var(--bg-tertiary)] px-5 text-sm font-medium text-[var(--text-primary)] transition-opacity hover:opacity-90 aria-disabled:cursor-wait aria-disabled:opacity-70';

/** What to tell the user about a failed re-match request, and whether to show its code. */
function rematchErrorCopy(error: unknown): { sentence: string; code?: string } {
  const { status, code } = (error ?? {}) as { status?: number; code?: string };
  if (code === 'ENRICHMENT_ALREADY_RUNNING' || status === 409) {
    return { sentence: '媒體庫正在比對其他檔案，請稍後再試。' };
  }
  if (code === 'METADATA_TIMEOUT' || status === 504) {
    return { sentence: '比對花太久，已經停止。請稍後再試。' };
  }
  return { sentence: '比對失敗，請稍後再試。', code };
}

export function DetailNoMetadataV2({
  variant,
  mediaType,
  onManualMatch,
  onRematch,
  rematching,
  lastRematch,
  rematchError,
}: DetailNoMetadataV2Props) {
  const failed = variant === 'failed';
  const noun = mediaType === 'series' ? '影集' : '電影';
  const errorCopy = rematchError ? rematchErrorCopy(rematchError) : null;

  const rematchButton = (
    <button
      type="button"
      // aria-disabled, not disabled: a disabled button drops keyboard focus the
      // moment it is pressed.
      onClick={rematching ? undefined : onRematch}
      aria-disabled={rematching}
      data-testid="no-metadata-rematch"
      className={failed ? secondaryButton : primaryButton}
    >
      {rematching ? (
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
      ) : (
        <RefreshCw className="h-4 w-4" aria-hidden="true" />
      )}
      {rematching ? '比對中…' : failed ? '重新比對' : '立即比對'}
    </button>
  );

  return (
    <section
      data-testid="detail-no-metadata"
      data-variant={variant}
      className="rounded-[var(--radius-lg)] bg-[var(--bg-secondary)] p-4 sm:p-6"
    >
      <h2 className="text-lg font-semibold text-[var(--text-primary)]">
        {failed ? `沒有找到這部${noun}的資料` : '這部片的資料還在整理'}
      </h2>
      <p className="mt-1 text-[var(--text-secondary)]">
        {failed
          ? '自動比對沒有找到符合的作品。你可以自己選對的那一部。'
          : '新加入的檔案會在下次掃描後自動比對。不想等的話，可以現在就比對。'}
      </p>

      <div className="mt-4 flex flex-wrap items-center gap-2">
        {failed && (
          <button
            type="button"
            onClick={onManualMatch}
            data-testid="no-metadata-manual-match"
            className={primaryButton}
          >
            <Search className="h-4 w-4" aria-hidden="true" />
            手動選片
          </button>
        )}
        {rematchButton}
      </div>

      {lastRematch === 'still-failed' && !rematching && (
        <p role="status" className="mt-3 text-sm text-[var(--text-primary)]">
          重新比對完成，還是沒有找到。
        </p>
      )}

      {errorCopy && !rematching && (
        <div
          role="alert"
          className="mt-3 flex flex-wrap items-center gap-2 text-sm text-[var(--error-text)]"
        >
          <span>{errorCopy.sentence}</span>
          {errorCopy.code ? (
            <span
              data-testid="no-metadata-error-code"
              className="rounded-[var(--radius-sm)] bg-[var(--bg-tertiary)] px-2 py-0.5 font-mono text-[11px] text-[var(--text-muted)]"
            >
              {errorCopy.code}
            </span>
          ) : null}
        </div>
      )}

      {failed && (
        <p className="mt-3 text-xs text-[var(--text-muted)]">
          上次如果是網路或 TMDb 暫時出錯，可以再比對一次。
          <br />
          也可以用上方的「修改資訊」自己填片名與年份。
        </p>
      )}
    </section>
  );
}
