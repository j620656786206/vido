/**
 * E2E test-data seeding helpers (Story 20-2).
 *
 * WHY THIS EXISTS
 * ---------------
 * Playwright's `webServer` boots a FRESH, EMPTY backend in CI
 * (`VIDO_DATA_DIR: ./vido-data`). Data-dependent specs used to start with
 * `const movie = (await api.listMovies()).data?.items?.[0]` and then
 * `test.skip(!movie, 'No movies available')` — so on the empty CI DB they all
 * silently self-skipped and the suite went green WITHOUT exercising anything.
 * That is false confidence. These helpers let a spec seed exactly the row it
 * needs (and clean it up), so the test actually RUNS on an empty DB.
 *
 * IMPORTANT — snake_case payloads.
 * The TS `Movie`/`Series` interfaces are camelCase (`releaseDate`, `tmdbId`),
 * but the Go create endpoints bind snake_case (`release_date`, `tmdb_id`, and
 * `release_date`/`first_air_date` are `binding:"required"`). Passing a camelCase
 * literal to `api.createMovie` (as `media-detail.spec.ts` previously did) drops
 * the required field → 400 → `data` undefined → the very self-skip we are
 * killing. So we POST raw snake_case bodies via the generic `api.post` helper,
 * matching `CreateMovieRequest` / `CreateSeriesRequest` exactly.
 */
import { expect } from '../fixtures';
import type { ApiHelpers, Movie, Series } from './api-helpers';

let counter = 0;
/** Unique-enough title across parallel workers + reruns (no DB unique-title constraint, but avoids confusing overlaps). */
function uniqueTitle(prefix: string): string {
  counter += 1;
  return `${prefix} ${Date.now()}-${counter}`;
}

export interface SeedMovieOptions {
  title?: string;
  releaseDate?: string;
  /** Omit (or 0) to create a NO-METADATA movie — renders the ColorPlaceholder + FallbackFailed path. */
  tmdbId?: number;
  posterPath?: string;
  genres?: string[];
  overview?: string;
}

export interface SeedSeriesOptions {
  title?: string;
  firstAirDate?: string;
  tmdbId?: number;
  posterPath?: string;
  numberOfSeasons?: number;
  numberOfEpisodes?: number;
  genres?: string[];
  overview?: string;
}

/**
 * Create a movie via `POST /movies` and assert it was created.
 * With `tmdbId > 0` the local detail view renders the full layout (title shown);
 * omit `tmdbId` to exercise the no-metadata fallback UI (Story 5-11).
 */
export async function seedMovie(api: ApiHelpers, opts: SeedMovieOptions = {}): Promise<Movie> {
  const body: Record<string, unknown> = {
    title: opts.title ?? uniqueTitle('E2E Seed Movie'),
    release_date: opts.releaseDate ?? '2020-01-01',
  };
  if (opts.tmdbId) body.tmdb_id = opts.tmdbId;
  if (opts.posterPath) body.poster_path = opts.posterPath;
  if (opts.genres) body.genres = opts.genres;
  if (opts.overview) body.overview = opts.overview;

  const res = await api.post<Movie>('/movies', body);
  expect(res.success, `seedMovie failed: ${JSON.stringify(res.error)}`).toBe(true);
  expect(res.data?.id, 'seedMovie returned no id').toBeTruthy();
  return res.data as Movie;
}

/**
 * Create a series via `POST /series` and assert it was created.
 * NOTE: this does NOT populate the `seasons` table (only `parse_queue_service`
 * does), so the season accordion stays out of E2E reach — it is guarded by the
 * Go `:memory:` integration test added in bugfix-20-1 instead. See sprint-status
 * `story-20-4-season-accordion-e2e` (deferred).
 */
export async function seedSeries(api: ApiHelpers, opts: SeedSeriesOptions = {}): Promise<Series> {
  const body: Record<string, unknown> = {
    title: opts.title ?? uniqueTitle('E2E Seed Series'),
    first_air_date: opts.firstAirDate ?? '2020-01-01',
  };
  if (opts.tmdbId) body.tmdb_id = opts.tmdbId;
  if (opts.posterPath) body.poster_path = opts.posterPath;
  if (opts.numberOfSeasons) body.number_of_seasons = opts.numberOfSeasons;
  if (opts.numberOfEpisodes) body.number_of_episodes = opts.numberOfEpisodes;
  if (opts.genres) body.genres = opts.genres;
  if (opts.overview) body.overview = opts.overview;

  const res = await api.post<Series>('/series', body);
  expect(res.success, `seedSeries failed: ${JSON.stringify(res.error)}`).toBe(true);
  expect(res.data?.id, 'seedSeries returned no id').toBeTruthy();
  return res.data as Series;
}

/** Best-effort cleanup — safe to call with empty/undefined ids. */
export async function deleteMovies(
  api: ApiHelpers,
  ...ids: Array<string | undefined>
): Promise<void> {
  for (const id of ids) {
    if (id) await api.deleteMovie(id);
  }
}

export async function deleteSeries(
  api: ApiHelpers,
  ...ids: Array<string | undefined>
): Promise<void> {
  for (const id of ids) {
    if (id) await api.deleteSeries(id);
  }
}
