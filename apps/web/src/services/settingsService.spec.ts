import { describe, it, expect, vi, afterEach } from 'vitest';
import { settingsService } from './settingsService';

function mockFetch(impl: () => Partial<Response> | Promise<Partial<Response>>) {
  vi.stubGlobal('fetch', vi.fn(impl) as unknown as typeof fetch);
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('settingsService.getBoolFlag (fail-soft flag reader)', () => {
  it('returns true when the setting value is the string "true"', async () => {
    mockFetch(() => ({
      ok: true,
      json: async () => ({
        success: true,
        data: { key: 'new_shell_enabled', value: 'true', type: 'bool' },
      }),
    }));
    await expect(settingsService.getBoolFlag('new_shell_enabled')).resolves.toBe(true);
  });

  it('returns false when the value is "false"', async () => {
    mockFetch(() => ({
      ok: true,
      json: async () => ({
        success: true,
        data: { key: 'new_shell_enabled', value: 'false', type: 'bool' },
      }),
    }));
    await expect(settingsService.getBoolFlag('new_shell_enabled')).resolves.toBe(false);
  });

  it('returns false (fail-soft) on a 404 — an unseeded flag', async () => {
    mockFetch(() => ({ ok: false, status: 404, json: async () => ({ success: false }) }));
    await expect(settingsService.getBoolFlag('missing')).resolves.toBe(false);
  });

  it('returns false (fail-soft) when the response is not successful', async () => {
    mockFetch(() => ({ ok: true, json: async () => ({ success: false }) }));
    await expect(settingsService.getBoolFlag('new_shell_enabled')).resolves.toBe(false);
  });

  it('returns false (fail-soft) on a network error', async () => {
    mockFetch(() => Promise.reject(new Error('network down')));
    await expect(settingsService.getBoolFlag('new_shell_enabled')).resolves.toBe(false);
  });
});
