import { describe, it, expect, vi, afterEach } from 'vitest';
import { newLocalId } from './uid';

const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;

describe('newLocalId', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('uses crypto.randomUUID when available (secure context)', () => {
    expect(newLocalId()).toMatch(UUID_V4);
  });

  it('falls back to getRandomValues on an insecure origin (http://<NAS-IP> — no randomUUID)', () => {
    // The exact environment of the first Synology install crash: crypto exists,
    // randomUUID does not.
    vi.stubGlobal('crypto', {
      getRandomValues: (arr: Uint8Array) => {
        for (let i = 0; i < arr.length; i++) arr[i] = i * 7 + 1;
        return arr;
      },
    });

    const id = newLocalId();
    expect(id).toMatch(UUID_V4);
  });

  it('still produces unique-ish ids with no WebCrypto at all', () => {
    vi.stubGlobal('crypto', undefined);
    const a = newLocalId();
    const b = newLocalId();
    expect(a).not.toBe(b);
    expect(a.startsWith('id-')).toBe(true);
  });
});
