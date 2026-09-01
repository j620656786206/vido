import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, afterEach } from 'vitest';
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
let mockInflight: number | undefined;
vi.mock('../../hooks/useActivity', () => ({
  useInflightJobCount: () => mockInflight,
}));

import { MobileTabBar } from './MobileTabBar';

// LogoutButton (rendered by SidebarFooter) owns real router + query-client hooks;
// this suite is about the shell chrome, so stub it out. Its own behaviour is
// covered by LogoutButton.spec.tsx.
vi.mock('./LogoutButton', () => ({ LogoutButton: () => null }));

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

describe('MobileTabBar — in-flight job badge (feat-nav-badge-inflight-jobs)', () => {
  afterEach(() => {
    mockInflight = undefined;
  });

  it('活動 tab wears the count and speaks it to AT', async () => {
    mockInflight = 3;
    renderBar();
    const activity = await screen.findByTestId('nav-activity');
    expect(screen.getByTestId('nav-activity-badge')).toHaveTextContent('3');
    expect(activity).toHaveAttribute('aria-label', '活動（3 個任務進行中）');
  });

  it('badge absent at zero / while unmeasured', async () => {
    renderBar();
    await screen.findByTestId('nav-activity');
    expect(screen.queryByTestId('nav-activity-badge')).toBeNull();
    expect(screen.getByTestId('nav-activity')).toHaveAttribute('aria-label', '活動');
  });
});
