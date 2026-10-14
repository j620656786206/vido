package ai

import (
	"fmt"
	"strings"
)

// Billing pages a CreditExhaustedError points the user at.
const (
	OpenAIBillingURL    = "https://platform.openai.com/settings/organization/billing"
	AnthropicBillingURL = "https://console.anthropic.com/settings/billing"
)

// CreditExhaustedError is the provider saying the ACCOUNT has no money left —
// not "slow down" (disc-2026-09-asr-quota-429-treated-as-transient). OpenAI
// sends it as a 429, the same status as a burst limit, so before this the
// Whisper client retried it three times, then took the 4xx for "this engine
// cannot do verbose_json", switched hallucination filtering off for the life of
// the process, and retried three more times on plain SRT — and the user got the
// whole JSON body as the reason (NAS log 2026-09-18 23:33).
//
// It is never retried and it unwraps to ErrAIQuotaExceeded, so the subtitle
// pipeline's "a provider quota stops the run, it does not degrade to English"
// rule (pipeline.go isTransientExhausted) covers it unchanged.
type CreditExhaustedError struct {
	// Provider is the lower-case provider name: "openai", "anthropic",
	// "elevenlabs".
	Provider string
	// Code is the provider's own error code ("insufficient_quota"), for logs.
	Code string
	// BillingURL is where the user tops the account up.
	BillingURL string
}

func (e *CreditExhaustedError) Error() string {
	return fmt.Sprintf("AI_CREDIT_EXHAUSTED: %s account has no credit left (%s)", e.Provider, e.Code)
}

// Unwrap makes errors.Is(err, ErrAIQuotaExceeded) hold.
func (e *CreditExhaustedError) Unwrap() error { return ErrAIQuotaExceeded }

// DisplayName is the provider as the user knows it.
func (e *CreditExhaustedError) DisplayName() string {
	switch e.Provider {
	case "openai":
		return "OpenAI"
	case "anthropic":
		return "Anthropic"
	case "elevenlabs":
		return "ElevenLabs"
	default:
		return e.Provider
	}
}

// openAICreditCodes are the error codes OpenAI-compatible endpoints use for an
// account that cannot pay — matched in the body, since OpenAI has sent them
// with 429 and with 400.
var openAICreditCodes = []string{"insufficient_quota", "billing_hard_limit_reached", "credit_balance_exhausted"}

// openAICreditExhausted returns the error for a response body that says the
// account is out of credit, or nil.
func openAICreditExhausted(body string) *CreditExhaustedError {
	lower := strings.ToLower(body)
	for _, code := range openAICreditCodes {
		if strings.Contains(lower, code) {
			return &CreditExhaustedError{Provider: "openai", Code: code, BillingURL: OpenAIBillingURL}
		}
	}
	return nil
}
