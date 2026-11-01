package zhtw

import (
	"fmt"
	"strings"
)

// ScriptConverter is a Converter that can also go Traditional → Simplified,
// which is how SimplifiedFields tells the two scripts apart.
type ScriptConverter interface {
	Converter
	ConvertT2S(content []byte) ([]byte, error)
}

// fieldSeparator joins many texts into one OpenCC run; plain ASCII that
// neither OpenCC nor the Taiwan lexicon rewrites.
const fieldSeparator = "\n@@vido-field-separator@@\n"

// SimplifiedFields reports, per text, whether it is Simplified Chinese
// (disc-2026-10-metadata-zh-fallback-simplified): s2tw changes it, and t2s
// changes it much less — no character at all, or fewer than half as many as
// s2tw does (a Simplified overview that quotes one Traditional name).
//
// s2tw alone is not enough. It is not a no-op on Traditional text — on the
// production library it turned 蘇里 into 蘇裡, 好傢伙 into 好傢夥 and 神祕 into
// 神秘 — and t2s rewrites Traditional text throughout (傑克蘇里: one s2tw change,
// two t2s changes), so it is never reported. When a conversion changes the
// text's length the counts cannot be compared and only "t2s changes nothing"
// counts. Two OpenCC runs cover all texts.
func SimplifiedFields(conv ScriptConverter, texts []string) ([]bool, error) {
	out := make([]bool, len(texts))
	if len(texts) == 0 {
		return out, nil
	}
	toTrad, err := convertJoined(conv.ConvertS2TW, texts)
	if err != nil {
		return nil, err
	}
	toSimp, err := convertJoined(conv.ConvertT2S, texts)
	if err != nil {
		return nil, err
	}
	for i, t := range texts {
		s2tw, okS := changedRunes(t, toTrad[i])
		t2s, okT := changedRunes(t, toSimp[i])
		switch {
		case toTrad[i] == t:
			out[i] = false
		case toSimp[i] == t:
			out[i] = true
		case okS && okT:
			out[i] = s2tw > 2*t2s
		}
	}
	return out, nil
}

// changedRunes counts the positions where a and b differ; ok is false when
// their lengths differ.
func changedRunes(a, b string) (n int, ok bool) {
	ra, rb := []rune(a), []rune(b)
	if len(ra) != len(rb) {
		return 0, false
	}
	for i := range ra {
		if ra[i] != rb[i] {
			n++
		}
	}
	return n, true
}

// ConvertJoined runs convert once over all texts and splits the result back.
func ConvertJoined(convert func(string) (string, error), texts []string) ([]string, error) {
	out, err := convert(strings.Join(texts, fieldSeparator))
	if err != nil {
		return nil, err
	}
	parts := strings.Split(out, fieldSeparator)
	if len(parts) != len(texts) {
		return nil, fmt.Errorf("conversion changed the field count: want %d, got %d", len(texts), len(parts))
	}
	return parts, nil
}

func convertJoined(convert func([]byte) ([]byte, error), texts []string) ([]string, error) {
	return ConvertJoined(func(s string) (string, error) {
		b, err := convert([]byte(s))
		return string(b), err
	}, texts)
}

// Scripts ClassifyScript tells apart — the BCP 47 tags subtitle tracks use.
const (
	ScriptSimplified  = "zh-Hans"
	ScriptTraditional = "zh-Hant"
	// ScriptUntold: Chinese whose script the text cannot tell — no character
	// that one script writes differently from the other, or a tie.
	ScriptUntold = "zh"
)

// ClassifyScript says which script lines of Chinese are written in, three
// ways: whichever direction OpenCC would rewrite MORE characters in is the
// script the text is NOT in (disc-2026-11-burned-in-subs-b1). Unlike
// SimplifiedFields it also names Traditional, and a tie — including "neither
// direction changes anything" — is ScriptUntold, never a guess.
//
// Counting rather than "does t2s change it at all" is what keeps a Traditional
// line with one stray Simplified character (《白髮》's 静) Traditional: s2tw
// changes that one character, t2s changes every Traditional one around it. The
// burned-in exam (disc-2026-11-burned-in-subs-a) got every decidable title right
// this way; the model guessing the script itself got one wrong.
func ClassifyScript(conv ScriptConverter, lines []string) (string, error) {
	text := strings.Join(lines, "\n")
	if strings.TrimSpace(text) == "" {
		return ScriptUntold, nil
	}
	toTrad, err := conv.ConvertS2TW([]byte(text))
	if err != nil {
		return "", err
	}
	toSimp, err := conv.ConvertT2S([]byte(text))
	if err != nil {
		return "", err
	}
	s2tw, okS := changedRunes(text, string(toTrad))
	t2s, okT := changedRunes(text, string(toSimp))
	switch {
	case !okS || !okT:
		// A conversion that changed the length cannot be counted position by
		// position; saying "Chinese, script untold" is still true.
		return ScriptUntold, nil
	case s2tw > t2s:
		return ScriptSimplified, nil
	case t2s > s2tw:
		return ScriptTraditional, nil
	}
	return ScriptUntold, nil
}
