import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { PosterCardV2 } from './PosterCardV2';
import type { LibraryMovie } from '../../types/library';

const media = (
  over: Partial<LibraryMovie> = {}
): Pick<LibraryMovie, 'parseStatus' | 'subtitleTracks'> => ({
  parseStatus: 'success',
  subtitleTracks: JSON.stringify([{ language: 'zh-Hant' }]),
  ...over,
});

function renderCard(props: React.ComponentProps<typeof PosterCardV2>) {
  const rootRoute = createRootRoute({
    component: () => React.createElement('div', null, React.createElement(PosterCardV2, props)),
  });
  const detail = createRoute({
    getParentRoute: () => rootRoute,
    path: '/media/$type/$id',
    component: () => null,
  });
  const router = createRouter({
    routeTree: rootRoute.addChildren([detail]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  return render(React.createElement(RouterProvider, { router }));
}

const base = {
  id: 'abc',
  type: 'movie' as const,
  title: '銀翼殺手 2049',
  posterPath: null,
  year: '2017',
  meta: '163 分',
  voteAverage: 8,
  media: media(),
};

describe('PosterCardV2', () => {
  it('renders the title + mono meta and links to the detail route', async () => {
    renderCard(base);
    const link = await screen.findByTestId('poster-v2-abc');
    expect(link).toHaveTextContent('銀翼殺手 2049');
    expect(link).toHaveTextContent('2017 · 163 分');
    expect(link).toHaveAttribute('href', '/media/movie/abc');
  });

  it('shows the subtitle status badge (繁中) for an in-library item', async () => {
    renderCard(base);
    await screen.findByTestId('poster-v2-abc');
    expect(screen.getByTestId('poster-status-badge')).toHaveTextContent('繁中');
  });

  it('surfaces a lifecycle exception (失敗) over the subtitle badge', async () => {
    renderCard({ ...base, media: media({ parseStatus: 'failed' }) });
    await screen.findByTestId('poster-v2-abc');
    expect(screen.getByTestId('poster-status-badge')).toHaveTextContent('失敗');
  });

  it('omits the badge when status is unknown (F3 — no error)', async () => {
    renderCard({ ...base, media: { parseStatus: 'success', subtitleTracks: undefined } });
    await screen.findByTestId('poster-v2-abc');
    expect(screen.queryByTestId('poster-status-badge')).not.toBeInTheDocument();
  });
});
