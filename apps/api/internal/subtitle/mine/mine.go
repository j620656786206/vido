package mine

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Term is one learned rendering.
type Term struct {
	Src string `json:"src"`
	Zh  string `json:"zh"`
	// Support is how many aligned segments carried BOTH the English term and
	// the Chinese rendering; Segments is how many carried the English term.
	Support  int `json:"support"`
	Segments int `json:"segments"`
	// How is "known" (confirmed against a rendering the caller already had,
	// e.g. the TMDb-seeded glossary) or "cooccurrence" (learned from the
	// subtitles alone).
	How string `json:"how"`
}

// Options tune the miner. The zero value is the production default.
type Options struct {
	// MinSegments is how many aligned segments an English candidate must
	// appear in before it is worth learning (AC #3: ≥ 3).
	MinSegments int
	// MinSupport is how many of those segments must contain the chosen
	// Chinese rendering (AC #3: ≥ 3).
	MinSupport int
	// MinShare is the fraction of the candidate's segments the rendering must
	// cover. Keeps a rendering that happens to sit in three of thirty lines
	// from being learned.
	MinShare float64
	// Known maps English terms the caller already trusts (TMDb cast seeded
	// by sub-7-3, the show's confirmed glossary) to their rendering. Those are
	// verified against the subtitles first, with a lower bar.
	Known map[string]string
}

func (o Options) withDefaults() Options {
	if o.MinSegments <= 0 {
		o.MinSegments = 3
	}
	if o.MinSupport <= 0 {
		o.MinSupport = 3
	}
	if o.MinShare <= 0 {
		o.MinShare = 0.6
	}
	return o
}

// Mine learns renderings from aligned segments. Zero cost: no model, no
// network; proper nouns are capitalised English token runs that are not
// sentence-initial, and the rendering is the 2–4 character Han substring that
// keeps appearing in exactly the segments the English term appears in.
func Mine(segments []Segment, opts Options) []Term {
	opts = opts.withDefaults()
	occurrences := map[string][]int{} // candidate → segment indexes
	for i, seg := range segments {
		for _, c := range candidates(seg.En) {
			occurrences[c] = appendUnique(occurrences[c], i)
		}
	}
	// Han substring frequency over ALL segments, so a rendering must be
	// specific to its term rather than a common word.
	global := map[string]int{}
	for _, seg := range segments {
		for sub := range hanSubstrings(seg.Zh) {
			global[sub]++
		}
	}

	var out []Term
	// Known renderings are trusted already; they are only VERIFIED against
	// the subtitles (did the professionals use the same one?), by a plain
	// word search — the candidate rules exist to find names, not to doubt
	// names we were handed.
	knownSrc := map[string]struct{}{}
	for src, zh := range opts.Known {
		if strings.TrimSpace(src) == "" || strings.TrimSpace(zh) == "" {
			continue
		}
		knownSrc[strings.ToLower(src)] = struct{}{}
		segs, support := 0, 0
		for _, seg := range segments {
			if !mentionsWord(seg.En, src) {
				continue
			}
			segs++
			if strings.Contains(seg.Zh, zh) {
				support++
			}
		}
		if support >= 1 {
			out = append(out, Term{Src: src, Zh: zh, Support: support, Segments: segs, How: "known"})
		}
	}
	for src, idxs := range occurrences {
		if _, known := knownSrc[strings.ToLower(src)]; known {
			continue
		}
		if len(idxs) < opts.MinSegments {
			continue
		}
		local := map[string]int{}
		bounded := map[string]int{}
		for _, i := range idxs {
			for sub, standsAlone := range hanSubstringsWithBoundary(segments[i].Zh) {
				local[sub]++
				if standsAlone {
					bounded[sub]++
				}
			}
		}
		best, bestCount := "", 0
		for sub, n := range local {
			if n < opts.MinSupport || float64(n) < opts.MinShare*float64(len(idxs)) {
				continue
			}
			// Specific to this term: at least half of its global occurrences
			// are in this term's segments.
			if float64(n) < 0.5*float64(global[sub]) {
				continue
			}
			// Ties, in order: the substring that more often stands ALONE
			// between punctuation (「嘿，華特，」— names are addressed and
			// listed; phrases run into their neighbours), then the more
			// SPECIFIC one (fewer occurrences elsewhere — 「托比」 beats
			// 「去問托比」 once 去問 shows up in other lines), then the longer
			// one (a full name beats its own prefix).
			if n > bestCount || (n == bestCount && (bounded[sub] > bounded[best] ||
				(bounded[sub] == bounded[best] && (global[sub] < global[best] ||
					(global[sub] == global[best] && utf8.RuneCountInString(sub) > utf8.RuneCountInString(best)))))) {
				best, bestCount = sub, n
			}
		}
		if best == "" {
			continue
		}
		out = append(out, Term{Src: src, Zh: best, Support: bestCount, Segments: len(idxs), How: "cooccurrence"})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Support != out[j].Support {
			return out[i].Support > out[j].Support
		}
		return out[i].Src < out[j].Src
	})
	return out
}

