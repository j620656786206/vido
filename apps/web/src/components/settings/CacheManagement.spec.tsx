import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { CacheManagement } from './CacheManagement';

// Mock the hooks
vi.mock('../../hooks/useCacheStats', () => ({
  useCacheStats: vi.fn(),
  useClearCacheByType: vi.fn(),
  useClearCacheByAge: vi.fn(),
  useClearAllCache: vi.fn(),
}));

import { useCacheStats, useClearCacheByType, useClearCacheByAge } from '../../hooks/useCacheStats';

const mockUseCacheStats = vi.mocked(useCacheStats);
const mockUseClearByType = vi.mocked(useClearCacheByType);
const mockUseClearByAge = vi.mocked(useClearCacheByAge);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(React.createElement(QueryClientProvider, { client: queryClient }, ui));
}

beforeEach(() => {
  // Default mock implementations
  mockUseClearByType.mockReturnValue({
    mutateAsync: vi.fn(),
    isPending: false,
  } as any);
  mockUseClearByAge.mockReturnValue({
    mutateAsync: vi.fn(),
    isPending: false,
  } as any);
});

describe('CacheManagement', () => {
  it('renders loading state', () => {
    mockUseCacheStats.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByTestId('cache-loading')).toBeInTheDocument();
  });

  it('renders error state', () => {
    mockUseCacheStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Network error'),
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByTestId('cache-error')).toBeInTheDocument();
    expect(screen.getByText('Network error')).toBeInTheDocument();
  });

  it('renders cache types when data is loaded', () => {
    mockUseCacheStats.mockReturnValue({
      data: {
        cacheTypes: [
          { type: 'image', label: '圖片快取', sizeBytes: 1024, entryCount: 10 },
          { type: 'ai', label: 'AI 解析快取', sizeBytes: 512, entryCount: 5 },
        ],
        totalSizeBytes: 1536,
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByTestId('cache-management')).toBeInTheDocument();
    expect(screen.getByTestId('cache-types-list')).toBeInTheDocument();
    expect(screen.getByTestId('cache-type-image')).toBeInTheDocument();
    expect(screen.getByTestId('cache-type-ai')).toBeInTheDocument();
  });

  it('renders total size in header', () => {
    mockUseCacheStats.mockReturnValue({
      data: {
        cacheTypes: [],
        totalSizeBytes: 1073741824,
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByText(/1\.0 GB/)).toBeInTheDocument();
  });

  it('renders clear old cache button', () => {
    mockUseCacheStats.mockReturnValue({
      data: {
        cacheTypes: [],
        totalSizeBytes: 0,
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByTestId('clear-old-cache-btn')).toBeInTheDocument();
    expect(screen.getByTestId('clear-old-cache-btn')).toHaveTextContent('清除 30 天前的快取');
  });

  it('renders heading text', () => {
    mockUseCacheStats.mockReturnValue({
      data: {
        cacheTypes: [],
        totalSizeBytes: 0,
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByText('快取管理')).toBeInTheDocument();
  });
});
