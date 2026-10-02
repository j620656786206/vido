/**
 * Pure selection model behind the L3 season/episode tree (Story 13-2b). No
 * React here, so every cascade rule and the selection→wire mapping is unit-
 * testable on its own.
 *
 * A season is either FULL (every selectable episode, without needing to know
 * the episode numbers — the season may never have been expanded) or PARTIAL
 * (an explicit episode set, only possible once its episodes were loaded).
 * "Locked" episodes are already owned or already requested: never selectable,
 * so a submitted selection never overlaps them (13-2a AC #3 stays a backstop).
 */
import type { RequestCoverage } from '../services/requestService';

export type { RequestCoverage };

export interface TreeSeason {
  seasonNumber: number;
  name: string;
  episodeCount: number;
}

export interface TreeSelection {
  full: ReadonlySet<number>;
  partial: ReadonlyMap<number, ReadonlySet<number>>;
}

export type CheckState = 'checked' | 'mixed' | 'empty';

export const EMPTY_SELECTION: TreeSelection = { full: new Set(), partial: new Map() };

/** Seasons the tree lists: real seasons with episodes (season 0 = specials is left out). */
export function treeSeasons(seasons: readonly TreeSeason[] | undefined): TreeSeason[] {
  return (seasons ?? [])
    .filter((s) => s.seasonNumber > 0 && s.episodeCount > 0)
    .sort((a, b) => a.seasonNumber - b.seasonNumber);
}

export type EpisodeLock = 'owned' | 'requested';

/** Why one episode cannot be picked, or null when it can. */
export function episodeLock(
  coverage: RequestCoverage | undefined,
  season: number,
  episode: number
): EpisodeLock | null {
  if (!coverage) return null;
  const key = String(season);
  if (coverage.owned[key]?.includes(episode)) return 'owned';
  if (coverage.requestedSeasons.includes(season)) return 'requested';
  if (coverage.requestedEpisodes[key]?.includes(episode)) return 'requested';
  return null;
}

function lockedCount(coverage: RequestCoverage | undefined, season: TreeSeason): number {
  if (!coverage) return 0;
  if (coverage.requestedSeasons.includes(season.seasonNumber)) return season.episodeCount;
  const key = String(season.seasonNumber);
  // Only numbers TMDb lists for the season count: a library with absolute
  // numbering (anime "S1E26") must not make the season read 已入庫 (13-2b CR).
  const listed = (e: number) => e >= 1 && e <= season.episodeCount;
  const locked = new Set([
    ...(coverage.owned[key] ?? []).filter(listed),
    ...(coverage.requestedEpisodes[key] ?? []).filter(listed),
  ]);
  return locked.size;
}

/** A season with nothing left to pick (every episode owned or requested). */
export function seasonLocked(coverage: RequestCoverage | undefined, season: TreeSeason): boolean {
  return lockedCount(coverage, season) >= season.episodeCount;
}

/** Whether a season is ENTIRELY owned (the whole row reads 已入庫). */
export function seasonFullyOwned(
  coverage: RequestCoverage | undefined,
  season: TreeSeason
): boolean {
  const owned = coverage?.owned[String(season.seasonNumber)] ?? [];
  return owned.filter((e) => e >= 1 && e <= season.episodeCount).length >= season.episodeCount;
}

/**
 * Whether the WIRE must avoid "whole season" for this season. Deliberately
 * unfiltered (unlike lockedCount, which drives what the row SHOWS): the
 * backend's overlap check (13-2a checkSelectionOwnership) rejects a whole
 * season if ANY owned number sits under that season key — even E26 on a
 * season TMDb lists as E1–E25 (13-2c CR).
 */
function seasonHasLocks(coverage: RequestCoverage | undefined, season: TreeSeason): boolean {
  if (!coverage) return false;
  if (coverage.requestedSeasons.includes(season.seasonNumber)) return true;
  const key = String(season.seasonNumber);
  return (
    (coverage.owned[key]?.length ?? 0) > 0 || (coverage.requestedEpisodes[key]?.length ?? 0) > 0
  );
}

function anyOwned(coverage: RequestCoverage | undefined): boolean {
  return !!coverage && Object.values(coverage.owned).some((eps) => eps.length > 0);
}

export function seasonState(sel: TreeSelection, season: number): CheckState {
  if (sel.full.has(season)) return 'checked';
  return (sel.partial.get(season)?.size ?? 0) > 0 ? 'mixed' : 'empty';
}

export function episodeChecked(sel: TreeSelection, season: number, episode: number): boolean {
  return sel.full.has(season) || (sel.partial.get(season)?.has(episode) ?? false);
}

function clone(sel: TreeSelection) {
  return { full: new Set(sel.full), partial: new Map(sel.partial) };
}

/** Season checkbox: any selection → clear; none → every selectable episode. */
export function toggleSeason(sel: TreeSelection, season: number): TreeSelection {
  const next = clone(sel);
  if (seasonState(sel, season) === 'empty') {
    next.full.add(season);
  } else {
    next.full.delete(season);
  }
  next.partial.delete(season);
  return next;
}

/**
 * Episode checkbox. `selectable` = the season's pickable episode numbers (the
 * row is only visible once loaded, so they are known). Unticking one episode of
 * a FULL season turns it into a partial set; ticking the last one back
 * collapses it to FULL again.
 */
