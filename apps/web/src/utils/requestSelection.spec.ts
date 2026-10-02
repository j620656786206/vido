import { describe, it, expect } from 'vitest';
import {
  EMPTY_SELECTION,
  buildRequestPayload,
  episodeLock,
  masterState,
  seasonFullyOwned,
  seasonLocked,
  seasonState,
  seasonsNeedingEpisodeList,
  selectionSummary,
  storedSelectionParts,
  toggleEpisode,
  toggleMaster,
  toggleSeason,
  treeSeasons,
  type RequestCoverage,
  type TreeSeason,
} from './requestSelection';

const seasons: TreeSeason[] = [
  { seasonNumber: 1, name: '第 1 季', episodeCount: 4 },
  { seasonNumber: 2, name: '第 2 季', episodeCount: 3 },
];
const none: RequestCoverage = {
  owned: {},
  requestedSeasons: [],
  requestedEpisodes: {},
  wholeSeriesRequested: false,
  activeRequest: false,
};
const s1Owned12: RequestCoverage = { ...none, owned: { '1': [1, 2] } };

describe('treeSeasons', () => {
  it('drops specials (season 0) and empty seasons, sorts by number', () => {
    expect(
      treeSeasons([
        { seasonNumber: 2, name: 'b', episodeCount: 3 },
        { seasonNumber: 0, name: '特別篇', episodeCount: 5 },
        { seasonNumber: 3, name: 'c', episodeCount: 0 },
        { seasonNumber: 1, name: 'a', episodeCount: 4 },
      ]).map((s) => s.seasonNumber)
    ).toEqual([1, 2]);
  });
});

describe('locks from coverage', () => {
  it('owned beats requested; a requested season locks every episode', () => {
    const cov: RequestCoverage = {
      ...none,
      owned: { '1': [1] },
      requestedSeasons: [2],
      requestedEpisodes: { '1': [3] },
    };
    expect(episodeLock(cov, 1, 1)).toBe('owned');
    expect(episodeLock(cov, 1, 3)).toBe('requested');
    expect(episodeLock(cov, 2, 1)).toBe('requested');
    expect(episodeLock(cov, 1, 4)).toBeNull();
    expect(episodeLock(undefined, 1, 1)).toBeNull();
  });

  it('a season is locked when nothing is left to pick; fully owned only when all owned', () => {
    const cov: RequestCoverage = {
      ...none,
      owned: { '2': [1, 2, 3] },
      requestedEpisodes: { '1': [1, 2] },
    };
    expect(seasonLocked(cov, seasons[1])).toBe(true);
    expect(seasonFullyOwned(cov, seasons[1])).toBe(true);
    expect(seasonLocked(cov, seasons[0])).toBe(false);
    expect(seasonFullyOwned(cov, seasons[0])).toBe(false);
  });
});

describe('absolute numbering (CR)', () => {
  it('owned numbers beyond the season’s TMDb count do not lock it', () => {
    // A library stores anime S2 as E26–E37; TMDb lists S2 as E1–E12.
    const cov: RequestCoverage = { ...none, owned: { '2': [26, 27, 28] } };
    expect(seasonLocked(cov, seasons[1])).toBe(false);
    expect(seasonFullyOwned(cov, seasons[1])).toBe(false);
  });
});

describe('owned series never go WHOLE (13-2c CR)', () => {
  it('owned specials only (season 0, not in the tree) → per-season, not whole', () => {
    const cov: RequestCoverage = { ...none, owned: { '0': [1] } };
    const all = toggleMaster(EMPTY_SELECTION, [1, 2]);
    expect(buildRequestPayload(all, seasons, cov, new Map())).toEqual({
      whole: false,
      seasons: [1, 2],
      episodes: {},
    });
  });

  it('a known-owned title with empty coverage → per-season, not whole', () => {
    const all = toggleMaster(EMPTY_SELECTION, [1, 2]);
    expect(buildRequestPayload(all, seasons, none, new Map(), true)).toEqual({
      whole: false,
      seasons: [1, 2],
      episodes: {},
    });
  });

  it('absolute-numbered owned episodes still make the season go per-episode (backend overlap check)', () => {
    const cov: RequestCoverage = { ...none, owned: { '1': [26, 27] } };
    const sel = toggleSeason(EMPTY_SELECTION, 1);
    expect(seasonsNeedingEpisodeList(sel, seasons, cov)).toEqual([1]);
    expect(buildRequestPayload(sel, seasons, cov, new Map([[1, [1, 2, 3, 4]]]))).toEqual({
      whole: false,
      seasons: [],
      episodes: { '1': [1, 2, 3, 4] },
    });
  });
});

