import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { Film } from 'lucide-react';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';
import { SidebarNavItem, type SidebarNavItemProps } from './SidebarNavItem';

function renderItem(props: SidebarNavItemProps, initialPath = '/') {
  const rootRoute = createRootRoute({
    component: () => React.createElement('div', null, React.createElement(SidebarNavItem, props)),
  });
  const mk = (path: string) =>
    createRoute({ getParentRoute: () => rootRoute, path, component: () => null });
  const routeTree = rootRoute.addChildren([
    mk('/'),
    mk('/library'),
    mk('/discover'),
    mk('/downloads'),
    mk('/settings'),
  ]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: [initialPath] }),
  });
  return render(React.createElement(RouterProvider, { router }));
}

const base: SidebarNavItemProps = { to: '/library', label: '電影', icon: Film, navKey: 'movies' };

describe('SidebarNavItem', () => {
  it('renders the label and a nav-{key} testid with the correct href', async () => {
    renderItem(base);
    const link = await screen.findByTestId('nav-movies');
    expect(link).toHaveTextContent('電影');
    expect(link).toHaveAttribute('href', '/library');
  });

  it('renders a Mono count when provided', async () => {
    renderItem({ ...base, count: 1284 });
    await screen.findByTestId('nav-movies');
    expect(screen.getByText('1,284')).toBeInTheDocument();
  });

  it('marks the item active (data-status) when the route matches', async () => {
    renderItem({ to: '/downloads', label: '下載', icon: Film, navKey: 'downloads' }, '/downloads');
    const link = await screen.findByTestId('nav-downloads');
    expect(link).toHaveAttribute('data-status', 'active');
  });

  it('is not active on a non-matching route', async () => {
    renderItem({ to: '/downloads', label: '下載', icon: Film, navKey: 'downloads' }, '/discover');
    const link = await screen.findByTestId('nav-downloads');
    expect(link).not.toHaveAttribute('data-status', 'active');
  });

  it('collapsed: renders icon-only with an aria-label carrying the count', async () => {
    renderItem({ ...base, collapsed: true, count: 86 });
    const link = await screen.findByTestId('nav-movies');
    expect(link).toHaveAttribute('aria-label', '電影（86）');
  });
});
