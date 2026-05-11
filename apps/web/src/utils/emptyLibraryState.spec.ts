import { describe, it, expect } from 'vitest';
import { classifyEmptyState } from './emptyLibraryState';

describe('classifyEmptyState (bugfix-10-5 AC #1 [@contract-v1])', () => {
  it('returns loading when isLoading=true regardless of other inputs', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: false,
        mediaLibrariesCount: 0,
        itemsCount: 0,
        isLoading: true,
      })
    ).toBe('loading');
    expect(
      classifyEmptyState({
        qbtConfigured: true,
        mediaLibrariesCount: 5,
        itemsCount: 100,
        isLoading: true,
      })
    ).toBe('loading');
  });

  it('returns loading when qbtConfigured is undefined (query still pending)', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: undefined,
        mediaLibrariesCount: 0,
        itemsCount: 0,
        isLoading: false,
      })
    ).toBe('loading');
  });

  it('returns no-qbt when qbtConfigured=false (priority over folder/items)', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: false,
        mediaLibrariesCount: 0,
        itemsCount: 0,
        isLoading: false,
      })
    ).toBe('no-qbt');
  });

  it('returns no-qbt even when folders+items exist (Case A wins per AC #1 priority)', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: false,
        mediaLibrariesCount: 3,
        itemsCount: 10,
        isLoading: false,
      })
    ).toBe('no-qbt');
  });

  it('returns no-folder when qBT OK but no media libraries', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: true,
        mediaLibrariesCount: 0,
        itemsCount: 0,
        isLoading: false,
      })
    ).toBe('no-folder');
  });

  it('returns no-folder even if items somehow exist (Case B wins over C per AC #1 priority)', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: true,
        mediaLibrariesCount: 0,
        itemsCount: 5,
        isLoading: false,
      })
    ).toBe('no-folder');
  });

  it('returns ready-for-scan when qBT OK + folders OK + library empty', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: true,
        mediaLibrariesCount: 1,
        itemsCount: 0,
        isLoading: false,
      })
    ).toBe('ready-for-scan');
  });

  it('returns ready-for-scan when multiple folders + library empty', () => {
    expect(
      classifyEmptyState({
        qbtConfigured: true,
        mediaLibrariesCount: 7,
        itemsCount: 0,
        isLoading: false,
      })
    ).toBe('ready-for-scan');
  });
});
