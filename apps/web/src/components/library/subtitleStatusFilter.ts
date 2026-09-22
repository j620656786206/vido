// Implements: <utility — no .pen counterpart>
/**
 * The ONE table behind the 字幕 filter (dsr-1b-b AC #2). The sheet, the desktop rail,
 * the chip row and the no-result sentence all read labels from here — four copies of
 * the same three strings is how 媒體庫/電影/影集 ended up with four maps
 * (disc-2026-09-media-type-label-four-maps).
 *
 * Values are backend `subtitle_status` states — confirmed against [@contract-v1]
 * (Story dsr-1b-a AC #1): lowercase, comma-joined on the wire. They are NOT the
 * poster badge's language (繁中/簡中 comes from embedded tracks), which is why the
 * labels talk about state, not language. `skipped` / `untranslated` /
 * `no_text_source` are accepted by the contract but get no chip yet — add a row
 * here and every surface follows.
 */
export const SUBTITLE_STATUS_FILTER_OPTIONS = [
  { value: 'found', label: '有字幕' },
  { value: 'not_found', label: '缺字幕' },
  { value: 'not_searched', label: '還沒搜尋' },
] as const;

export type SubtitleStatusFilterValue = (typeof SUBTITLE_STATUS_FILTER_OPTIONS)[number]['value'];

/**
 * Every value the backend accepts — mirrors `models.AllSubtitleStatuses()` (dsr-1b-a AC #1
 * [@contract-v1]). Wider than the three chips on purpose: a deep link to `skipped` is legal
 * and must survive parsing, while `?subtitleStatus=bogus` must NOT reach the wire (the
 * handler answers 400 and the whole page would fall into its error state).
 */
export const KNOWN_SUBTITLE_STATUSES = [
  'not_searched',
  'searching',
  'found',
  'not_found',
  'probing',
  'extracting',
  'translating',
  'no_text_source',
  'skipped',
  'untranslated',
] as const;
const KNOWN = new Set<string>(KNOWN_SUBTITLE_STATUSES);

/** Label for a status value; unknown values echo the raw value so nothing renders blank. */
export function subtitleStatusLabel(value: string): string {
  return SUBTITLE_STATUS_FILTER_OPTIONS.find((o) => o.value === value)?.label ?? value;
}

/**
 * URL/wire csv → selection. Empty/absent → []. Unknown values are dropped (a hand-edited
 * deep link cannot 400 the list) and duplicates collapse (one chip per value, one count).
 */
export function parseSubtitleStatusCsv(csv: string | undefined): string[] {
  if (!csv) return [];
  const out: string[] = [];
  for (const raw of csv.split(',')) {
    const v = raw.trim();
    if (v && KNOWN.has(v) && !out.includes(v)) out.push(v);
  }
  return out;
}

/** Selection → URL/wire csv. Empty → undefined so the param leaves the URL. */
export function joinSubtitleStatusCsv(values: string[] | undefined): string | undefined {
  return values && values.length > 0 ? values.join(',') : undefined;
}
