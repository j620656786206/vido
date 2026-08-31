// Implements: <utility — no .pen counterpart>
/**
 * newLocalId — client-side unique id that works on ANY origin.
 *
 * `crypto.randomUUID` is SECURE-CONTEXT-ONLY (https or localhost). A NAS
 * install is browsed over `http://<LAN-IP>` — an insecure origin — where the
 * function simply does not exist, and the first real Synology install crashed
 * the setup wizard on it (globalThis.crypto.randomUUID is not a function).
 * `crypto.getRandomValues` has no such restriction, so the fallback builds an
 * RFC-4122 v4 UUID from it; the final Math.random tier only exists for
 * environments with no WebCrypto at all.
 *
 * Use this for LOCAL ids (React keys, draft rows). Server-side ids stay
 * server-generated.
 */
export function newLocalId(): string {
  const c = globalThis.crypto;
  if (c?.randomUUID) return c.randomUUID();
  if (c?.getRandomValues) {
    const bytes = c.getRandomValues(new Uint8Array(16));
    bytes[6] = (bytes[6] & 0x0f) | 0x40; // version 4
    bytes[8] = (bytes[8] & 0x3f) | 0x80; // variant 10
    const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join('');
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
  }
  return `id-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}
