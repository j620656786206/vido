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

vi.mock('../../hooks/useLibrary', () => ({
  useLibraryStats: () => ({ data: { movieCount: 1284, tvCount: 86 } }),
}));
vi.mock('../../hooks/useStatusSummary', () => ({
  useStatusSummary: () => ({ data: undefined }),
}));
// feat-nav-badge-inflight-jobs — default: no jobs (badge absent). Tests flip it.
let mockInflight: number | undefined;
vi.mock('../../hooks/useActivity', () => ({
  useInflightJobCount: () => mockInflight,
}));

import { AppSidebar } from './AppSidebar';

// LogoutButton (rendered by SidebarFooter) owns real router + query-client hooks;
// this suite is about the shell chrome, so stub it out. Its own behaviour is
// covered by LogoutButton.spec.tsx.
vi.mock('./LogoutButton', () => ({ LogoutButton: () => null }));

function renderSidebar(opts: { collapsed?: boolean; onToggle?: () => void; path?: string } = {}) {
  const { collapsed = false, onToggle = () => {}, path = '/' } = opts;
  const rootRoute = createRootRoute({
    component: () => React.createElement(AppSidebar, { collapsed, onToggleCollapse: onToggle }),
  });
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

describe('AppSidebar', () => {
  it('renders the live destinations (內容 + 任務), not the still-deferred 系統', async () => {
    renderSidebar();
    expect(await screen.findByTestId('nav-home')).toBeInTheDocument();
    expect(screen.getByTestId('nav-library')).toBeInTheDocument();
    expect(screen.getByTestId('nav-movies')).toBeInTheDocument();
    expect(screen.getByTestId('nav-tv')).toBeInTheDocument();
    expect(screen.getByTestId('nav-discover')).toBeInTheDocument();
    // 活動 went live in ux3-2-3.
    expect(screen.getByTestId('nav-activity')).toBeInTheDocument();
    expect(screen.getByTestId('nav-downloads')).toBeInTheDocument();
    expect(screen.getByTestId('nav-settings')).toBeInTheDocument();
    // 系統 is still deferred (route not built yet).
    expect(screen.queryByTestId('nav-system')).not.toBeInTheDocument();
  });

  // The <aside> was the only named landmark; the <nav> inside it was anonymous,
  // so AT announced two nested regions and could name only one.
  it('names the nav inside the sidebar distinctly from the sidebar itself', async () => {
    renderSidebar();
    const aside = await screen.findByTestId('app-sidebar');
    expect(aside).toHaveAttribute('aria-label', '主要導航');
    const nav = aside.querySelector('nav');
    expect(nav).not.toBeNull();
    expect(nav).toHaveAttribute('aria-label', '內容與任務');
    expect(nav!.getAttribute('aria-label')).not.toBe(aside.getAttribute('aria-label'));
  });

  it('shows library counts from the stats query', async () => {
    renderSidebar();
    await screen.findByTestId('nav-movies');
    expect(screen.getByText('1,284')).toBeInTheDocument();
    expect(screen.getByText('86')).toBeInTheDocument();
  });

  it('calls onToggleCollapse when the collapse control is clicked (expanded)', async () => {
    const onToggle = vi.fn();
    renderSidebar({ onToggle });
    fireEvent.click(await screen.findByTestId('sidebar-collapse-toggle'));
    expect(onToggle).toHaveBeenCalledTimes(1);
  });

  it('collapsed: renders the rail with an expand control and hides group labels', async () => {
    renderSidebar({ collapsed: true });
    expect(await screen.findByTestId('sidebar-expand-toggle')).toBeInTheDocument();
    expect(screen.getByTestId('app-sidebar')).toHaveAttribute('data-collapsed', 'true');
    expect(screen.queryByText('內容')).not.toBeInTheDocument();
  });

  it('marks the active destination via TanStack router matching', async () => {
    renderSidebar({ path: '/downloads' });
    const downloads = await screen.findByTestId('nav-downloads');
    expect(downloads).toHaveAttribute('data-status', 'active');
  });
});

describe('AppSidebar — in-flight job badge (feat-nav-badge-inflight-jobs)', () => {
  afterEach(() => {
    mockInflight = undefined;
  });

  it('活動 wears the count when jobs are running, and speaks it to AT', async () => {
    mockInflight = 3;
    renderSidebar();
    const activity = await screen.findByTestId('nav-activity');
    expect(screen.getByTestId('nav-activity-badge')).toHaveTextContent('3');
    expect(activity).toHaveAttribute('aria-label', '活動（3 個任務進行中）');
  });

  it('badge is ABSENT at zero and while the source is degraded/loading', async () => {
    mockInflight = 0;
    renderSidebar();
    await screen.findByTestId('nav-activity');
    expect(screen.queryByTestId('nav-activity-badge')).toBeNull();
    expect(screen.getByTestId('nav-activity')).toHaveAttribute('aria-label', '活動');
  });

  it('the readout survives the collapsed rail', async () => {
    mockInflight = 2;
    renderSidebar({ collapsed: true });
    const activity = await screen.findByTestId('nav-activity');
    expect(screen.getByTestId('nav-activity-badge')).toHaveTextContent('2');
    expect(activity).toHaveAttribute('aria-label', '活動（2 個任務進行中）');
  });

  it('the badge is a wash readout, never a solid accent fill (配給強調)', async () => {
    mockInflight = 5;
    renderSidebar();
    const badge = await screen.findByTestId('nav-activity-badge');
    // The wash lives on the breathing halo behind the digit, not on the badge
    // box itself (InFlightBadge) — assert on the subtree so the rule survives
    // that split. The figure keeps the accent TEXT twin either way.
    expect(badge.innerHTML).toContain('bg-[var(--accent-subtle)]');
    expect(badge.className).toContain('text-[var(--accent-text)]');
    expect(badge.outerHTML).not.toContain('bg-[var(--accent-primary)]');
  });
});
