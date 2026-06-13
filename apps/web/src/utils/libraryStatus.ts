/**
 * Library item status derivation (UX Redesign Phase 2 — UX2-2, N1 / §2.5).
 *
 * Renders the one truthful lifecycle on poster/list/detail from the fields a
 * library item actually carries. NOTE (Rule 24 triage): movies/series expose
 * `parseStatus` ('success' | 'pending' | 'failed') + `subtitleTracks` (a JSON
 * string of tracks) — there is NO item-level `subtitleStatus`/`downloadProgress`
 * field, so the richer process states (下載中·% / 簡轉繁 / AI 校正中) are NOT
 * derivable at list scope and are a Phase-3 backend addition. We derive what's
 * truthful today: in-library lifecycle + subtitle availability. Tints/text use
 * the §2.5 token classes; an unknown/degraded state returns null (badge absent,
 * never an error — F3).
 */
import type { LibraryMovie, LibrarySeries } from '../types/library';

export interface StatusDescriptor {
  label: string;
  /** Tailwind token classes: tint background + AA-safe text color (§2.5). */
  className: string;
}

const TINT = {
  success: 'bg-[var(--success-tint)] text-[var(--success)]',
  accent: 'bg-[var(--accent-tint)] text-[var(--accent-text)]',
  warning: 'bg-[var(--warning-tint)] text-[var(--warning)]',
  error: 'bg-[var(--error-tint)] text-[var(--error-text)]',
  info: 'bg-[var(--info-tint)] text-[var(--info)]',
  neutral: 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]',
} as const;

type Media = Pick<LibraryMovie | LibrarySeries, 'parseStatus' | 'subtitleTracks'>;

/** Lifecycle badge from `parseStatus`. An in-library item with a clean parse is 已入庫. */
export function deriveLifecycleStatus(media: Media | undefined): StatusDescriptor | null {
  if (!media) return null;
  switch (media.parseStatus) {
    case 'success':
      return { label: '已入庫', className: TINT.success };
    case 'pending':
      return { label: '整理中', className: TINT.warning };
    case 'failed':
      return { label: '失敗', className: TINT.error };
    default:
      return null; // unknown → no badge (F3)
  }
}

interface SubtitleTrack {
  language?: string;
  lang?: string;
}

/**
 * Subtitle badge from `subtitleTracks` JSON. Distinguishes 繁中 (zh-Hant present)
 * / 簡中 (only zh-Hans) / 缺字幕 (no zh subtitle). Returns null if the field is
 * absent (status genuinely unknown) — distinct from 缺字幕 (known-missing).
 */
export function deriveSubtitleStatus(media: Media | undefined): StatusDescriptor | null {
  if (!media || media.subtitleTracks === undefined) return null;

  let tracks: SubtitleTrack[] = [];
  try {
    const parsed = JSON.parse(media.subtitleTracks);
    if (Array.isArray(parsed)) tracks = parsed;
  } catch {
    // Non-JSON legacy value → can't classify reliably; treat as unknown.
    return null;
  }

  const langs = tracks.map((t) => (t.language || t.lang || '').toLowerCase());
  const hasHant = langs.some(
    (l) => l === 'zh-hant' || l === 'zh-tw' || l === 'zh' || l === 'zh-hk'
  );
  const hasHans = langs.some((l) => l === 'zh-hans' || l === 'zh-cn');

  if (hasHant) return { label: '繁中', className: TINT.success };
  if (hasHans) return { label: '簡中', className: TINT.accent };
  if (langs.length > 0) return { label: '有字幕', className: TINT.neutral };
  return { label: '缺字幕', className: TINT.neutral };
}
