// Package zhtw is the one place a Chinese subtitle is turned into what a
// Taiwanese viewer reads: script first (OpenCC, 简→繁), then vocabulary (the
// built-in Taiwan lexicon, 質量→品質) — or, for titles that keep their own
// wording, characters only. Every delivery path calls Finalize
// instead of chaining the two steps itself, so a path can no longer ship the
// script half without the vocabulary half (backlog-lexicon-on-non-llm-convert-paths).
//
// It sits beside segkey (project-context.md Rule 19): both `subtitle` and
// `services` import it, and `services` may not import `subtitle`, so it may
// depend on nothing heavier than ai/prompts.
package zhtw

import (
	"strings"

	"github.com/vido/api/internal/ai/prompts"
)

// Converter is the script step. *subtitle.Converter, subtitle.VariantConverter
// and services.OpenCCConverter all satisfy it; whether OpenCC is installed is
// the caller's call — pass nil to skip the script step.
//
// The two profiles differ in more than script: s2twp also rewrites mainland
// phrases into Taiwan ones (视频→影片, 信息→資訊, 出租车→計程車); s2tw converts
// characters only.
type Converter interface {
	ConvertS2TWP(content []byte) ([]byte, error)
	ConvertS2TW(content []byte) ([]byte, error)
}

// ownWordingCountries are the productions whose subtitles keep their own
// wording — Traditional characters, but never Taiwan phrases or vocabulary
// (Alexyu rulings 2026-10-05: mainland like a Netflix 陸劇; Hong Kong and
// Macau the same).
var ownWordingCountries = map[string]bool{"CN": true, "HK": true, "MO": true}

// KeepsOwnWording is the single predicate for that rule. Any matching country
// counts, so a co-production with CN qualifies — an open product question
// (disc-2026-10-coproduction-wording-rule).
func KeepsOwnWording(countries []string) bool {
	for _, c := range countries {
		if ownWordingCountries[strings.ToUpper(strings.TrimSpace(c))] {
			return true
		}
	}
	return false
}

// Finalize turns a Chinese subtitle into what the viewer reads. Titles that
// keep their own wording get characters only (s2tw). Everything else gets
// Taiwan script and phrases (s2twp), then the Taiwan lexicon. When the script
// step fails, the result is the INPUT run through the vocabulary step (none
// for own-wording titles), returned with the error: callers whose input is
// already Traditional (a gated translation) deliver it; callers for whom the
// conversion IS the deliverable discard it.
func Finalize(conv Converter, text string, countries []string) (string, error) {
	own := KeepsOwnWording(countries)
	var err error
	if conv != nil {
		convert := conv.ConvertS2TWP
		if own {
			convert = conv.ConvertS2TW
		}
		var out []byte
		if out, err = convert([]byte(text)); err == nil {
			text = string(out)
		}
	}
	if !own {
		text = prompts.ZhTWLexicon().Apply(text)
	}
	return text, err
}
