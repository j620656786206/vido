/**
 * Library item status derivation (UX Redesign — N1 / §2.5).
 *
 * Renders the one truthful lifecycle on poster/list from the fields a library item
 * carries. As of ux3-0-1 the list exposes the AUTHORITATIVE subtitle-engine result
 * (`subtitleStatus` + `subtitleLanguage`) alongside `parseStatus` and the embedded
 * `subtitleTracks`. We derive the durable states truthfully: in-library lifecycle
 * (整理中 / 已入庫 / 失敗) + subtitle availability (繁中 / 簡中 / 有字幕 / 缺字幕).
 *
 * The transient process states (簡轉繁 / AI 校正中 / probing / extracting /
 * translating) are surfaced by the Activity hub, NOT this badge. NOTE the reason
 * changed: it used to be "ephemeral, no persisted per-item field", but sub-1-2 made
 * probing/extracting/translating real `subtitle_status` column values. The rule
 * survives on the ORIGINAL principle instead — the poster badge is an EXCEPTION
 * signal (ux3-0-2) and normal progress is not an exception (ruled in sub-1-7a,
 * spec screen `flow-j-specs/j2-d`). 下載中·% is not derivable for a library item
 * (Epic 13/14). Tints/text use the §2.5 token classes; an unknown/degraded state
 * returns null (badge absent, never an error — F3).
 */
import type { LibraryMovie, LibrarySeries } from '../types/library';

export interface StatusDescriptor {
  label: string;
  /** Tailwind token classes: tint background + AA-safe text color (§2.5). */
  className: string;
  /**
   * True for the "happy" steady state (已入庫 / 繁中). The poster badge is an
   * EXCEPTION signal (ux3-0-2): steady states are suppressed on the grid to avoid
   * always-on info-noise. Other surfaces (detail) may still render them.
   */
  steadyState?: boolean;
}

const TINT = {
  success: 'bg-[var(--success-tint)] text-[var(--success)]',
  accent: 'bg-[var(--accent-tint)] text-[var(--accent-text)]',
  warning: 'bg-[var(--warning-tint)] text-[var(--warning)]',
  error: 'bg-[var(--error-tint)] text-[var(--error-text)]',
  info: 'bg-[var(--info-tint)] text-[var(--info)]',
  neutral: 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]',
} as const;

type Media = Pick<
  LibraryMovie | LibrarySeries,
  'parseStatus' | 'subtitleTracks' | 'subtitleStatus' | 'subtitleLanguage'
>;