// ─── English candidates ─────────────────────────────────────────────────────

var tokenRe = regexp.MustCompile(`[A-Za-z][A-Za-z'’-]*`)

// stopwords are capitalised words that are not names. Keep it to the words
// that actually show up capitalised mid-sentence in dialogue: pronoun "I",
// titles, days and the like.
var stopwords = map[string]struct{}{
	"i": {}, "i'm": {}, "i've": {}, "i'll": {}, "i'd": {}, "ok": {}, "okay": {}, "tv": {}, "dna": {}, "fbi": {}, "cia": {}, "nsa": {}, "nypd": {}, "lapd": {}, "usa": {}, "u.s": {}, "uk": {},
	"mr": {}, "mrs": {}, "ms": {}, "dr": {}, "sir": {}, "ma'am": {}, "god": {}, "jesus": {}, "christ": {}, "lord": {},
	"monday": {}, "tuesday": {}, "wednesday": {}, "thursday": {}, "friday": {}, "saturday": {}, "sunday": {},
	"january": {}, "february": {}, "march": {}, "april": {}, "may": {}, "june": {}, "july": {}, "august": {}, "september": {}, "october": {}, "november": {}, "december": {},
	"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "but": {}, "of": {}, "to": {}, "in": {}, "on": {}, "at": {}, "for": {}, "with": {}, "by": {}, "from": {}, "no": {}, "yes": {}, "not": {}, "it": {}, "it's": {}, "that": {}, "this": {}, "what": {}, "who": {}, "why": {}, "how": {}, "when": {}, "where": {}, "you": {}, "he": {}, "she": {}, "we": {}, "they": {}, "me": {}, "my": {}, "your": {}, "his": {}, "her": {}, "our": {}, "their": {}, "oh": {}, "hey": {}, "well": {}, "so": {}, "if": {}, "then": {}, "now": {}, "just": {}, "all": {}, "come": {}, "go": {}, "get": {}, "let": {}, "let's": {}, "look": {}, "listen": {}, "wait": {}, "stop": {}, "please": {}, "thanks": {}, "thank": {}, "sorry": {}, "right": {}, "yeah": {}, "hello": {}, "hi": {}, "bye": {}, "good": {}, "great": {}, "fine": {}, "sure": {}, "really": {}, "maybe": {}, "because": {}, "don't": {}, "can't": {}, "won't": {}, "didn't": {}, "doesn't": {}, "isn't": {}, "there": {}, "here": {}, "one": {}, "two": {}, "three": {},
}

