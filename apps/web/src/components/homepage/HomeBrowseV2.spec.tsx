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
vi.mock('./ContinueWatchingSlot', () => ({
  ContinueWatchingSlot: () => React.createElement('div', { 'data-testid': 'stub-cw' }, 'cw'),
}));
vi.mock('./RecentlyAddedRowV2', () => ({
  RecentlyAddedRowV2: () =>
    React.createElement('div', { 'data-testid': 'stub-recent-v2' }, 'recent'),
}));

import { HomeBrowseV2 } from './HomeBrowseV2';

describe('HomeBrowseV2 (Home v2 composition)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('[P1] D3 ordering law — own-content (繼續觀看 + 最近新增) is structurally ABOVE Hero + Explore', () => {
    render(<HomeBrowseV2 />);
    const root = screen.getByTestId('home-v2-root');

    const order = Array.from(
      root.querySelectorAll<HTMLElement>(
        '[data-testid="stub-cw"], [data-testid="stub-recent-v2"], [data-testid="stub-hero"], [data-testid="stub-explore"]'
      )
    ).map((el) => el.getAttribute('data-testid'));

    // Both own-content blocks precede both external-curation blocks, in this order.
    expect(order).toEqual(['stub-cw', 'stub-recent-v2', 'stub-hero', 'stub-explore']);
  });

  it('[P1] D3 ordering law — own-content zone DOM-precedes the Hero', () => {
    render(<HomeBrowseV2 />);
    const ownContent = screen.getByTestId('home-own-content');
    const hero = screen.getByTestId('stub-hero');
    // compareDocumentPosition: FOLLOWING bit set means `hero` comes after own-content.
    expect(
      ownContent.compareDocumentPosition(hero) & Node.DOCUMENT_POSITION_FOLLOWING
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

  it('[P2] both own-content blocks render — the reserved slot is never silently dropped', () => {
    render(<HomeBrowseV2 />);
    expect(screen.getByTestId('stub-cw')).toBeInTheDocument();
    expect(screen.getByTestId('stub-recent-v2')).toBeInTheDocument();
  });
});
