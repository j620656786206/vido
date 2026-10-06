package ai

import (
	"context"
	"strings"
)

// The speech-recognition PROMPT travels on the ctx, the way the per-run model
// and Budget do (disc-2026-10-asr-proper-names-inconsistent, the 聽 half).
//
// Whisper's `prompt` field conditions the decoder's spelling: given
// "Jerlamarel, Baba Voss, Paris, Maghra" it writes those names instead of
// Morel / Jerla / Chola Morel (See S01E02: one character, seven spellings,
// each one then translated differently). The transcription service composes
// it from TMDb credits and the show's trusted glossary; the client just
// forwards it, so ASRProvider — a stamped seam other engines implement — does
// not grow a parameter.
type asrPromptKey struct{}

// WithASRPrompt pins the conditioning text for every transcription request
// made under ctx. An empty prompt leaves the ctx unchanged.
func WithASRPrompt(ctx context.Context, prompt string) context.Context {
	if strings.TrimSpace(prompt) == "" {
		return ctx
	}
	return context.WithValue(ctx, asrPromptKey{}, prompt)
}

// ASRPromptFromContext returns the pinned prompt, or "" when none was set.
func ASRPromptFromContext(ctx context.Context) string {
	p, _ := ctx.Value(asrPromptKey{}).(string)
	return p
}

// ASR prompt limits. Whisper reads only the LAST ~224 tokens of the prompt;
// a long list is silently truncated from the FRONT, so the cap keeps the
// whole list inside the window and the caller must put the important names
// first (characters before actors).
const (
	ASRPromptMaxNames = 40
	ASRPromptMaxRunes = 700
)

// BuildASRPrompt renders a name list as whisper conditioning text: trimmed,
// de-duplicated (case-insensitively, first spelling wins), order preserved,
// capped by ASRPromptMaxNames and ASRPromptMaxRunes. "" when nothing is left
// — the caller then sends no prompt at all.
func BuildASRPrompt(names []string) string {
	seen := make(map[string]struct{}, len(names))
	var kept []string
	runes := 0
	for _, raw := range names {
		n := strings.TrimSpace(raw)
		if n == "" {
			continue
		}
		key := strings.ToLower(n)
		if _, dup := seen[key]; dup {
			continue
		}
		next := runes + len([]rune(n)) + 2 // ", "
		if len(kept) >= ASRPromptMaxNames || next > ASRPromptMaxRunes {
			break
		}
		seen[key] = struct{}{}
		kept = append(kept, n)
		runes = next
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, ", ")
}
