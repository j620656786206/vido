// Implements: <utility — no .pen counterpart>
/**
 * Words for a batch that stopped because the AI account ran out of money
 * (disc-2026-10-batch-stop-on-credit-exhausted-b). The F9b-D-v2 banner (YHnnC)
 * is the source; the batch dialog and the generation workspace both show it
 * verbatim, so it is built in one place.
 *
 * No money is not "it broke": like the budget ceiling it is ochre, the rest of
 * the queue is 已暫停, and after a top-up 下次繼續 picks it up.
 */

const PROVIDER_NAMES: Record<string, string> = {
  openai: 'OpenAI',
  anthropic: 'Anthropic',
  elevenlabs: 'ElevenLabs',
};

/** 'openai' → 'OpenAI'. An id this build does not know (or '') says just "AI". */
export function creditProviderName(provider: string | null | undefined): string {
  return (provider && PROVIDER_NAMES[provider]) || 'AI';
}

/**
 * The F9b banner as one sentence — for the screen-reader announcement; the
 * visible banner sets the two numbers in mono between the same words.
 */
export function creditStopSentence(
  provider: string | null | undefined,
  done: number,
  paused: number
): string {
  return `${creditProviderName(provider)} 帳號沒有額度了 — 已完成${done}部，剩餘${paused}部已暫停，儲值後按「下次繼續」`;
}

/**
 * Only an https link from the server is followed — the URL is ours (two
 * constants in the Go `ai` package), but it reaches the page through the SSE
 * stream, and an href is no place to trust a value we did not check.
 */
export function safeBillingUrl(url: string | null | undefined): string | null {
  return url && url.startsWith('https://') ? url : null;
}
