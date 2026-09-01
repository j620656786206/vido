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
    suggestion?: string;
  };
}

/**
 * A failed login, carrying everything the server already worked out.
 *
 * The plain `Error` this replaced kept only `message` ("密碼錯誤") and dropped
 * the two things the user actually needs: the API's `suggestion` — which is
 * where the server puts "還可以再試 2 次" — and the `Retry-After` header that
 * says how long a lockout has to run. Both are real, measured numbers; the
 * house rule against inventing readouts has a mirror image, which is that a
 * readout the system DID measure has to reach the screen.
 */
export class AuthError extends Error {
  readonly code: string;
  readonly suggestion?: string;
  /** Seconds left on a per-IP lockout, from the Retry-After header (429 only). */
  readonly retryAfterSeconds?: number;

  constructor(args: {
    message: string;
    code: string;
    suggestion?: string;
    retryAfterSeconds?: number;
  }) {
    super(args.message);
    this.name = 'AuthError';
    this.code = args.code;
    this.suggestion = args.suggestion;
    this.retryAfterSeconds = args.retryAfterSeconds;
  }

  /** True when this IP is locked out rather than merely wrong. */
  get isLockedOut(): boolean {
    return this.code === 'TOO_MANY_ATTEMPTS';
  }
}

function parseRetryAfter(response: Response): number | undefined {
  const raw = response.headers.get('Retry-After');
  if (!raw) return undefined;
  const seconds = Number.parseInt(raw, 10);
  return Number.isFinite(seconds) && seconds > 0 ? seconds : undefined;
}

async function fetchApi<T>(endpoint: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  });

  const data: ApiResponse<T> = await response.json();

  if (!response.ok || !data.success) {
    throw new AuthError({
      message: data.error?.message || `API request failed: ${response.status}`,
      code: data.error?.code || 'UNKNOWN',
      suggestion: data.error?.suggestion,
      retryAfterSeconds: parseRetryAfter(response),
    });
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
