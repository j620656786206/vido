import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  keySettingsService,
  KeySettingsApiError,
  REASON_ENCRYPTION_KEY_MISSING,
} from './keySettingsService';

const fetchMock = vi.fn();

beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function ok(data: unknown) {
  return { ok: true, status: 200, json: async () => ({ success: true, data }) };
}

function fail(status: number, code: string, message: string) {
  return {
    ok: false,
    status,
    json: async () => ({ success: false, error: { code, message } }),
  };
}

describe('keySettingsService.getKeys', () => {
  it('reads the 2-1a GET shape: per-key source + mask, plus writability', async () => {
    fetchMock.mockResolvedValue(
      ok({
        keys: [
          { name: 'claude', configured: true, source: 'secret', masked: 'sk-ant…7f3a' },
          { name: 'tmdb', configured: true, source: 'env' },
          { name: 'openai', configured: false, source: 'none' },
        ],
        writable: true,
      })
    );

    const settings = await keySettingsService.getKeys();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/settings/keys', undefined);
    expect(settings.writable).toBe(true);
    expect(settings.keys).toHaveLength(3);
    expect(settings.keys[0]).toEqual({
      name: 'claude',
      configured: true,
      source: 'secret',
      masked: 'sk-ant…7f3a',
    });
    // An env-sourced key is never masked-echoed — 2-1a returns no hint for it.
    expect(settings.keys[1].masked).toBeUndefined();
  });

  it('carries the encryption_key_missing reason through untouched', async () => {
    fetchMock.mockResolvedValue(
      ok({ keys: [], writable: false, reason: REASON_ENCRYPTION_KEY_MISSING })
    );

    const settings = await keySettingsService.getKeys();

    expect(settings.writable).toBe(false);
    expect(settings.reason).toBe('encryption_key_missing');
  });
});

describe('keySettingsService.saveKeys', () => {
  it('PUTs exactly the fields it was given — an omitted key is left untouched', async () => {
    fetchMock.mockResolvedValue(ok({ keys: [], writable: true }));

    await keySettingsService.saveKeys({ claude: 'sk-ant-x' });

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/v1/settings/keys');
    expect(init.method).toBe('PUT');
    expect(JSON.parse(init.body)).toEqual({ claude: 'sk-ant-x' });
  });

  it('sends an explicit empty string for a delete, not an omitted field', async () => {
    fetchMock.mockResolvedValue(ok({ keys: [], writable: true }));

    await keySettingsService.saveKeys({ tmdb: '' });

    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({ tmdb: '' });
  });

  it('preserves the Rule-7 code alongside the zh-TW message on refusal', async () => {
    fetchMock.mockResolvedValue(
      fail(409, 'AI_NOT_CONFIGURED', '未設定加密金鑰，無法安全儲存 API 金鑰')
    );

    await expect(keySettingsService.saveKeys({ claude: 'x' })).rejects.toMatchObject({
      name: 'KeySettingsApiError',
      code: 'AI_NOT_CONFIGURED',
      message: '未設定加密金鑰，無法安全儲存 API 金鑰',
    });
  });
});

describe('keySettingsService.testClaudeKey', () => {
  it('POSTs the candidate so a key can be probed before it is stored', async () => {
    fetchMock.mockResolvedValue(ok({ valid: true }));

    await keySettingsService.testClaudeKey('sk-ant-candidate');

    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe('/api/v1/settings/keys/test');
    expect(init.method).toBe('POST');
    expect(JSON.parse(init.body)).toEqual({ claude: 'sk-ant-candidate' });
  });

  it('POSTs an empty candidate to mean "test whatever currently resolves"', async () => {
    fetchMock.mockResolvedValue(ok({ valid: true }));

    await keySettingsService.testClaudeKey();

    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({ claude: '' });
  });

  it('rejects with the provider diagnosis rather than a generic failure', async () => {
    fetchMock.mockResolvedValue(fail(502, 'AI_PROVIDER_ERROR', '金鑰無效或已撤銷'));

    await expect(keySettingsService.testClaudeKey('bad')).rejects.toBeInstanceOf(
      KeySettingsApiError
    );
  });

  it('falls back to a status-derived message when the body is not an envelope', async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 502,
      json: async () => {
        throw new Error('not json');
      },
    });

    await expect(keySettingsService.testClaudeKey('x')).rejects.toMatchObject({
      code: 'INTERNAL_ERROR',
      message: 'API request failed: 502',
    });
  });
});