export function toggleEpisode(
  sel: TreeSelection,
  season: number,
  episode: number,
  selectable: readonly number[]
): TreeSelection {
  const next = clone(sel);
  const current = sel.full.has(season)
    ? new Set(selectable)
    : new Set(sel.partial.get(season) ?? []);
  if (current.has(episode)) current.delete(episode);
  else current.add(episode);

  next.full.delete(season);
  next.partial.delete(season);
  if (current.size > 0 && selectable.every((e) => current.has(e))) {
    next.full.add(season);
  } else if (current.size > 0) {
    next.partial.set(season, current);
  }
  return next;
}

export function masterState(sel: TreeSelection, selectableSeasons: readonly number[]): CheckState {
  if (selectableSeasons.length > 0 && selectableSeasons.every((s) => sel.full.has(s))) {
    return 'checked';
  }
  return sel.full.size > 0 || sel.partial.size > 0 ? 'mixed' : 'empty';
}

/** 整部影集: everything selectable when not already all-checked, else nothing. */
export function toggleMaster(
  sel: TreeSelection,
  selectableSeasons: readonly number[]
): TreeSelection {
  if (masterState(sel, selectableSeasons) === 'checked') return EMPTY_SELECTION;
  return { full: new Set(selectableSeasons), partial: new Map() };
}

/** Footer 已選 N 季 · M 集: full seasons, plus episodes of partial seasons. */
export function selectionSummary(sel: TreeSelection): { seasons: number; episodes: number } {
  let episodes = 0;
  for (const set of sel.partial.values()) episodes += set.size;
  return { seasons: sel.full.size, episodes };
}

export function isEmptySelection(sel: TreeSelection): boolean {
  return sel.full.size === 0 && sel.partial.size === 0;
}

/**
 * FULL seasons that contain locked episodes: the wire cannot say "whole
 * season" for them (it would overlap what is owned/requested → 400), so the
 * caller must load their episode numbers before buildRequestPayload.
 */
export function seasonsNeedingEpisodeList(
  sel: TreeSelection,
  seasons: readonly TreeSeason[],
  coverage: RequestCoverage | undefined
): number[] {
  return seasons
    .filter((s) => sel.full.has(s.seasonNumber) && seasonHasLocks(coverage, s))
    .map((s) => s.seasonNumber);
}

export type RequestPayload =
  | { whole: true }
  | { whole: false; seasons: number[]; episodes: Record<string, number[]> };

/**
 * Selection → the 13-2a wire (confirmed against [@contract-v1], 13-2a AC #1).
 * RED LINE: everything checked on a show with nothing owned or requested is a
 * WHOLE-title request with NO selection — byte-identical to the one-click 想要.
 */
export function buildRequestPayload(
  sel: TreeSelection,
  seasons: readonly TreeSeason[],
  coverage: RequestCoverage | undefined,
  episodeLists: ReadonlyMap<number, readonly number[]>,
  /** The title is known to be in the library (13-2c entry): never send WHOLE. */
  titleOwned = false
): RequestPayload {
  // The backend refuses a whole-title request for ANY locally present series
  // (409 REQUEST_ALREADY_IN_LIBRARY) — owned specials or episodes outside the
  // tree's seasons count too (13-2c CR).
  const anyLock =
    titleOwned || anyOwned(coverage) || seasons.some((s) => seasonHasLocks(coverage, s));
  const allFull = seasons.length > 0 && seasons.every((s) => sel.full.has(s.seasonNumber));
  if (!anyLock && allFull) return { whole: true };

  const wholeSeasons: number[] = [];
  const episodes: Record<string, number[]> = {};
  for (const season of seasons) {
    const n = season.seasonNumber;
    if (sel.full.has(n)) {
      if (!seasonHasLocks(coverage, season)) {
        wholeSeasons.push(n);
        continue;
      }
      const picked = (episodeLists.get(n) ?? []).filter(
        (e) => episodeLock(coverage, n, e) === null
      );
      if (picked.length > 0) episodes[String(n)] = [...picked].sort((a, b) => a - b);
      continue;
    }
    const partial = sel.partial.get(n);
    if (partial && partial.size > 0) {
      episodes[String(n)] = [...partial].sort((a, b) => a - b);
    }
  }
  return { whole: false, seasons: wholeSeasons, episodes };
}

/**
 * The stored selection columns (JSON text) → a short range label for the
 * 想要清單 row, e.g. 「第 1、2 季」「第 3 季 · 3 集」. Null for a whole-title
 * request or unreadable data.
 */
export function storedSelectionParts(
  seasonsJson: string | null,
  episodesJson: string | null
): { seasons: number[]; episodes: Array<[number, number]> } | null {
  let seasons: number[] = [];
  let episodes: Record<string, number[]> = {};
  try {
    if (seasonsJson) seasons = JSON.parse(seasonsJson) as number[];
    if (episodesJson) episodes = JSON.parse(episodesJson) as Record<string, number[]>;
  } catch {
    return null;
  }
  // JSON text "null" / a non-object must not crash the request list.
  if (episodes === null || typeof episodes !== 'object' || Array.isArray(episodes)) episodes = {};
  const eps = Object.entries(episodes)
    .map(([s, list]) => [Number(s), Array.isArray(list) ? list.length : 0] as [number, number])
    .filter(([s, n]) => Number.isFinite(s) && n > 0)
    .sort((a, b) => a[0] - b[0]);
  const seas = Array.isArray(seasons) ? [...seasons].sort((a, b) => a - b) : [];
  if (seas.length === 0 && eps.length === 0) return null;
  return { seasons: seas, episodes: eps };
}
