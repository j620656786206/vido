package ai

import (
	"strings"
	"time"
)

// Request timeouts are derived per call, not fixed per provider (sub-6-2).
//
// The 15-second constant this replaces was NFR-I12's bound for a filename
// parse — a few hundred output tokens. A subtitle chunk on Sonnet is ten cues
// plus a glossary trailer, up to ~2k output tokens, and routinely runs past
// 15 s while perfectly healthy. Three such "timeouts" in a row failed a
// 986-cue run at cue 935 after the first 935 had already been paid for
// (eval-1 product problem 4, twice). A bigger constant only moves the cliff;
// the bound has to follow what the call actually asks for.
const (
	// timeoutPerThousandOutputTokens is the linear allowance on top of the
	// family base: the model has to WRITE those tokens, and output rate is the
	// slow side of every provider.
	timeoutPerThousandOutputTokens = 10 * time.Second

	// MaxRequestTimeout caps the derivation so a runaway max_tokens cannot turn
	// one hung connection into an unbounded wait — retryTransient still owns
	// the three attempts above this.
	MaxRequestTimeout = 180 * time.Second
)

// RequestTimeoutFor returns the per-ATTEMPT deadline for one request to model
// that may produce up to maxTokens output tokens: the model family's base
// (llmTimeoutBase, kept beside the pricing table) plus
// timeoutPerThousandOutputTokens per 1k output tokens, capped at
// MaxRequestTimeout. An unknown model is treated as Sonnet-class — the middle
// of the table, so a new model id is neither cut off like Haiku nor waited on
// like Opus. maxTokens <= 0 adds nothing (a Ping, a parse with the SDK default).
func RequestTimeoutFor(model string, maxTokens int) time.Duration {
	timeout := baseTimeoutFor(model)
	if maxTokens > 0 {
		timeout += time.Duration(maxTokens) * timeoutPerThousandOutputTokens / 1000
	}
	if timeout > MaxRequestTimeout {
		timeout = MaxRequestTimeout
	}
	return timeout
}

// baseTimeoutFor matches the model id against the family table by substring,
// first match wins, so "claude-sonnet-5" and "claude-sonnet-4-6" share a row
// and a future "claude-haiku-5" needs no table edit.
func baseTimeoutFor(model string) time.Duration {
	id := strings.ToLower(model)
	for _, row := range llmTimeoutBase {
		if strings.Contains(id, row.family) {
			return row.base
		}
	}
	return unknownModelTimeoutBase
}
