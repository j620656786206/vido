// Implements: <utility — no .pen counterpart>
/**
 * The ONE table behind the 字幕 filter (dsr-1b-b AC #2, re-cut by
 * disc-2026-10-subtitle-filter-disagrees-with-badges AC #8). The desktop rail, the
 * phone sheet (its 「套用篩選 · N 部」 count), the chip row and the no-result sentence
 * all read labels from here — four copies of the same strings is how 媒體庫/電影/影集
 * ended up with four maps (disc-2026-09-media-type-label-four-maps).
 *
 * Values are the backend's `chinese_subtitle` groups — confirmed against
 * [@contract-v1] (Story disc-2026-10-subtitle-filter-disagrees-with-badges AC #2):
 * lowercase, comma-joined on the wire. They select on the SAME verdict the poster /
 * list badge reads (`chineseSubtitle`), so 有中文字幕 lists exactly the titles badged
 * 繁中／簡中／中文 and 缺中文字幕 exactly the ones badged 缺中文 (or 無字幕源／已略過／
 * 未翻譯). ⚖️ D1 (Alexyu 2026-10-06): three chips that add up to the whole library.
 *
 * `label` is the button inside the 字幕 section; `chipLabel` is the same choice
 * standing alone (chip row, no-result sentence), where a bare 不知道 would not say
 * what is unknown.
 */
export const CHINESE_SUBTITLE_FILTER_OPTIONS = [
  { value: 'has', label: '有中文字幕', chipLabel: '有中文字幕' },
  { value: 'missing', label: '缺中文字幕', chipLabel: '缺中文字幕' },
  { value: 'unknown', label: '不知道', chipLabel: '不知道有沒有中文字幕' },
] as const;

export type ChineseSubtitleFilterValue = (typeof CHINESE_SUBTITLE_FILTER_OPTIONS)[number]['value'];

const KNOWN = new Set<string>(CHINESE_SUBTITLE_FILTER_OPTIONS.map((o) => o.value));

/** Stand-alone label for a value; unknown values echo the raw value so nothing renders blank. */
export function chineseSubtitleChipLabel(value: string): string {
  return CHINESE_SUBTITLE_FILTER_OPTIONS.find((o) => o.value === value)?.chipLabel ?? value;
}

/**
 * URL/wire csv → selection. Empty/absent → []. Unknown values are dropped (a hand-edited
 * deep link must not reach the wire — the handler answers 400 and the whole page would
 * fall into its error state) and duplicates collapse (one chip per value, one count).
 */
export function parseChineseSubtitleCsv(csv: string | undefined): string[] {
  if (!csv) return [];
  const out: string[] = [];
  for (const raw of csv.split(',')) {
    const v = raw.trim();
    if (v && KNOWN.has(v) && !out.includes(v)) out.push(v);
  }
  return out;
}

/** Selection → URL/wire csv. Empty → undefined so the param leaves the URL. */
export function joinChineseSubtitleCsv(values: string[] | undefined): string | undefined {
  return values && values.length > 0 ? values.join(',') : undefined;
}
