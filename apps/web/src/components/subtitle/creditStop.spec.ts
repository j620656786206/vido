import { describe, it, expect } from 'vitest';
import { creditProviderName, creditStopSentence, safeBillingUrl } from './creditStop';

describe('creditStop', () => {
  it('names the two providers and says AI for anything else', () => {
    expect(creditProviderName('openai')).toBe('OpenAI');
    expect(creditProviderName('anthropic')).toBe('Anthropic');
    expect(creditProviderName('elevenlabs')).toBe('ElevenLabs');
    expect(creditProviderName('gemini')).toBe('AI');
    expect(creditProviderName('')).toBe('AI');
    expect(creditProviderName(undefined)).toBe('AI');
  });

  it('the announcement is the F9b banner sentence', () => {
    expect(creditStopSentence('openai', 3, 2)).toBe(
      'OpenAI 帳號沒有額度了 — 已完成3部，剩餘2部已暫停，儲值後按「下次繼續」'
    );
  });

  it('only an https billing url becomes a link', () => {
    expect(safeBillingUrl('https://console.anthropic.com/settings/billing')).toBe(
      'https://console.anthropic.com/settings/billing'
    );
    expect(safeBillingUrl('javascript:alert(1)')).toBeNull();
    expect(safeBillingUrl('http://example.com')).toBeNull();
    expect(safeBillingUrl('')).toBeNull();
    expect(safeBillingUrl(undefined)).toBeNull();
  });
});
