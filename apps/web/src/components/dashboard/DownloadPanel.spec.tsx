import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  createMemoryHistory,
  RouterProvider,
  createRootRoute,
  createRoute,
  createRouter,
} from '@tanstack/react-router';
import React from 'react';
import { DownloadPanel } from './DownloadPanel';

vi.mock('../../hooks/useDownloads', () => ({
  useDownloads: vi.fn(),
}));

vi.mock('../../hooks/useQBittorrent', () => ({
  useQBittorrentConfig: vi.fn(),
}));

import { useDownloads } from '../../hooks/useDownloads';
import { useQBittorrentConfig } from '../../hooks/useQBittorrent';

const mockUseDownloads = vi.mocked(useDownloads);
const mockUseQBConfig = vi.mocked(useQBittorrentConfig);

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const rootRoute = createRootRoute({
    component: () => React.createElement(DownloadPanel),
  });
  const downloadsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/downloads',
    component: () => React.createElement('div', null, 'Downloads'),
  });
  const settingsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings/qbittorrent',
    component: () => React.createElement('div', null, 'Settings'),
  });

  const routeTree = rootRoute.addChildren([downloadsRoute, settingsRoute]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });

  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(RouterProvider, { router } as any)
    );
  };
}

function renderPanel() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const rootRoute = createRootRoute({
    component: () => React.createElement(DownloadPanel),
  });
  const downloadsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/downloads',
    component: () => React.createElement('div', null, 'Downloads'),
  });
  const settingsRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/settings/qbittorrent',
    component: () => React.createElement('div', null, 'Settings'),
  });

  const routeTree = rootRoute.addChildren([downloadsRoute, settingsRoute]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });

  return render(
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(RouterProvider, { router } as any)
    )
  );
}

const mockDownloads = [
  {
    hash: 'abc123',
    name: '[SubGroup] Movie Name (2024) [1080p]',
    size: 4294967296,
    progress: 0.85,
    downloadSpeed: 1048576,
    uploadSpeed: 0,
    eta: 300,
    status: 'downloading' as const,
    addedOn: '2026-02-10T10:00:00Z',
    seeds: 5,
    peers: 3,
    downloaded: 3650722201,
    uploaded: 0,
    ratio: 0,
    savePath: '/downloads',
  },
  {
    hash: 'def456',
    name: 'Another Download',
    size: 1073741824,
    progress: 0.45,
    downloadSpeed: 524288,
    uploadSpeed: 0,
    eta: 600,
    status: 'downloading' as const,
    addedOn: '2026-02-10T11:00:00Z',
    seeds: 2,
    peers: 1,
    downloaded: 483183820,
    uploaded: 0,
    ratio: 0,
    savePath: '/downloads',
  },
];

function mockConnected(downloads = mockDownloads) {
  mockUseQBConfig.mockReturnValue({
    data: { host: 'http://localhost:8080', username: 'admin', basePath: '', configured: true },
    isLoading: false,
    error: null,
  } as ReturnType<typeof useQBittorrentConfig>);
  mockUseDownloads.mockReturnValue({
    data: downloads,
    isLoading: false,
    isSuccess: true,
    error: null,
  } as ReturnType<typeof useDownloads>);
}

function mockDisconnected() {
  mockUseQBConfig.mockReturnValue({
    data: { host: '', username: '', basePath: '', configured: false },
    isLoading: false,
    error: null,
  } as ReturnType<typeof useQBittorrentConfig>);
  mockUseDownloads.mockReturnValue({
    data: undefined,
    isLoading: false,
    isSuccess: false,
    error: new Error('Not configured'),
  } as ReturnType<typeof useDownloads>);
}

function mockLoading() {
  mockUseQBConfig.mockReturnValue({
    data: undefined,
    isLoading: true,
    error: null,
  } as ReturnType<typeof useQBittorrentConfig>);
  mockUseDownloads.mockReturnValue({
    data: undefined,
    isLoading: true,
    isSuccess: false,
    error: null,
  } as ReturnType<typeof useDownloads>);
}

describe('DownloadPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('[P1] renders panel title', async () => {
    mockConnected();
    renderPanel();
    const heading = await screen.findByRole('heading', { name: '下載中' });
    expect(heading).toBeTruthy();
  });

  it('[P1] renders download items when connected', async () => {
    mockConnected();
    renderPanel();
    expect(await screen.findByText('[SubGroup] Movie Name (2024) [1080p]')).toBeTruthy();
    expect(screen.getByText('Another Download')).toBeTruthy();
  });

  it('[P1] shows disconnected state when not configured', async () => {
    mockDisconnected();
    renderPanel();
    expect(await screen.findByText('qBittorrent 未連線')).toBeTruthy();
  });

  it('[P1] shows download count badge when connected', async () => {
    mockConnected();
    renderPanel();
    expect(await screen.findByText('2')).toBeTruthy();
  });

  it('[P1] shows "查看全部下載" link', async () => {
    mockConnected();
    renderPanel();
    expect(await screen.findByText(/查看全部下載/)).toBeTruthy();
  });

  it('[P1] shows loading state', async () => {
    mockLoading();
    renderPanel();
    expect(await screen.findByTestId('download-panel-loading')).toBeTruthy();
  });

  it('[P1] shows empty state when no downloads', async () => {
    mockConnected([]);
    renderPanel();
    expect(await screen.findByText('目前沒有下載任務')).toBeTruthy();
  });

  it('[P2] limits displayed downloads to 5', async () => {
    const manyDownloads = Array.from({ length: 8 }, (_, i) => ({
      ...mockDownloads[0],
      hash: `hash-${i}`,
      name: `Download ${i}`,
    }));
    mockConnected(manyDownloads);
    renderPanel();
    expect(await screen.findByText('Download 0')).toBeTruthy();
    expect(screen.getByText('Download 4')).toBeTruthy();
    expect(screen.queryByText('Download 5')).toBeNull();
  });

  it('[P1] has download-panel test id', async () => {
    mockConnected();
    renderPanel();
    expect(await screen.findByTestId('download-panel')).toBeTruthy();
  });

  it('[P2] shows progress bar for each download', async () => {
    mockConnected();
    renderPanel();
    await screen.findByText('[SubGroup] Movie Name (2024) [1080p]');
    const progressBars = screen.getAllByRole('progressbar');
    expect(progressBars.length).toBe(2);
  });

  it('[P2] shows "前往設定" link when disconnected', async () => {
    mockDisconnected();
    renderPanel();
    expect(await screen.findByText('前往設定')).toBeTruthy();
  });
});
