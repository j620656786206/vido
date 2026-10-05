// Package zhtw is the one place a Chinese subtitle is turned into what a
// Taiwanese viewer reads: script first (OpenCC s2twp, 简→繁), then vocabulary
// (the built-in Taiwan lexicon, 質量→品質). Every delivery path calls Finalize
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
type Converter interface {
	ConvertS2TWP(content []byte) ([]byte, error)
}

// IsMainland is the PRD mainland rule's single predicate: content produced in
// mainland China keeps its own vocabulary. Only CN counts; whether HK/MO
// should is an open product question (backlog-mainland-rule-three-predicates).
func IsMainland(countries []string) bool {
	for _, c := range countries {
		if strings.EqualFold(strings.TrimSpace(c), "CN") {
			return true
		}
	}
	return false
}

// Finalize runs the script step, then the vocabulary step — skipped for
// mainland content. When the script step fails, the result is the INPUT run
// through the vocabulary step, returned with the error: callers whose input is
// already Traditional (a gated translation) deliver it; callers for whom the
// conversion IS the deliverable discard it.
func Finalize(conv Converter, text string, countries []string) (string, error) {
	var err error
	if conv != nil {
		var out []byte
		if out, err = conv.ConvertS2TWP([]byte(text)); err == nil {
			text = string(out)
		}
	}
	if !IsMainland(countries) {
		text = prompts.ZhTWLexicon().Apply(text)
	}
	return text, err
}
