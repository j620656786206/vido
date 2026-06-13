/**
 * Settings flag reader (UX Redesign Phase 2 — FOUNDATION / UX2-1).
 *
 * Reads a single boolean setting from the existing key/value settings endpoint
 * (`GET /api/v1/settings/:key`, backed by `SettingsRepository.GetBool`). This is
 * deliberately fail-soft: an absent key (404), a non-`ok` response, or a network
 * error all resolve to `false` so the shell-selection read in `__root.tsx` falls
 * back to the known-good legacy chassis rather than breaking the whole app on a
 * settings hiccup. The flag is seeded OFF at backend startup (`main.go`).
 */
import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

interface SettingRow {
  key: string;
  value: string;
  type: string;
}

export const settingsService = {
  /**
   * Read a boolean feature flag. Returns `false` on absence/error (fail-soft).
   * A bool setting stores its value as the string `"true"` / `"false"`.
   */
  async getBoolFlag(key: string): Promise<boolean> {
    try {
      const response = await fetch(`${API_BASE_URL}/settings/${encodeURIComponent(key)}`);
      if (!response.ok) return false; // 404 (unseeded) or any error → flag off
      const data = await response.json();
      if (!data?.success) return false;
      const row = snakeToCamel<SettingRow>(data.data);
      return row?.value === 'true';
    } catch {
      return false; // network failure → flag off (legacy shell)
    }
  },
};
