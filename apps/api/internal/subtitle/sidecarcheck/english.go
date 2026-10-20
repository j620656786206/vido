package sidecarcheck

import (
	"strings"
	"unicode"
)

// Cheap English check (AC #4): no language model, no API call.
var (
	// minCues: fewer lines than this is a forced/partial track, not a
	// subtitle to translate (the miner's minUsableCues).
	minCues = 20
	// minLatinShare: of all letters, at least this share are Latin.
	minLatinShare = 0.9
	// minFunctionWordShare: at least this share of cues carry one of the
	// common English function words below.
	minFunctionWordShare = 0.3
)

// functionWords are words almost every stretch of English dialogue has and
// the other Latin-script languages a NAS library holds mostly lack. Words
// that are also Spanish, Dutch, German or French ("a", "me", "in", "is",
// "we", "of", "die") are left out: with them a Spanish subtitle passed.
var functionWords = map[string]bool{
	"the": true, "you": true, "and": true, "that": true, "what": true, "with": true,
	"this": true, "your": true, "don't": true, "it's": true, "i'm": true, "are": true,
	"have": true, "was": true, "he": true, "she": true, "they": true, "not": true,
	"just": true, "know": true, "can't": true, "you're": true, "there": true, "it": true,
}

// maxDiacriticShare: English letters are almost never accented; Spanish,
// French, German and Portuguese dialogue is (ñ, é, ü, ç, ß).
var maxDiacriticShare = 0.01

// englishVerdict reports whether the cue texts read as English, and why not.
// PURE.
func englishVerdict(texts []string) (bool, string) {
	if len(texts) < minCues {
		return false, "too_few_cues"
	}
	var latin, letters, accented, withWord int
	for _, t := range texts {
		hit := false
		for _, w := range strings.FieldsFunc(strings.ToLower(t), func(r rune) bool {
			return !unicode.IsLetter(r) && r != '\'' && r != '’'
		}) {
			if functionWords[strings.ReplaceAll(w, "’", "'")] {
				hit = true
			}
		}
		if hit {
			withWord++
		}
		for _, r := range t {
			if !unicode.IsLetter(r) {
				continue
			}
			letters++
			if unicode.Is(unicode.Latin, r) {
				latin++
				if r > unicode.MaxASCII {
					accented++
				}
			}
		}
	}
	if letters == 0 || float64(latin)/float64(letters) < minLatinShare {
		return false, "not_latin"
	}
	if float64(accented)/float64(letters) > maxDiacriticShare {
		return false, "not_english"
	}
	if float64(withWord)/float64(len(texts)) < minFunctionWordShare {
		return false, "not_english"
	}
	return true, ""
}
