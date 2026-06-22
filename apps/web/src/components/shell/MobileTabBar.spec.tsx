import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';

vi.mock('../../hooks/useStatusSummary', () => ({
  useStatusSummary: () => ({ data: undefined }),
}));

import { MobileTabBar } from './MobileTabBar';

function renderBar(path = '/') {
  const rootRoute = createRootRoute({ component: () => React.createElement(MobileTabBar) });
  const mk = (p: string) =>
    createRoute({ getParentRoute: () => rootRoute, path: p, component: () => null });
  const routeTree = rootRoute.addChildren([
    mk('/'),
    mk('/library'),
    mk('/activity'),
    mk('/discover'),
    mk('/downloads'),
    mk('/settings'),
  ]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [path] }),
  });
  return render(React.createElement(RouterProvider, { router }));
}

describe('MobileTabBar', () => {
  it('renders the four bottom-tabs (首頁·媒體庫·活動·下載) plus a 更多 control', async () => {
    renderBar();
    expect(await screen.findByTestId('nav-home')).toBeInTheDocument();
    expect(screen.getByTestId('nav-library')).toBeInTheDocument();
    // 活動 went live in ux3-2-3, taking the 3rd slot (探索 moved to the More sheet).
    expect(screen.getByTestId('nav-activity')).toBeInTheDocument();
    expect(screen.getByTestId('nav-downloads')).toBeInTheDocument();
    expect(screen.getByTestId('nav-more')).toBeInTheDocument();
  });

  it('marks the active tab via router matching', async () => {
    renderBar('/activity');
    const activity = await screen.findByTestId('nav-activity');
    expect(activity).toHaveAttribute('data-status', 'active');
  });

  it('opens the More sheet (revealing 設定) when 更多 is tapped', async () => {
    renderBar();
    fireEvent.click(await screen.findByTestId('nav-more'));
    expect(await screen.findByTestId('nav-settings')).toBeInTheDocument();
  });

  it('the bar carries the primary-navigation aria-label', async () => {
    renderBar();
    expect(await screen.findByTestId('mobile-tab-bar')).toHaveAttribute('aria-label', '主要導航');
  });
});
