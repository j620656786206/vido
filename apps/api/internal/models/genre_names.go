package models

import "strings"

// Genre names — disc-2026-10-genre-names-simplified.
//
// TMDb answers genre names in Simplified Chinese for zh-TW, zh-CN and zh-HK
// alike (剧情, 动作冒险 — checked against /genre/{movie,tv}/list on
// 2026-10-12), and has no Chinese at all for two TV genres (Sci-Fi & Fantasy,
// War & Politics). Stored as-is, the library's 類型 filter listed Simplified
// and English names, while the Metadata Editor offered the Taiwan names — so a
// hand edit left 劇情 and 剧情 side by side.
//
// Every write of genres goes through Movie.GenresJSON / Series.GenresJSON, so
// that is where names become the Taiwan wording. The table is the web app's
// GENRE_MAP (apps/web/src/lib/genres.ts) — a test keeps the two identical.

// GenreNamesByTMDbID is each TMDb genre id's Taiwan name (web GENRE_MAP).
var GenreNamesByTMDbID = map[int]string{
	// Movie genres
	28:    "動作",
	12:    "冒險",
	16:    "動畫",
	35:    "喜劇",
	80:    "犯罪",
	99:    "紀錄",
	18:    "劇情",
	10751: "家庭",
	14:    "奇幻",
	36:    "歷史",
	27:    "恐怖",
	10402: "音樂",
	9648:  "懸疑",
	10749: "愛情",
	878:   "科幻",
	10770: "電視電影",
	53:    "驚悚",
	10752: "戰爭",
	37:    "西部",
	// TV genres
	10759: "動作冒險",
	10762: "兒童",
	10763: "新聞",
	10764: "真人秀",
	10765: "科幻奇幻",
	10766: "肥皂劇",
	10767: "脫口秀",
	10768: "戰爭政治",
}

// genreAliases are the other names TMDb (and the old editor) use for each id:
// TMDb's Chinese name (Simplified, or English where it has none) and its
// en-US name. English is matched case-insensitively.
var genreAliases = map[int][]string{
	28:    {"动作", "Action"},
	12:    {"冒险", "Adventure"},
	16:    {"动画", "Animation"},
	35:    {"喜剧", "Comedy"},
	80:    {"Crime"},
	99:    {"纪录", "Documentary"},
	18:    {"剧情", "Drama"},
	10751: {"Family"},
	14:    {"Fantasy"},
	36:    {"历史", "History"},
	27:    {"Horror"},
	10402: {"音乐", "Music"},
	9648:  {"悬疑", "Mystery"},
	10749: {"爱情", "Romance"},
	878:   {"Science Fiction", "sci-fi"}, // sci-fi: the pre-#539 editor's key
	10770: {"电视电影", "TV Movie"},
	53:    {"惊悚", "Thriller"},
	10752: {"战争", "War"},
	37:    {"Western"},
	10759: {"动作冒险", "Action & Adventure"},
	10762: {"儿童", "Kids"},
	10763: {"新闻", "News"},
	10764: {"Reality"},
	10765: {"Sci-Fi & Fantasy"},
	10766: {"肥皂剧", "Soap"},
	10767: {"脱口秀", "Talk"},
	10768: {"War & Politics"},
}

// genreCanonical maps a lower-cased known name to its Taiwan name.
var genreCanonical = func() map[string]string {
	m := make(map[string]string, 3*len(GenreNamesByTMDbID))
	for id, name := range GenreNamesByTMDbID {
		m[strings.ToLower(name)] = name
		for _, alias := range genreAliases[id] {
			m[strings.ToLower(alias)] = name
		}
	}
	return m
}()

// CanonicalGenres returns names in the Taiwan wording: a known TMDb name in
// any of its spellings becomes the Taiwan name, an unknown name is kept as
// given (trimmed), empty names are dropped, and duplicates collapse to the
// first occurrence. Never nil.
func CanonicalGenres(names []string) []string {
	out := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if c, ok := genreCanonical[strings.ToLower(n)]; ok {
			n = c
		}
		if _, dup := seen[n]; dup {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
