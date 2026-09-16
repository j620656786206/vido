import { describe, it, expect } from 'vitest';
import { ApiError, isNotFoundError } from './apiError';

describe('ApiError', () => {
  it('is an Error that keeps the message, HTTP status and Rule-7 code', () => {
    const err = new ApiError('Movie not found', 404, 'DB_NOT_FOUND');
    expect(err).toBeInstanceOf(Error);
    expect(err.message).toBe('Movie not found');
    expect(err.status).toBe(404);
    expect(err.code).toBe('DB_NOT_FOUND');
    expect(err.name).toBe('ApiError');
  });
});

describe('isNotFoundError', () => {
  it('is true only for a 404', () => {
    expect(isNotFoundError(new ApiError('x', 404, 'DB_NOT_FOUND'))).toBe(true);
    expect(isNotFoundError(new ApiError('x', 500, 'DB_QUERY_FAILED'))).toBe(false);
  });

  // dsr-2 AC #8: anything we cannot positively identify as "missing" is a load
  // failure — never tell the user an item was removed on a guess.
  it('is false for plain errors, timeouts and non-errors', () => {
    expect(isNotFoundError(new Error('TMDB_TIMEOUT'))).toBe(false);
    expect(isNotFoundError(undefined)).toBe(false);
    expect(isNotFoundError(null)).toBe(false);
    expect(isNotFoundError('404')).toBe(false);
  });

  // Duck-typed on purpose: a spec that vi.mock()s a service module must not break
  // the check through a second copy of the class.
  it('recognises a status-404 error object without relying on instanceof', () => {
    expect(isNotFoundError(Object.assign(new Error('gone'), { status: 404 }))).toBe(true);
  });
});
