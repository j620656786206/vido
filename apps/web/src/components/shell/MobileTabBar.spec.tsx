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
  it('renders the four pilot bottom-tabs plus a 更多 control', async () => {
    renderBar();
    expect(await screen.findByTestId('nav-home')).toBeInTheDocument();
    expect(screen.getByTestId('nav-library')).toBeInTheDocument();
    expect(screen.getByTestId('nav-discover')).toBeInTheDocument();
    expect(screen.getByTestId('nav-downloads')).toBeInTheDocument();
    expect(screen.getByTestId('nav-more')).toBeInTheDocument();
  });

  it('marks the active tab via router matching', async () => {
    renderBar('/discover');
    const discover = await screen.findByTestId('nav-discover');
    expect(discover).toHaveAttribute('data-status', 'active');
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
