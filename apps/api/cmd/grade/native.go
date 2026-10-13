package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/eval"
)

// Prompt styles (spike-local-translation-golden-eval AC #3).
const (
	// promptStyleProduction drives the model through services.TranslationService
	// — Vido's real translator prompt, [N] batches, 5-cue context — i.e. what
	// plugging the model into the app with no prompt work would give.
	promptStyleProduction = "production"
	// promptStyleNative uses the model's own published template, one cue per
	// call — what a dedicated small-model translation path could give.
	promptStyleNative = "native"
)

// Native templates this spike supports.
const (
	nativeTranslateGemma = "translategemma"
	nativeHyMT           = "hymt"
)

// nativeContextCues is how many preceding source lines ride along as context
// in templates that have a slot for it (HY-MT's background template).
const nativeContextCues = 2

// nativeMaxTokens bounds one cue's reply; a subtitle line is short.
const nativeMaxTokens = 256

// nativeMaxConsecutiveFailures stops a run whose endpoint is clearly down
// instead of logging 200 identical errors.
const nativeMaxConsecutiveFailures = 5

// guessNativeTemplate maps a model id to the template family it speaks.
func guessNativeTemplate(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "translategemma"):
		return nativeTranslateGemma
	case strings.Contains(m, "hy-mt"), strings.Contains(m, "hymt"), strings.Contains(m, "hy_mt"):
		return nativeHyMT
	}
	return ""
}

// nativePrompt renders one cue in the model's own format. prev holds the
// preceding source lines, oldest first (may be empty).
//
// translategemma: the single-user-message structure Ollama's model page
// documents ("You are a professional {SOURCE_LANG} ({SOURCE_CODE}) to
// {TARGET_LANG} ({TARGET_CODE}) translator. …", two blank lines, then the
// text); Ollama's template wraps it in plain Gemma turns. The target name is
// "Traditional Chinese": with plain "Chinese" the 12B model answered in
// simplified characters on the NAS (2026-10-13 smoke test). It has no context
// slot, so prev is unused.
//
// hymt: Tencent Hy-MT2's English prompts (model card "Prompt Templates").
// When the cue ITSELF names someone in the glossary (golden-v2): the
// terminology template — "Reference the following translations:" then one
// "X translates to Y" line per such name. Otherwise: the default template for
// the first cue, the background-information template when there are
// preceding lines. The card has no combined terminology+background template;
// keying on the cue's own names keeps context for every nameless line. The target uses the card's full language name;
// Taiwan vocabulary is left to Vido's finalizer, as in production.
func nativePrompt(template, text string, prev []string, terms []eval.NameTerm) (string, error) {
	switch template {
	case nativeTranslateGemma:
		return "You are a professional English (en) to Traditional Chinese (zh-TW) translator. " +
			"Your goal is to accurately convey the meaning and nuances of the original English text " +
			"while adhering to Traditional Chinese grammar, vocabulary, and cultural sensitivities.\n" +
			"Produce only the Traditional Chinese translation, without any additional explanations or commentary. " +
			"Please translate the following English text into Traditional Chinese:\n\n\n" + text, nil
	case nativeHyMT:
		if len(terms) > 0 {
			var sb strings.Builder
			sb.WriteString("Reference the following translations:\n")
			for _, t := range terms {
				sb.WriteString(t.Source + " translates to " + t.Target + "\n")
			}
			sb.WriteString("\nTranslate the following text into Traditional Chinese. " +
				"Note that you must ONLY output the translated result without any additional explanation:\n\n" + text)
			return sb.String(), nil
		}
		if len(prev) == 0 {
			return "Translate the following text into Traditional Chinese. " +
				"Note that you should only output the translated result without any additional explanation:\n\n" + text, nil
		}
		return "[Background Information]\n" + strings.Join(prev, "\n") + "\n\n" +
			"Please translate the following text into Traditional Chinese, " +
			"taking the provided background information into consideration.\n\n" +
			"[Source Text]\n" + text, nil
	}
	return "", fmt.Errorf("unknown native template %q (want %s or %s)", template, nativeTranslateGemma, nativeHyMT)
}

// nativeTranslator renders cues one call at a time through completer. A cue
// the model fails on is left absent (the rule layer scores it "missing"), so
// one bad reply does not void the run; a stopped ctx, an exhausted budget, or
// a run of consecutive failures ends it, keeping what was already translated.
//
// glossaries (nil = none) are the per-source name glossaries; each prompt
// carries only the terms its cue or context mentions.
func nativeTranslator(completer ai.TextCompleter, template string, glossaries map[string][]eval.NameTerm, stderr io.Writer) eval.Translator {
	return func(ctx context.Context, cues []eval.Cue) (map[string]string, error) {
		got := make(map[string]string, len(cues))
		failures := 0
		for i, c := range cues {
			var prev []string
			for j := max(0, i-nativeContextCues); j < i; j++ {
				prev = append(prev, cues[j].Text)
			}
			var terms []eval.NameTerm
			if glossaries != nil {
				terms = eval.TermsIn(glossaries[c.Source], c.Text)
			}
			prompt, err := nativePrompt(template, c.Text, prev, terms)
			if err != nil {
				return got, err
			}
			out, err := completer.CompleteText(ctx, "", prompt, nativeMaxTokens)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, ai.ErrBudgetExceeded) {
					fmt.Fprintln(stderr)
					return got, err
				}
				failures++
				fmt.Fprintf(stderr, "\n  cue %s failed: %v\n", c.ID, err)
				if failures >= nativeMaxConsecutiveFailures {
					return got, fmt.Errorf("%d consecutive failures, last: %w", failures, err)
				}
				continue
			}
			failures = 0
			got[c.ID] = strings.TrimSpace(out)
			fmt.Fprintf(stderr, "\r  translating… %d/%d", i+1, len(cues))
		}
		fmt.Fprintln(stderr)
		return got, nil
	}
}
