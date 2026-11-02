package providers

import (
	"regexp"
	"strconv"
)

// episodeSpanTag matches "S01E02", "S01 E19", "s1.e2" and a range such as
// "S01E01-E22" or "S01E01-22".
var episodeSpanTag = regexp.MustCompile(`(?i)s(\d{1,2})[ ._-]?e(\d{1,3})(?:\s*[-~]\s*e?(\d{1,3}))?`)

// MatchesEpisode reports whether a subtitle named name can hold the given
// episode: a name with no SxxEyy tag (a season pack, a film-style name) can;
// a tagged name can only when its season matches and its episode, or episode
// range, covers the wanted one. Used to keep other episodes out of an
// episode's search results.
func MatchesEpisode(name string, season, episode int) bool {
	tags := episodeSpanTag.FindAllStringSubmatch(name, -1)
	if len(tags) == 0 {
		return true
	}
	for _, m := range tags {
		s, _ := strconv.Atoi(m[1])
		from, _ := strconv.Atoi(m[2])
		to := from
		if m[3] != "" {
			to, _ = strconv.Atoi(m[3])
		}
		if s == season && from <= episode && episode <= to {
			return true
		}
	}
	return false
}
