/**
 * API error that keeps what a bare `new Error(message)` throws away: the HTTP
 * status and the backend's Rule-7 code (project-context.md Rule 7).
 *
 * dsr-2 AC #8 — the detail page used to render 「找不到這部影片」 for EVERY failed
 * request, because the services threw plain Errors and a 404 was indistinguishable
 * from a 500. It lives in `lib/`, not beside a service, on purpose: specs routinely
 * `vi.mock()` whole service modules, and a class defined inside a mocked module
 * would import as `undefined` there.
 */
export class ApiError extends Error {
  readonly status: number;
  readonly code?: string;

  constructor(message: string, status: number, code?: string) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.code = code;
  }
}

/**
 * True only when the request positively came back 404. Duck-typed (not
 * `instanceof`) so a second copy of the class — e.g. through a mocked module —
 * cannot flip the answer. Anything unrecognised is NOT "not found": telling the
 * user an item was removed on a guess is the exact lie this exists to prevent.
 */
export function isNotFoundError(error: unknown): boolean {
  return (
    typeof error === 'object' && error !== null && (error as { status?: unknown }).status === 404
  );
}
