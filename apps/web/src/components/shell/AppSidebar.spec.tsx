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

vi.mock('../../hooks/useLibrary', () => ({
  useLibraryStats: () => ({ data: { movieCount: 1284, tvCount: 86 } }),
}));
vi.mock('../../hooks/useStatusSummary', () => ({
  useStatusSummary: () => ({ data: undefined }),
}));

import { AppSidebar } from './AppSidebar';

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
