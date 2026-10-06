/**
 * Library item status derivation (UX Redesign — N1 / §2.5).
 *
 * Renders the one truthful lifecycle on poster/list from the fields a library item
 * carries. As of ux3-0-1 the list exposes the AUTHORITATIVE subtitle-engine result
 * (`subtitleStatus` + `subtitleLanguage`) alongside `parseStatus` and the embedded
 * `subtitleTracks`. We derive the durable states truthfully: in-library lifecycle
 * (整理中 / 已入庫 / 失敗) + Chinese-subtitle availability (繁中 / 簡中 / 中文 /
 * 缺中文 / 無字幕源 / 已略過 / 未翻譯). Since disc-2026-10-subtitle-filter-disagrees-
 * with-badges the "has Chinese?" half is the backend's `chineseSubtitle` verdict —
 * the same rule the library filter matches on — and is never re-derived here.
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

// Text ALWAYS wears the *-text AA variant — the raw family colours measure
// sub-AA as text (styles.css documents 4.31:1 worst case for --warning; the
// critique R1 probe measured 整理中 at 3.73:1 on its own tint-over-tertiary).
// The *-text tokens are held ≥AA on tint-composited surfaces by
// styles-contrast.spec.ts.
const TINT = {
  success: 'bg-[var(--success-tint)] text-[var(--success-text)]',
  accent: 'bg-[var(--accent-tint)] text-[var(--accent-text)]',
  warning: 'bg-[var(--warning-tint)] text-[var(--warning-text)]',
  error: 'bg-[var(--error-tint)] text-[var(--error-text)]',
  info: 'bg-[var(--info-tint)] text-[var(--info-text)]',
  neutral: 'bg-[var(--bg-tertiary)] text-[var(--text-muted)]',
} as const;

type Media = Pick<
  LibraryMovie | LibrarySeries,
  'parseStatus' | 'subtitleTracks' | 'subtitleStatus' | 'subtitleLanguage' | 'chineseSubtitle'
>;

/** Lifecycle badge from `parseStatus`. An in-library item with a clean parse is 已入庫 (steady). */
export function deriveLifecycleStatus(media: Media | undefined): StatusDescriptor | null {
  if (!media) return null;
  switch (media.parseStatus) {
    case 'success':
      return { label: '已入庫', className: TINT.success, steadyState: true };
    case 'pending':
      // ⚖️ Alexyu 2026-09-11（dsr-11）：泥金，不是赭。赭色的意思是「你要求了，但它
      //「沒發生」；整理中正在發生，那是泥金「正在跑」。七月 13-0 立的「暫態處理中家族」
      // 被 2026-09-10／11 的固定詞彙裁定取代——赭色的價值來自零誤報，不是涵蓋得廣。
      return { label: '整理中', className: TINT.accent };
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

export type SubtitleLangFamily = 'hant' | 'hans' | 'en' | 'other';

/**
 * One subtitle-language tag → its display label (story dsr-6b AC #3).
 *
 * The ONE mapping for every surface that names a track's language — the detail
 * page's 檔案資訊 row and the 管理字幕 dialog's pills used to carry two copies
 * that disagreed (`eng`/`und` shown raw in the dialog). Case-insensitive: the
 * dialog receives raw values such as the `zh-Hant` the pipeline writes. Anything
 * unrecognised stays exactly as the file stated it — `chi`/`zho` are in neither
 * script set, so no script is invented for them.
 */
export function subtitleLangLabel(lang: string): { label: string; family: SubtitleLangFamily } {
  const l = lang.toLowerCase();
  if (HANT.has(l)) return { label: '繁中', family: 'hant' };
  if (HANS.has(l)) return { label: '簡中', family: 'hans' };
  if (l === 'en' || l === 'eng' || l.startsWith('en-')) return { label: '英文', family: 'en' };
  if (l === '' || l === 'und') return { label: '未標示', family: 'other' };
  return { label: lang, family: 'other' };
}

/**
 * Lowercased embedded-track language tags. `null` when `subtitleTracks` is
 * absent or a non-JSON legacy value (can't classify → unknown); an empty array
 * when the field parses but lists no tracks. Exported (dsr-2) so the detail
 * page's 字幕軌 fact row reads tracks through the same parser as the badges —
 * do NOT write a second one.
 */
export function trackLangs(media: Pick<Media, 'subtitleTracks'>): string[] | null {
  if (media.subtitleTracks === undefined) return null;

  let tracks: SubtitleTrack[] = [];
  try {
    const parsed = JSON.parse(media.subtitleTracks);
    if (Array.isArray(parsed)) tracks = parsed;
  } catch {
    return null; // non-JSON legacy value → can't classify → unknown
  }

  return tracks.map((t) => (t.language || t.lang || '').toLowerCase());
}

/**
 * Subtitle badge (disc-2026-10-subtitle-filter-disagrees-with-badges AC #7).
 *
 * Reads the backend's `chineseSubtitle` verdict — the SAME function the library's
 * 有中文字幕／缺中文字幕／不知道 filter matches on — so every title the 有 chip lists
 * wears 繁中 / 簡中 / 中文 and every title the 缺 chip lists wears 缺中文 (or the
 * pipeline's reason: 無字幕源 / 已略過 / 未翻譯). Do NOT infer "has Chinese" from
 * `subtitleTracks` here (AC #3): an absent verdict (old fixture, older API) is
 * unknown → no badge. Returns null when genuinely unknown (badge absent — F3).
 */
export function deriveSubtitleStatus(media: Media | undefined): StatusDescriptor | null {
  if (!media) return null;

  // 1. Has Chinese — the verdict wins over every pipeline state: a title that
  // already has Chinese is never a "missing" badge, and it is in the 有 chip.
  switch (media.chineseSubtitle) {
    case 'zh_hant':
      return { label: '繁中', className: TINT.success, steadyState: true };
    case 'zh_hans':
      // 簡中 = static informational state → info tint (F1-D-v2 pill C8lUe + DL-v2
      // §2.5: accent is reserved for in-progress states — Sally gate 2026-07-05).
      return { label: '簡中', className: TINT.info };
    case 'zh':
      // ⚖️ D2 (Alexyu 2026-10-06): Chinese whose script the file does not say (an
      // embedded `chi` track). It IS "has" — steady like 繁中, so the poster stays
      // quiet; the list row and detail header name it.
      return { label: '中文', className: TINT.success, steadyState: true };
  }

  // 2. Not "has": the subtitle-pipeline verdicts say WHY it is missing and what
  // the next step is (sub-1-2's enum; spec: flow-j-specs/j2-d). They outrank the
  // plain 缺中文 because they name the recovery path.
  //
  // The three in-flight values return null: normal progress is not an
  // exception, so the grid stays quiet and the Activity hub + `subtitle_progress`
  // SSE (sub-1-6) own them.
  switch (media.subtitleStatus) {
    case 'no_text_source':
      // Distinct from 缺中文 by recovery, not by meaning: this file has no text to
      // extract at all, so only P2 ASR can change it (sub-1-7a AC #4).
      return { label: '無字幕源', className: TINT.neutral };
    case 'skipped':
      return { label: '已略過', className: TINT.neutral };
    case 'untranslated':
      // sub-2-2b AC #2: a generated subtitle exists but the EXPECTED translation
      // step did not run. Names the MISSING STEP — the user's next action (set
      // the key, re-run). Neutral: not an error, and accent stays reserved for
      // in-progress (Sally gate 2026-07-05).
      return { label: '未翻譯', className: TINT.neutral };
    case 'probing':
    case 'extracting':
    case 'translating':
      return null;
  }

  // 3. Known to have no Chinese (English only, nothing, Cantonese only, or
  // searched online and not found) — ⚖️ D2 wording: say 中文 out loud.
  if (media.chineseSubtitle === 'none') return { label: '缺中文', className: TINT.neutral };

  // 4. Genuinely unknown.
  return null;
}

/**
 * The single poster badge (ux3-0-2, N1 §2.5). The badge is an EXCEPTION signal:
 * a lifecycle exception (整理中 / 失敗) wins; otherwise a subtitle exception
 * (缺中文 / 簡中 / 無字幕源 / 已略過 / 未翻譯). The happy steady states (已入庫 +
 * 繁中 or 中文) and unknown states render NO badge — avoiding always-on
 * info-noise on the grid.
 */
export function pickPosterBadge(media: Media | undefined): StatusDescriptor | null {
  const lifecycle = deriveLifecycleStatus(media);
  if (lifecycle && !lifecycle.steadyState) return lifecycle;
  const subtitle = deriveSubtitleStatus(media);
  if (subtitle && !subtitle.steadyState) return subtitle;
  return null;
}
