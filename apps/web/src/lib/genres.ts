export const GENRE_MAP: Record<number, string> = {
  // Movie genres
  28: '動作',
  12: '冒險',
  16: '動畫',
  35: '喜劇',
  80: '犯罪',
  99: '紀錄',
  18: '劇情',
  10751: '家庭',
  14: '奇幻',
  36: '歷史',
  27: '恐怖',
  10402: '音樂',
  9648: '懸疑',
  10749: '愛情',
  878: '科幻',
  10770: '電視電影',
  53: '驚悚',
  10752: '戰爭',
  37: '西部',
  // TV genres
  10759: '動作冒險',
  10762: '兒童',
  10763: '新聞',
  10764: '真人秀',
  10765: '科幻奇幻',
  10766: '肥皂劇',
  10767: '脫口秀',
  10768: '戰爭政治',
};

export function getGenreNames(genreIds: number[], limit = 3): string[] {
  return genreIds
    .slice(0, limit)
    .map((id) => GENRE_MAP[id])
    .filter(Boolean);
}

// TMDb's genre ids per media type (GET /genre/movie/list, /genre/tv/list).
const MOVIE_GENRE_IDS = [
  28, 12, 16, 35, 80, 99, 18, 10751, 14, 36, 27, 10402, 9648, 10749, 878, 10770, 53, 10752, 37,
];
const TV_GENRE_IDS = [
  10759, 16, 35, 80, 99, 18, 10751, 10762, 9648, 10763, 10764, 10765, 10766, 10767, 10768, 37,
];

/**
 * The genre names a hand edit can pick from. These are the SAME strings the
 * library stores: the API rewrites every genre into this table's Taiwan names
 * when it saves (Go `models.CanonicalGenres` — TMDb's own zh-TW names are
 * Simplified), so an edit round-trips. The old editor offered English keys and
 * saved 「drama」 over 「劇情」. A Go test keeps GENRE_MAP and the API's table identical.
 */
export function genreNamesFor(mediaType: 'movie' | 'series'): string[] {
  return (mediaType === 'movie' ? MOVIE_GENRE_IDS : TV_GENRE_IDS).map((id) => GENRE_MAP[id]);
}

/** TMDb genre ids for one content type, in TMDb's own list order (explore blocks). */
export function genreIdsFor(contentType: 'movie' | 'tv'): number[] {
  return contentType === 'movie' ? [...MOVIE_GENRE_IDS] : [...TV_GENRE_IDS];
}

/**
 * A TMDb genre object's name in the Taiwan wording. TMDb's own zh-TW names are
 * Simplified (剧情, 动作冒险) or English (Sci-Fi & Fantasy), so a known id reads
 * from GENRE_MAP; an id this table does not know keeps TMDb's name.
 */
export function tmdbGenreName(genre: { id: number; name: string }): string {
  return GENRE_MAP[genre.id] ?? genre.name;
}

/** A genre id's zh-TW name; an id this table does not know reads 「ID 123」. */
export function genreLabel(id: number): string {
  return GENRE_MAP[id] ?? `ID ${id}`;
}

/**
 * The explore-block wire format — a comma-separated id string ("28,12") —
 * parsed into ids. Blank / non-numeric parts are ignored (hand-typed data
 * from before the chip picker).
 */
export function parseGenreIdList(value: string | null | undefined): number[] {
  if (!value) return [];
  const ids: number[] = [];
  for (const part of value.split(',')) {
    const n = Number(part.trim());
    if (part.trim() !== '' && Number.isInteger(n) && n > 0 && !ids.includes(n)) ids.push(n);
  }
  return ids;
}