describe('cascade', () => {
  it('season toggle: empty → full → empty', () => {
    const a = toggleSeason(EMPTY_SELECTION, 1);
    expect(seasonState(a, 1)).toBe('checked');
    expect(seasonState(toggleSeason(a, 1), 1)).toBe('empty');
  });

  it('unticking one episode of a full season makes it partial (mixed); ticking it back restores full', () => {
    const full = toggleSeason(EMPTY_SELECTION, 1);
    const partial = toggleEpisode(full, 1, 2, [1, 2, 3, 4]);
    expect(seasonState(partial, 1)).toBe('mixed');
    expect([...partial.partial.get(1)!]).toEqual([1, 3, 4]);
    expect(seasonState(toggleEpisode(partial, 1, 2, [1, 2, 3, 4]), 1)).toBe('checked');
  });

  it('a mixed season toggles to empty, not full', () => {
    const mixed = toggleEpisode(EMPTY_SELECTION, 1, 1, [1, 2, 3, 4]);
    expect(seasonState(mixed, 1)).toBe('mixed');
    expect(seasonState(toggleSeason(mixed, 1), 1)).toBe('empty');
  });

  it('master: empty → all selectable full → empty; mixed in between', () => {
    expect(masterState(EMPTY_SELECTION, [1, 2])).toBe('empty');
    const all = toggleMaster(EMPTY_SELECTION, [1, 2]);
    expect(masterState(all, [1, 2])).toBe('checked');
    expect(masterState(toggleSeason(all, 2), [1, 2])).toBe('mixed');
    expect(masterState(toggleMaster(all, [1, 2]), [1, 2])).toBe('empty');
  });

  it('summary counts full seasons and partial episodes (design: 已選 1 季 · 3 集)', () => {
    let sel = toggleSeason(EMPTY_SELECTION, 2);
    for (const e of [1, 2, 3]) sel = toggleEpisode(sel, 1, e, [1, 2, 3, 4]);
    expect(selectionSummary(sel)).toEqual({ seasons: 1, episodes: 3 });
  });
});

describe('buildRequestPayload', () => {
  it('RED LINE: everything checked, nothing owned → whole title, no selection', () => {
    const all = toggleMaster(EMPTY_SELECTION, [1, 2]);
    expect(buildRequestPayload(all, seasons, none, new Map())).toEqual({ whole: true });
    expect(buildRequestPayload(all, seasons, undefined, new Map())).toEqual({ whole: true });
  });

  it('full seasons without locks go as seasons; partial ones as episodes', () => {
    let sel = toggleSeason(EMPTY_SELECTION, 2);
    sel = toggleEpisode(sel, 1, 3, [1, 2, 3, 4]);
    sel = toggleEpisode(sel, 1, 1, [1, 2, 3, 4]);
    expect(buildRequestPayload(sel, seasons, none, new Map())).toEqual({
      whole: false,
      seasons: [2],
      episodes: { '1': [1, 3] },
    });
  });

  it('everything checked on a partly owned show is NOT whole: the owned season goes as its missing episodes', () => {
    const all = toggleMaster(EMPTY_SELECTION, [1, 2]);
    expect(seasonsNeedingEpisodeList(all, seasons, s1Owned12)).toEqual([1]);
    expect(buildRequestPayload(all, seasons, s1Owned12, new Map([[1, [1, 2, 3, 4]]]))).toEqual({
      whole: false,
      seasons: [2],
      episodes: { '1': [3, 4] },
    });
  });
});

describe('empty partial payload (CR)', () => {
  it('a full season whose listed episodes are all locked yields NO selection — the dialog must refuse it', () => {
    const cov: RequestCoverage = { ...none, owned: { '1': [1, 2, 3] } };
    const sel = toggleSeason(EMPTY_SELECTION, 1);
    expect(buildRequestPayload(sel, seasons, cov, new Map([[1, [1, 2, 3]]]))).toEqual({
      whole: false,
      seasons: [],
      episodes: {},
    });
  });
});

describe('storedSelectionParts', () => {
  it('parses the stored JSON columns', () => {
    expect(storedSelectionParts('[2,1]', '{"3":[1,2,5]}')).toEqual({
      seasons: [1, 2],
      episodes: [[3, 3]],
    });
  });

  it('whole-title and unreadable data → null', () => {
    expect(storedSelectionParts(null, null)).toBeNull();
    expect(storedSelectionParts('not json', null)).toBeNull();
    expect(storedSelectionParts(null, 'null')).toBeNull(); // CR: must not throw
  });
});
