package prompts

import (
	"fmt"
	"strings"
)

// LocalizationLevel is the OTT-style localization dial (sub-7-4 AC #3). It is
// a matter of TASTE — some viewers love 全聯 in an American sitcom, some hate
// it — so it is a user setting, never a score-driven default (AC #5).
//
// The three levels differ only in a style section appended to the invariant
// system prompt, and the level rides PromptVersion so the segment cache never
// serves an `ott` translation to a `literal` viewer.
type LocalizationLevel string

const (
	// LocalizationLiteral — no localization: generic nouns stay generic (a
	// grocery store is 超市), cultural references are translated, not swapped.
	LocalizationLiteral LocalizationLevel = "literal"
	// LocalizationStandard — Taiwan vocabulary and idiom throughout, but no
	// local brand or slang substituted for a generic noun. The default.
	LocalizationStandard LocalizationLevel = "standard"
	// LocalizationOTT — Netflix / Apple TV+ house style: everyday scenes may
	// use a Taiwanese brand or colloquialism for a generic noun, under the
	// guardrails in the section below.
	LocalizationOTT LocalizationLevel = "ott"

	// DefaultLocalizationLevel is what an unset setting resolves to.
	DefaultLocalizationLevel = LocalizationStandard
)

// LocalizationLevels lists the valid levels in display order.
var LocalizationLevels = []LocalizationLevel{LocalizationLiteral, LocalizationStandard, LocalizationOTT}

// ParseLocalizationLevel validates a user/env supplied value. Empty resolves
// to the default; anything else unknown is an error (never silently default a
// typo — the user asked for a taste and would get another).
func ParseLocalizationLevel(s string) (LocalizationLevel, error) {
	v := LocalizationLevel(strings.ToLower(strings.TrimSpace(s)))
	if v == "" {
		return DefaultLocalizationLevel, nil
	}
	for _, known := range LocalizationLevels {
		if v == known {
			return known, nil
		}
	}
	return "", fmt.Errorf("unknown localization level %q (want one of literal, standard, ott)", s)
}

// Normalized returns the level itself when valid, else the default — for
// call sites that have already logged the problem and must keep translating.
func (l LocalizationLevel) Normalized() LocalizationLevel {
	if v, err := ParseLocalizationLevel(string(l)); err == nil {
		return v
	}
	return DefaultLocalizationLevel
}

// BuildLocalizationSection renders the level's style rules and examples. The
// examples are deliberately concrete: the model copies what it is shown far
// more reliably than what it is told.
func BuildLocalizationSection(level LocalizationLevel) string {
	var sb strings.Builder
	sb.WriteString("## Localization style (")
	sb.WriteString(string(level.Normalized()))
	sb.WriteString("):\n")
	switch level.Normalized() {
	case LocalizationLiteral:
		sb.WriteString(`Translate what is said, in Taiwan Traditional Chinese, without localizing the world of the story.
- Generic nouns stay generic: "the grocery store" → 超市, "the convenience store" → 便利商店, "the pharmacy" → 藥局.
- Cultural references are translated or kept, never swapped for a local equivalent: "Thanksgiving dinner" → 感恩節晚餐, "the Super Bowl" → 超級盃.
- Money, units and holidays stay as in the source: "twenty bucks" → 二十美元, "a mile" → 一英里.
- No Taiwanese slang or internet phrasing; neutral spoken register.
`)
	case LocalizationOTT:
		sb.WriteString(`Write the way Netflix and Apple TV+ Taiwan subtitles read: natural, current spoken Taiwanese, with everyday local colour where it helps.
- In an EVERYDAY-LIFE scene, a GENERIC noun may take the word a Taiwanese viewer would say: "grab something at the convenience store" → 去超商買個東西 (小七 is fine in casual banter), "a grocery run" → 去全聯／去超市買菜, "the pharmacy" → 藥局.
- Guardrails: substitute ONLY when the source is a generic noun AND the scene is everyday life. A NAMED brand, place or institution is never replaced with a different one ("Whole Foods" stays Whole Foods, never 全聯; "Walgreens" is Walgreens 藥局, never 屈臣氏).
- Casual dialogue may use light Taiwanese colloquialisms ("that's sick" → 超讚的, "no way" → 不會吧, "dude" → 老兄／欸), matched to the speaker's register; never dated internet slang.
- Money: "twenty bucks" → 二十美元 (or 二十塊 in loose banter when the currency is obvious). Distances and temperatures may be rendered in metric when the exact figure does not matter to the plot.
- Formal, period or non-American settings get NO local colour — the substitution rule is for contemporary everyday scenes only.
`)
	default: // standard
		sb.WriteString(`Taiwan vocabulary and idiom throughout — but no local brand or slang stands in for a generic noun.
- "the grocery store" → 超市 (never 全聯), "the convenience store" → 超商／便利商店 (never 小七), "the pharmacy" → 藥局.
- Everyday phrasing is Taiwanese: "no way" → 不會吧, "that's awesome" → 太棒了, "I'm starving" → 我快餓死了.
- Money and units stay as in the source: "twenty bucks" → 二十美元, "a mile" → 一英里.
- Named brands and institutions keep their Taiwan rendering from the global glossary; do not invent local substitutes.
`)
	}
	sb.WriteString("\n")
	return sb.String()
}

// PromptVersionFor is the cache-key prompt version for a given localization
// level: the pinned base revision, the lexicon version and the level, joined
// so any of the three changing re-keys the segment cache (AC #1/#3).
func PromptVersionFor(level LocalizationLevel) string {
	return SubtitleTranslatorPromptVersion + "+" + LexiconVersion() + "+" + string(level.Normalized())
}

// ComposeInvariantSystemPrompt is the install-wide, show-independent system
// text both translation legs put FIRST: the translator prompt, the
// localization style for the configured level, and the global lexicon terms.
// Everything per-show (media context, show glossary) comes after it, so a
// provider-side prompt-cache breakpoint on this prefix is shared by every
// title on the box.
func ComposeInvariantSystemPrompt(level LocalizationLevel) string {
	return SubtitleTranslatorSystemPrompt + "\n\n" +
		BuildLocalizationSection(level) +
		BuildLexiconTermsSection(ZhTWLexicon().Terms)
}
