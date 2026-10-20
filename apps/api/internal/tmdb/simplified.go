package tmdb

import (
	"log/slog"
	"strings"

	"github.com/vido/api/internal/zhtw"
)

// Simplified → Traditional for borrowed text (disc-2026-10-metadata-zh-fallback-simplified).
//
// When TMDb has no Taiwan text the chain borrows zh-CN, and that text used to
// be stored as is: a fresh library of five Blender films showed 钢铁之泪,
// 寻龙记 and 大象之梦, and Tears of Steel's whole overview in Simplified.
// Every response that comes back in a Simplified language now goes through
// zhtw.Finalize — the same script + wording rule the subtitles follow — before
// it is merged, so a borrowed field is Traditional wherever it lands. zh-TW
// and en answers are never touched: running s2twp over Taiwan text could only
// make it worse.

// keepOwnWording is the country list that makes zhtw.Finalize convert the
// characters only. It is used where a response carries no production country:
// for a list item whose original language is Chinese, and for a season (the
// season response names no country at all — under-converting a foreign
// show's episode names beats rewriting a mainland show's wording).
var keepOwnWording = []string{"CN"}

// isSimplifiedLanguage reports whether a TMDb language code answers in
// Simplified script.
func isSimplifiedLanguage(lang string) bool {
	l := strings.ToLower(strings.TrimSpace(lang))
	return l == "zh-cn" || l == "zh-sg" || strings.HasPrefix(l, "zh-hans")
}

// chineseOriginalLanguage reports whether a list item is itself a Chinese
// production — the stand-in for a production country, which list results lack.
func chineseOriginalLanguage(lang string) bool {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "zh", "cn", "yue":
		return true
	}
	return false
}

// SetChineseConverter wires the OpenCC converter used for Simplified answers.
// Wiring only — call it during startup. Without one, Simplified text is kept
// as TMDb sent it (logged once), never a failed lookup.
func (c *LanguageFallbackClient) SetChineseConverter(conv zhtw.ScriptConverter) {
	c.converter = conv
}

// finalize converts the fields behind ptrs that really are Simplified. A
// zh-CN answer is not always Simplified — TMDb users type Traditional into it
// too — and s2twp over Traditional text breaks names (蘇里→蘇裡), so each field
// is checked first (zhtw.SimplifiedFields). Any failure keeps the original
// text (Rule 13: degrade, log, never fail). Two or three OpenCC runs per call.
func (c *LanguageFallbackClient) finalize(ptrs []*string, countries []string) {
	var fields []*string
	for _, p := range ptrs {
		if p != nil && *p != "" {
			fields = append(fields, p)
		}
	}
	if len(fields) == 0 {
		return
	}
	if c.converter == nil {
		c.warnNoConverter.Do(func() {
			slog.Warn("TMDb answered in Simplified Chinese but no converter is wired — text is stored as sent")
		})
		return
	}

	texts := make([]string, len(fields))
	for i, p := range fields {
		texts[i] = *p
	}
	simplified, err := zhtw.SimplifiedFields(c.converter, texts)
	if err != nil {
		slog.Warn("TMDb Simplified check failed — keeping the original text", "error", err)
		return
	}
	var targets []*string
	var sources []string
	for i, p := range fields {
		if simplified[i] {
			targets = append(targets, p)
			sources = append(sources, texts[i])
		}
	}
	if len(targets) == 0 {
		return
	}
	out, err := zhtw.ConvertJoined(func(text string) (string, error) {
		return zhtw.Finalize(c.converter, text, countries)
	}, sources)
	if err != nil {
		slog.Warn("TMDb Simplified→Traditional conversion failed — keeping the original text", "error", err)
		return
	}
	for i, p := range targets {
		*p = out[i]
	}
}

// The localize* helpers return a converted COPY for a Simplified-language
// response and the response itself otherwise (CR M1 of bugfix-d: never write
// into what the client handed back). nil in, nil out, so they sit straight
// after the client call whatever its error.

func (c *LanguageFallbackClient) localizeMovieDetails(d *MovieDetails, lang string) *MovieDetails {
	if d == nil || !isSimplifiedLanguage(lang) {
		return d
	}
	out := *d
	countries := make([]string, 0, len(out.ProductionCountries))
	for _, pc := range out.ProductionCountries {
		countries = append(countries, pc.ISO31661)
	}
	c.finalize([]*string{&out.Title, &out.Overview, &out.Tagline}, countries)
	return &out
}

func (c *LanguageFallbackClient) localizeTVShowDetails(d *TVShowDetails, lang string) *TVShowDetails {
	if d == nil || !isSimplifiedLanguage(lang) {
		return d
	}
	out := *d
	countries := append([]string(nil), out.OriginCountry...)
	for _, pc := range out.ProductionCountries {
		countries = append(countries, pc.ISO31661)
	}
	c.finalize([]*string{&out.Name, &out.Overview, &out.Tagline}, countries)
	return &out
}

func (c *LanguageFallbackClient) localizeSeasonDetails(d *SeasonDetails, lang string) *SeasonDetails {
	if d == nil || !isSimplifiedLanguage(lang) {
		return d
	}
	out := *d
	out.Episodes = append([]EpisodeInfo(nil), d.Episodes...)
	ptrs := []*string{&out.Name, &out.Overview}
	for i := range out.Episodes {
		ptrs = append(ptrs, &out.Episodes[i].Name, &out.Episodes[i].Overview)
	}
	c.finalize(ptrs, keepOwnWording)
	return &out
}

func (c *LanguageFallbackClient) localizeMovies(r *SearchResultMovies, lang string) *SearchResultMovies {
	if r == nil || !isSimplifiedLanguage(lang) {
		return r
	}
	out := *r
	out.Results = append([]Movie(nil), r.Results...)
	var own, taiwan []*string
	for i := range out.Results {
		m := &out.Results[i]
		if chineseOriginalLanguage(m.OriginalLanguage) {
			own = append(own, &m.Title, &m.Overview)
		} else {
			taiwan = append(taiwan, &m.Title, &m.Overview)
		}
	}
	c.finalize(own, keepOwnWording)
	c.finalize(taiwan, nil)
	return &out
}

func (c *LanguageFallbackClient) localizeTVShows(r *SearchResultTVShows, lang string) *SearchResultTVShows {
	if r == nil || !isSimplifiedLanguage(lang) {
		return r
	}
	out := *r
	out.Results = append([]TVShow(nil), r.Results...)
	var own, taiwan []*string
	for i := range out.Results {
		s := &out.Results[i]
		if zhtw.KeepsOwnWording(s.OriginCountry) || chineseOriginalLanguage(s.OriginalLanguage) {
			own = append(own, &s.Name, &s.Overview)
		} else {
			taiwan = append(taiwan, &s.Name, &s.Overview)
		}
	}
	c.finalize(own, keepOwnWording)
	c.finalize(taiwan, nil)
	return &out
}
