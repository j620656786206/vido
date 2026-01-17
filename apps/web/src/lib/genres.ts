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
