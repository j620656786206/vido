/**
 * Auth API service (V0.1.1 password gate).
 *
 * The API and the SPA are served from the same origin, so the browser sends the
 * HttpOnly session cookie automatically — no credentials handling needed here.
 */

import { snakeToCamel } from '../utils/caseTransform';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api/v1';

export interface AuthStatus {
  /** True when the server was started with VIDO_AUTH_PASSWORD set. */
  authEnabled: boolean;
  /** True when the current session cookie is valid (always true when disabled). */
  authenticated: boolean;
}

interface ApiResponse<T> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  const data: ApiResponse<T> = await response.json();

  if (!response.ok || !data.success) {
    throw new Error(data.error?.message || `API request failed: ${response.status}`);
  }

  return snakeToCamel<T>(data.data);
}

export const authService = {
  async getStatus(): Promise<AuthStatus> {
    return fetchApi<AuthStatus>('/auth/status');
  },

  async login(password: string): Promise<AuthStatus> {
    return fetchApi<AuthStatus>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ password }),
    });
  },

  async logout(): Promise<void> {
    await fetchApi('/auth/logout', { method: 'POST' });
  },
};

export default authService;