// candidates returns the proper-noun candidates in one English segment:
// runs of 1–4 capitalised tokens, minus stopwords and acronyms. The FIRST
// token of a sentence is capitalised because it starts the sentence, not
// because it is a name, so a sentence-initial run loses its first token
// ("Ask Toby" → Toby, "Walter is here" → nothing); a first name that only
// ever starts sentences is recovered through the Known map (TMDb cast) or by
// its mid-sentence occurrences.
func candidates(en string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, sentence := range splitSentences(en) {
		toks := tokenRe.FindAllStringIndex(sentence, -1)
		var run []string
		flush := func(initial bool) {
			if len(run) == 0 {
				return
			}
			if initial {
				run = run[1:]
			}
			if len(run) > 0 && len(run) <= 4 {
				c := strings.Join(run, " ")
				if _, dup := seen[c]; !dup {
					seen[c] = struct{}{}
					out = append(out, c)
				}
			}
			run = nil
		}
		runStartsSentence := false
		for ti, span := range toks {
			tok := sentence[span[0]:span[1]]
			if isCapitalised(tok) && !isStopword(tok) && !allUpper(tok) {
				if len(run) == 0 {
					runStartsSentence = ti == 0
				}
				run = append(run, tok)
				continue
			}
			flush(runStartsSentence)
		}
		flush(runStartsSentence)
	}
	return out
}

var sentenceSplit = regexp.MustCompile(`[.!?…]+\s+|\s+-\s+|\s*[—–]\s*`)

func splitSentences(s string) []string {
	parts := sentenceSplit.Split(s, -1)
	out := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isCapitalised(tok string) bool {
	r, _ := utf8.DecodeRuneInString(tok)
	return unicode.IsUpper(r) && len(tok) > 1
}

func allUpper(tok string) bool {
	letters := 0
	for _, r := range tok {
		if unicode.IsLetter(r) {
			letters++
			if !unicode.IsUpper(r) {
				return false
			}
		}
	}
	return letters > 1
}

func isStopword(tok string) bool {
	_, ok := stopwords[strings.ToLower(strings.Trim(tok, "'’-"))]
	return ok
}

// mentionsWord is a whole-word, case-insensitive match of term inside text.
func mentionsWord(text, term string) bool {
	lt, lterm := strings.ToLower(text), strings.ToLower(strings.TrimSpace(term))
	if lterm == "" {
		return false
	}
	for start := 0; ; {
		i := strings.Index(lt[start:], lterm)
		if i < 0 {
			return false
		}
		i += start
		end := i + len(lterm)
		before := i == 0 || !isWordByte(lt[i-1])
		after := end == len(lt) || !isWordByte(lt[end])
		if before && after {
			return true
		}
		start = i + 1
	}
}

func isWordByte(c byte) bool {
	return c == '_' || c == '\'' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func appendUnique(list []int, v int) []int {
	if len(list) > 0 && list[len(list)-1] == v {
		return list
	}
	return append(list, v)
}

// ─── Chinese side ───────────────────────────────────────────────────────────

// hanSubstrings yields every 2–4 character run of Han characters in zh, each
// counted once per segment. Punctuation and Latin letters break runs, so a
// substring never spans two words that happen to sit next to each other
// across a comma.
func hanSubstrings(zh string) map[string]struct{} {
	out := map[string]struct{}{}
	for sub := range hanSubstringsWithBoundary(zh) {
		out[sub] = struct{}{}
	}
	return out
}

// hanSubstringsWithBoundary is hanSubstrings plus, per substring, whether it
// occurred at least once as a WHOLE run — bounded by punctuation, Latin text
// or the string edges on both sides.
func hanSubstringsWithBoundary(zh string) map[string]bool {
	out := map[string]bool{}
	var run []rune
	flush := func() {
		for i := 0; i < len(run); i++ {
			for n := 2; n <= 4 && i+n <= len(run); n++ {
				sub := string(run[i : i+n])
				whole := i == 0 && i+n == len(run)
				out[sub] = out[sub] || whole
			}
		}
		run = run[:0]
	}
	for _, r := range zh {
		if unicode.Is(unicode.Han, r) {
			run = append(run, r)
			continue
		}
		flush()
	}
	flush()
	return out
}
