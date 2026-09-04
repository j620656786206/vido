/**
 * Provider API-key settings client (Story sub-2-1b — FR25 / architecture D9).
 *
 * Consumes the sub-2-1a triad; confirmed against [@contract-v1] sub-2-1a AC #1
 * (KeyResolver — an encrypted secret WINS over an environment variable, and
 * `source` is surfaced precisely so this page can be honest about that
 * precedence) and confirmed against [@contract-v1] sub-2-1a AC #3 (the
 * GET/PUT/POST-test wire shape mirrored below). This story stamps nothing.
 */

import type { ApiResponse } from '../types/tmdb';
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

/** Closed set — mirrors `services.KeyName`; doubles as the PUT body field names. */
export type KeyName = 'claude' | 'tmdb' | 'openai';

/** Where a key resolved from. `secret` (runtime, user-set) beats `env` (deploy-time). */
export type KeySource = 'secret' | 'env' | 'none';

export interface KeyState {
  name: KeyName;
  configured: boolean;
  source: KeySource;
  /**
   * first6…last4 hint. Present ONLY for secret-sourced keys — an env-sourced
   * key is a deploy secret and 2-1a never echoes any part of it back.
   */
  masked?: string;
}

export interface KeySettings {
  keys: KeyState[];
  /** false ⇒ ENCRYPTION_KEY is unset, so the page must render read-only (AC #2). */
  writable: boolean;
  /** Explains a false `writable`. Empty/absent when writable. */
  reason?: string;
}

/** The only `reason` 2-1a emits today (`services.ReasonEncryptionKeyMissing`). */
export const REASON_ENCRYPTION_KEY_MISSING = 'encryption_key_missing';

/**
 * Carries the backend's Rule-7 code alongside the zh-TW message.
 *
 * NOTE (sub-2-1a CR L1): the encryption-key refusal and the no-key-to-test
 * refusal BOTH answer `AI_NOT_CONFIGURED`, so the code alone cannot tell them
 * apart. Read-only mode is therefore driven by GET's `writable`/`reason`, never
 * by branching on this code.
 */
export class KeySettingsApiError extends Error {
  readonly code: string;

  constructor(message: string, code: string) {
    super(message);
    this.name = 'KeySettingsApiError';
    this.code = code;
  }
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, options);
  const data = (await response.json().catch(() => null)) as ApiResponse<T> | null;

  if (!response.ok || !data?.success) {
    throw new KeySettingsApiError(
      data?.error?.message || `API request failed: ${response.status}`,
      data?.error?.code || 'INTERNAL_ERROR'
    );
  }

  return snakeToCamel<T>(data.data as T);
}

/** Partial update: only the named keys are touched. `''` DELETES a stored secret. */
export type KeyUpdates = Partial<Record<KeyName, string>>;

/** POST /settings/keys/test 200 body. `model` is additive since sub-6-6. */
export interface KeyTestResult {
  valid: boolean;
  /** The model id the probe verified against; absent on a pre-sub-6-6 server. */
  model?: string;
}

export const keySettingsService = {
  /** GET /api/v1/settings/keys — state only; values are never returned. */
  async getKeys(): Promise<KeySettings> {
    return fetchApi<KeySettings>('/settings/keys');
  },

  /**
   * PUT /api/v1/settings/keys — omitted fields stay untouched, an explicit `''`
   * deletes the stored secret and reverts resolution to the env var.
   * Responds with the fresh state, so the caller can seed the cache without a
   * second round trip (the env→secret flip is the visible proof it took).
   *
   * The field names are already lowercase single words, so no camelToSnake pass
   * is needed (or wanted — it would be a no-op that hides the literal contract).
   */
  async saveKeys(updates: KeyUpdates): Promise<KeySettings> {
    return fetchApi<KeySettings>('/settings/keys', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(updates),
    });
  },

  /**
   * POST /api/v1/settings/keys/test — probes the live provider.
   * `candidate` lets the page verify BEFORE saving; omit it to test whatever
   * currently resolves. Only the Claude key has a probe endpoint today.
   *
   * sub-6-6 (additive): the 200 body now names the model the probe actually
   * reached. That answers the question a bare 「金鑰驗證成功」 leaves open — a
   * key can be valid while `CLAUDE_MODEL` points at a model this account
   * cannot call, and the two failures need different fixes. Absent on a
   * pre-sub-6-6 server.
   */
  async testClaudeKey(candidate?: string): Promise<KeyTestResult> {
    return fetchApi<KeyTestResult>('/settings/keys/test', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ claude: candidate ?? '' }),
    });
  },
};

export default keySettingsService;