/** Lifecycle badge from `parseStatus`. An in-library item with a clean parse is 已入庫 (steady). */
export function deriveLifecycleStatus(media: Media | undefined): StatusDescriptor | null {
  if (!media) return null;
  switch (media.parseStatus) {
    case 'success':
      return { label: '已入庫', className: TINT.success, steadyState: true };
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
 * Canonical zh-script classification (lowercased BCP-47-ish tags). Exported so
 * every subtitle surface (badges here, ManageSubtitleDialogV2 track pills, §9b
 * CN-policy display) classifies 繁/簡 identically — do NOT redeclare locally.
 */
export const HANT = new Set(['zh-hant', 'zh-tw', 'zh', 'zh-hk']);
export const HANS = new Set(['zh-hans', 'zh-cn']);

/** Subtitle badge from embedded file tracks (`subtitleTracks` JSON). */
function deriveFromTracks(media: Media): StatusDescriptor | null {
  if (media.subtitleTracks === undefined) return null;

  let tracks: SubtitleTrack[] = [];
  try {
    const parsed = JSON.parse(media.subtitleTracks);
    if (Array.isArray(parsed)) tracks = parsed;
  } catch {
    return null; // non-JSON legacy value → can't classify → unknown
  }

  const langs = tracks.map((t) => (t.language || t.lang || '').toLowerCase());
  if (langs.some((l) => HANT.has(l)))
    return { label: '繁中', className: TINT.success, steadyState: true };
  // 簡中 = static informational state → info tint (F1-D-v2 pill C8lUe + DL-v2 §2.5:
  // accent is reserved for in-progress states — Sally gate ruling 2026-07-05).
  if (langs.some((l) => HANS.has(l))) return { label: '簡中', className: TINT.info };
  if (langs.length > 0) return { label: '有字幕', className: TINT.neutral };
  return { label: '缺字幕', className: TINT.neutral };
}

/**
 * Subtitle badge. Prefers the AUTHORITATIVE subtitle-engine result
 * (`subtitleStatus` + `subtitleLanguage`, exposed to the list by ux3-0-1): a
 * downloaded zh-Hant subtitle is 繁中 (the happy steady state), a confirmed
 * not_found is 缺字幕. Falls back to embedded-track inference when the engine has
 * no terminal result (not_searched / searching / absent). Returns null when
 * genuinely unknown (badge absent, never errors — F3).
 */
export function deriveSubtitleStatus(media: Media | undefined): StatusDescriptor | null {
  if (!media) return null;

  // 1. Authoritative downloaded-subtitle result.
  if (media.subtitleStatus === 'found') {
    const lang = (media.subtitleLanguage || '').toLowerCase();
    if (HANT.has(lang)) return { label: '繁中', className: TINT.success, steadyState: true };
    if (HANS.has(lang)) return { label: '簡中', className: TINT.info };
    // found but language unknown → defer to embedded tracks, else "有字幕".
    return deriveFromTracks(media) ?? { label: '有字幕', className: TINT.neutral };
  }

  // 2. Subtitle-pipeline verdicts (sub-1-2's 9-value enum; spec: flow-j-specs/j2-d).
  //
  // These sit ABOVE track inference ON PURPOSE — do not "simplify" them to the
  // bottom of the ladder. Both terminal values describe a file that HAS tracks
  // the pipeline cannot use: `no_text_source` typically carries image-only
  // (PGS/VobSub) tracks, and `skipped` carries a real text track with an `und` or
  // non-English tag. Step 3 returns on ANY track, so from below it these items
  // would infer 有字幕 and the engine's authoritative verdict would lose to a
  // naive track count — the user would see "has subtitles" on a file that will
  // never get one.
  //
  // The three in-flight values return null for the same reason: falling through
  // to track inference would badge a mid-run item with a stale track guess. They
  // are normal progress, not an exception, so the grid stays quiet and the
  // Activity hub + `subtitle_progress` SSE (sub-1-6) own them.
  switch (media.subtitleStatus) {
    case 'no_text_source':
      // Distinct from 缺字幕 by recovery, not by meaning: 缺字幕 was searched for
      // online and may be found later; this file has no text to extract at all,
      // so only P2 ASR can change it (sub-1-7a AC #4).
      return { label: '無字幕源', className: TINT.neutral };
    case 'skipped':
      return { label: '已略過', className: TINT.neutral };
    case 'probing':
    case 'extracting':
    case 'translating':
      return null;
  }

  // 3. Embedded tracks (covers not_searched / searching / absent subtitleStatus).
  const fromTracks = deriveFromTracks(media);
  if (fromTracks) return fromTracks;

  // 4. Engine searched and found nothing, no embedded tracks → known-missing.
  if (media.subtitleStatus === 'not_found') return { label: '缺字幕', className: TINT.neutral };

  // 5. Genuinely unknown.
  return null;
}

/**
 * The single poster badge (ux3-0-2, N1 §2.5). The badge is an EXCEPTION signal:
 * a lifecycle exception (整理中 / 失敗) wins; otherwise a subtitle exception
 * (缺字幕 / 簡中 / 有字幕). The happy steady state (已入庫 + 繁中) and unknown
 * states render NO badge — avoiding always-on info-noise on the grid.
 */
export function pickPosterBadge(media: Media | undefined): StatusDescriptor | null {
  const lifecycle = deriveLifecycleStatus(media);
  if (lifecycle && !lifecycle.steadyState) return lifecycle;
  const subtitle = deriveSubtitleStatus(media);
  if (subtitle && !subtitle.steadyState) return subtitle;
  return null;
}
