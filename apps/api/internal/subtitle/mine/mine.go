package mine

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// maxHanRunes caps a rendering's length. Full names run long (凱利根將軍,
// 艾利娜史塔科夫) and a 4-character cap truncated them in the Shadow and
// Bone run (「凱利根將」, 「利娜史塔科夫」); 8 keeps those whole without
// letting phrases through.
const maxHanRunes = 8

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
	// Candidate DISCOVERY uses the capitalisation rules (mid-sentence runs);
	// the segment set a candidate is then scored on comes from a plain
	// whole-word search over every segment — including the lines where the
	// name starts the sentence. Without that second pass a vocative name
	// ("Walter, we need you.") is invisible to the scorer but still counted
	// against the rendering's specificity, and nothing ever clears the bar
	// (Scorpion S01: 華特 in 23 of 86 lines, 0 terms learned).
	discovered := map[string]struct{}{}
	for _, seg := range segments {
		for _, c := range candidates(seg.En) {
			discovered[c] = struct{}{}
		}
	}
	// Common-word guard: a NAME is never written in lower case. A single-word
	// candidate that also shows up lower-cased in the dialogue ("Tell",
	// "Ready", "Welcome", "Focus" after a dash or a mis-split sentence) is a
	// word, not a name, and is dropped before scoring.
	lowerCounts := map[string]int{}
	for _, seg := range segments {
		for _, tok := range tokenRe.FindAllString(seg.En, -1) {
			if r, _ := utf8.DecodeRuneInString(tok); unicode.IsLower(r) {
				lowerCounts[strings.ToLower(strings.Trim(tok, "'’-"))]++
			}
		}
	}
	occurrences := make(map[string][]int, len(discovered)) // candidate → segment indexes
	for c := range discovered {
		c = strings.TrimSuffix(strings.TrimSuffix(c, "'s"), "’s")
		if c == "" {
			continue
		}
		if !strings.Contains(c, " ") && lowerCounts[strings.ToLower(c)] > 0 {
			continue
		}
		if _, done := occurrences[c]; done {
			continue
		}
		for i, seg := range segments {
			if mentionsWord(seg.En, c) {
				occurrences[c] = append(occurrences[c], i)
			}
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
		atStart := map[string]int{} // seen at the start of a Han run
		atEnd := map[string]int{}   // seen at the end of a Han run
		for _, i := range idxs {
			for sub, e := range hanSubstringsWithEdges(segments[i].Zh) {
				local[sub]++
				if e.Whole {
					bounded[sub]++
				}
				if e.AtStart {
					atStart[sub]++
				}
				if e.AtEnd {
					atEnd[sub]++
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
					(global[sub] == global[best] && (utf8.RuneCountInString(sub) > utf8.RuneCountInString(best) ||
						(utf8.RuneCountInString(sub) == utf8.RuneCountInString(best) && sub < best))))))) {
				best, bestCount = sub, n
			}
		}
		if best == "" {
			continue
		}
		// disc-2026-10-mine-cross-season-rendering-split: when the official
		// translators spelled one name two ways across seasons (See:
		// 謝拉馬威 in S1, 傑拉馬瑞爾 in S2), the piece they share — 拉馬 —
		// outscores either full name and won. Grow the winner into the
		// superstring that explains at least half of its occurrences and
		// stands alone no less often, so the majority FULL rendering is
		// learned instead of the fragment.
		best, bestCount = growToFullRendering(best, bestCount, opts.MinSupport, local, bounded, atStart, atEnd, global)
		if best == "" {
			continue // a two-sided fragment nothing could complete: 拉馬 alone teaches nothing
		}
		out = append(out, Term{Src: src, Zh: best, Support: bestCount, Segments: len(idxs), How: "cooccurrence"})
	}
	out = dropSubTerms(out)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Support != out[j].Support {
			return out[i].Support > out[j].Support
		}
		return out[i].Src < out[j].Src
	})
	return out
}

