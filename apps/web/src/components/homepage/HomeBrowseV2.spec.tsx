import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import React from 'react';

// Stub every child so the composition test focuses on STRUCTURE (the D3 ordering
// law) — the sections are covered by their own specs. Stubs let us assert raw DOM
// order without pulling in data fetching or the router.
vi.mock('./HeroBanner', () => ({
  HeroBanner: () => React.createElement('div', { 'data-testid': 'stub-hero' }, 'hero'),
}));
vi.mock('./ExploreBlocksList', () => ({
  ExploreBlocksList: () => React.createElement('div', { 'data-testid': 'stub-explore' }, 'explore'),
}));
vi.mock('./RecentlyAddedRowV2', () => ({
  RecentlyAddedRowV2: () =>
    React.createElement('div', { 'data-testid': 'stub-recent-v2' }, 'recent'),
}));
vi.mock('./HomeReadoutBand', () => ({
  HomeReadoutBand: () => React.createElement('div', { 'data-testid': 'stub-readout' }, 'readout'),
}));

import { HomeBrowseV2 } from './HomeBrowseV2';

describe('HomeBrowseV2 (Home v2 composition)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] D3 ordering law (v3 form) — own content (hero + 最近新增) is structurally ABOVE the TMDb tail', () => {
    render(<HomeBrowseV2 />);
    const root = screen.getByTestId('home-v2-root');

    const order = Array.from(
      root.querySelectorAll<HTMLElement>(
        '[data-testid="stub-recent-v2"], [data-testid="stub-hero"], [data-testid="stub-explore"]'
      )
    ).map((el) => el.getAttribute('data-testid'));

    // ux3-1-8: the hero is OWN content now (own-library newest-with-backdrop),
    // so the entire first fold is yours — hero, then 最近新增, then the TMDb
    // tail last. D3 own-above-external holds and strengthens.
    expect(order).toEqual(['stub-hero', 'stub-recent-v2', 'stub-explore']);
  });

  it('[P1] D3 ordering law — the TMDb tail DOM-follows the own-content zone', () => {
    render(<HomeBrowseV2 />);
    const ownContent = screen.getByTestId('home-own-content');
    const explore = screen.getByTestId('stub-explore');
    // compareDocumentPosition: FOLLOWING bit set means `explore` comes after own-content.
    expect(
      ownContent.compareDocumentPosition(explore) & Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
  });

  it('[P2] ux3-1-4 — dashboard remnants (downloads / qB / connection history) are absent from the v2 home', () => {
    render(<HomeBrowseV2 />);
    // D3 guardrail #3: home is curation-first; these belong to Activity/status now.
    expect(screen.queryByTestId('stub-downloads')).toBeNull();
    expect(screen.queryByTestId('stub-qb')).toBeNull();
    expect(screen.queryByTestId('download-panel')).toBeNull();
    expect(screen.queryByTestId('qb-status-indicator')).toBeNull();
  });

  it('[P2] own-content renders; the CW reserved slot stays OUT until Epic 17 (⚖️ R3)', () => {
    render(<HomeBrowseV2 />);
    expect(screen.queryByTestId('stub-cw')).toBeNull(); // R3: unmounted until Epic 17
    expect(screen.getByTestId('stub-recent-v2')).toBeInTheDocument();
  });

  it('[P1] ux3-1-7 — the readout band is the FIRST section, above everything (讀數先於瀏覽)', () => {
    render(<HomeBrowseV2 />);
    const root = screen.getByTestId('home-v2-root');
    const order = Array.from(
      root.querySelectorAll<HTMLElement>(
        '[data-testid="stub-readout"], [data-testid="stub-recent-v2"], [data-testid="stub-hero"], [data-testid="stub-explore"]'
      )
    ).map((el) => el.getAttribute('data-testid'));
    expect(order[0]).toBe('stub-readout');
  });
});