// growToFullRendering climbs from a winning FRAGMENT to the full rendering.
//
// A fragment is a substring with a side that is always glued to more Han
// text in this term's lines: never seen at the start of a Han run (拉馬,
// 拉公主, 斯人 — something always precedes), or never at the end. A word that
// has been seen at both edges (托比 in 「托比在哪」 and 「去問托比」) is complete
// and is left alone, vocative or not. Growth extends only towards the glued
// side(s), to the superstring that accounts for at least half of the ORIGINAL
// fragment's occurrences and clears MinSupport and the specificity gate,
// preferring one that stands alone somewhere (a name), then the better
// supported, then the longer, then the lexically smaller (deterministic).
// Repeats until the result is complete or nothing qualifies. A two-sided
// fragment that cannot be completed returns "" — it is not worth learning.
// PURE.
func growToFullRendering(best string, bestCount, minSupport int, local, bounded, atStart, atEnd, global map[string]int) (string, int) {
	origin := bestCount
	for {
		gluedLeft, gluedRight := atStart[best] == 0, atEnd[best] == 0
		if !gluedLeft && !gluedRight {
			return best, bestCount
		}
		next, nextCount := "", 0
		for sub, n := range local {
			if len(sub) <= len(best) {
				continue
			}
			switch {
			case gluedLeft && gluedRight:
				if !strings.Contains(sub, best) {
					continue
				}
			case gluedLeft:
				if !strings.HasSuffix(sub, best) {
					continue
				}
			default:
				if !strings.HasPrefix(sub, best) {
					continue
				}
			}
			if 2*n < origin || n < minSupport {
				continue
			}
			// Specific to this term, same gate the first pick cleared.
			if float64(n) < 0.5*float64(global[sub]) {
				continue
			}
			if next == "" || bounded[sub] > bounded[next] || (bounded[sub] == bounded[next] &&
				(n > nextCount || (n == nextCount && (utf8.RuneCountInString(sub) > utf8.RuneCountInString(next) ||
					(utf8.RuneCountInString(sub) == utf8.RuneCountInString(next) && sub < next))))) {
				next, nextCount = sub, n
			}
		}
		if next == "" {
			if gluedLeft && gluedRight {
				return "", 0
			}
			return best, bestCount
		}
		best, bestCount = next, nextCount
	}
}

// dropSubTerms removes a single-word term whose rendering is exactly a
// multi-word term's rendering that contains the word ("Rollins → 佩卡羅林斯"
// next to "Pekka Rollins → 佩卡羅林斯"): the surname alone did not earn the
// full name, it only ever appeared inside it.
func dropSubTerms(terms []Term) []Term {
	multi := map[string]string{} // rendering → multi-word src
	for _, t := range terms {
		if strings.Contains(t.Src, " ") {
			multi[t.Zh] = t.Src
		}
	}
	out := terms[:0]
	for _, t := range terms {
		if !strings.Contains(t.Src, " ") {
			if full, ok := multi[t.Zh]; ok && mentionsWord(full, t.Src) {
				continue
			}
		}
		out = append(out, t)
	}
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

// A dialogue dash glued to the next word ("-Shut up.") starts a sentence
// too; the word after it is capitalised for that reason alone.
var (
	sentenceSplit = regexp.MustCompile(`[.!?…]+\s+|\s+-\s+|\s*[—–]\s*`)
	dashGlued     = regexp.MustCompile(`(^|\s)-(\S)`)
)

func splitSentences(s string) []string {
	s = dashGlued.ReplaceAllString(s, "$1 - $2")
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
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// ─── Chinese side ───────────────────────────────────────────────────────────

// A rendering never starts with a pronoun, preposition or verb particle, and
// never ends with a sentence particle or the possessive 的 — those are the
// characters that sit NEXT to a name in a sentence (「跟佩卡」「佩卡的」) and
// would otherwise ride along into the learned term.
var (
	particleStart = runeSet("我你他她它您們的了是在和跟把被給讓叫去找對從向與及或也都就還很")
	particleEnd   = runeSet("的了嗎呢啊吧呀喔哦啦嘛欸耶哇喂")
)

func runeSet(s string) map[rune]bool {
	out := map[rune]bool{}
	for _, r := range s {
		out[r] = true
	}
	return out
}

// hanSubstrings yields every 2–6 character run of Han characters in zh, each
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

// hanEdges says where a substring sat inside the Han runs of one line: as a
// WHOLE run (bounded by punctuation, Latin text or the string edges on both
// sides), at the start of a run, at the end of a run.
type hanEdges struct{ Whole, AtStart, AtEnd bool }

// hanSubstringsWithBoundary is hanSubstrings plus, per substring, whether it
// occurred at least once as a WHOLE run.
func hanSubstringsWithBoundary(zh string) map[string]bool {
	out := map[string]bool{}
	for sub, e := range hanSubstringsWithEdges(zh) {
		out[sub] = e.Whole
	}
	return out
}

// hanSubstringsWithEdges is hanSubstrings plus, per substring, the edges it
// touched at least once (disc-2026-10-mine-cross-season-rendering-split: a
// fragment of a name never touches one of its edges).
func hanSubstringsWithEdges(zh string) map[string]hanEdges {
	out := map[string]hanEdges{}
	var run []rune
	flush := func() {
		for i := 0; i < len(run); i++ {
			for n := 2; n <= maxHanRunes && i+n <= len(run); n++ {
				if particleStart[run[i]] || particleEnd[run[i+n-1]] {
					continue
				}
				sub := string(run[i : i+n])
				e := out[sub]
				e.AtStart = e.AtStart || i == 0
				e.AtEnd = e.AtEnd || i+n == len(run)
				e.Whole = e.Whole || (i == 0 && i+n == len(run))
				out[sub] = e
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
