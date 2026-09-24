import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
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
  // The critique's Error-Prevention finding: this one-click purge sat ten
  // pixels above per-type clears that DO confirm. Same grammar now.
  describe('清除 30 天前的快取 is a two-step confirm', () => {
    const arm = async () => {
      // Shape must match CleanupResult — a wrong key here made the result banner
      // render undefined.toLocaleString() and blow up only under CI timing.
      const clearOld = vi.fn().mockResolvedValue({ entriesRemoved: 3, bytesReclaimed: 0 });
      mockUseClearByAge.mockReturnValue({ mutateAsync: clearOld, isPending: false } as any);
      mockUseCacheStats.mockReturnValue({
        data: { totalSizeBytes: 0, cacheTypes: [] },
        isLoading: false,
      } as any);
      renderWithQuery(React.createElement(CacheManagement));
      const user = userEvent.setup();
      await user.click(await screen.findByTestId('clear-old-cache-btn'));
      return { user, clearOld };
    };

    it('first click only arms — nothing is cleared yet', async () => {
      const { clearOld } = await arm();
      expect(clearOld).not.toHaveBeenCalled();
      expect(screen.getByTestId('clear-old-cache-btn')).toHaveTextContent('確認清除');
      expect(screen.getByTestId('clear-old-cache-cancel-btn')).toBeInTheDocument();
    });

    it('second click actually clears', async () => {
      const { user, clearOld } = await arm();
      await user.click(screen.getByTestId('clear-old-cache-btn'));
      expect(clearOld).toHaveBeenCalledWith(30);
    });

    it('取消 disarms without clearing', async () => {
      const { user, clearOld } = await arm();
      await user.click(screen.getByTestId('clear-old-cache-cancel-btn'));
      expect(clearOld).not.toHaveBeenCalled();
      expect(screen.getByTestId('clear-old-cache-btn')).not.toHaveTextContent('確認');
    });
  });

  // 固定詞彙: green means IN PROGRESS. 已清除 is a report of done-ness and
  // wears neutral — spending green on "done" devalues the green of 已連線.
  it('reports a completed clear in neutral, not success green', async () => {
    const clearOld = vi.fn().mockResolvedValue({ entriesRemoved: 5, bytesReclaimed: 0 });
    mockUseClearByAge.mockReturnValue({ mutateAsync: clearOld, isPending: false } as any);
    mockUseCacheStats.mockReturnValue({
      data: { totalSizeBytes: 0, cacheTypes: [] },
      isLoading: false,
    } as any);
    renderWithQuery(React.createElement(CacheManagement));
    const user = userEvent.setup();
    await user.click(await screen.findByTestId('clear-old-cache-btn'));
    await user.click(screen.getByTestId('clear-old-cache-btn'));
    const banner = await screen.findByTestId('cache-result');
    expect(banner.className).toContain('bg-[var(--bg-tertiary)]');
    expect(banner.className).not.toContain('success');
  });

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
    // dsr-3e: the backend's words stay off the page (C16 shape).
    expect(screen.queryByText('Network error')).toBeNull();
    expect(screen.getByText('與後端的連線中斷了。快取本身不受影響。')).toBeInTheDocument();
  });

  it('renders cache types when data is loaded', () => {
    mockUseCacheStats.mockReturnValue({
      data: {
        cacheTypes: [
          { type: 'wikipedia', label: '維基百科快取', sizeBytes: 1024, entryCount: 10 },
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
    expect(screen.getByTestId('cache-type-wikipedia')).toBeInTheDocument();
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
    // Title lives at the route level now (one header contract); the component
    // keeps the live total readout.
    expect(screen.getByText(/總計/)).toBeInTheDocument();
  });

  it('disables clear old cache button while pending', () => {
    mockUseCacheStats.mockReturnValue({
      data: {
        cacheTypes: [],
        totalSizeBytes: 0,
      },
      isLoading: false,
      error: null,
    } as any);
    mockUseClearByAge.mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: true,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    const btn = screen.getByTestId('clear-old-cache-btn');
    expect(btn).toBeDisabled();
  });

  it('displays error message text from error object', () => {
    mockUseCacheStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Connection refused'),
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByText('無法載入快取資訊')).toBeInTheDocument();
    expect(screen.queryByText('Connection refused')).toBeNull();
  });

  // No 圖片快取 since bugfix-custom-posters-served-and-not-cache: four types.
  it('renders all 4 cache type cards when data has 4 types', () => {
    mockUseCacheStats.mockReturnValue({
      data: {
        cacheTypes: [
          { type: 'ai', label: 'AI 解析快取', sizeBytes: 512, entryCount: 5 },
          { type: 'metadata', label: 'TMDb 中繼資料', sizeBytes: 256, entryCount: 3 },
          { type: 'douban', label: '豆瓣快取', sizeBytes: 128, entryCount: 2 },
          { type: 'wikipedia', label: '維基百科快取', sizeBytes: 64, entryCount: 1 },
        ],
        totalSizeBytes: 960,
      },
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.queryByTestId('cache-type-image')).toBeNull();
    expect(screen.getByTestId('cache-type-ai')).toBeInTheDocument();
    expect(screen.getByTestId('cache-type-metadata')).toBeInTheDocument();
    expect(screen.getByTestId('cache-type-douban')).toBeInTheDocument();
    expect(screen.getByTestId('cache-type-wikipedia')).toBeInTheDocument();
  });

  // dsr-3e: with no data and no finished attempt yet, this is the loading
  // state (the old fallback「總計 —」rendered a page with nothing in it).
  it('shows the loading state when stats are not yet loaded', () => {
    mockUseCacheStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: null,
    } as any);

    renderWithQuery(React.createElement(CacheManagement));
    expect(screen.getByTestId('cache-loading')).toBeInTheDocument();
  });

  describe('dsr-3e', () => {
    const stats = {
      totalSizeBytes: 1024,
      cacheTypes: [{ type: 'ai', label: 'AI 解析快取', sizeBytes: 1024, entryCount: 3 }],
    };
    const loaded = () =>
      mockUseCacheStats.mockReturnValue({ data: stats, isLoading: false, error: null } as any);
    const WARNING =
      '再按一次才會真的清除。這會刪掉 30 天前的所有快取，之後第一次瀏覽會比較慢，但不會影響影片與字幕檔案。';

    it('the first press arms it and explains what the second press does; no request yet', async () => {
      const user = userEvent.setup();
      const mutateAsync = vi.fn();
      mockUseClearByAge.mockReturnValue({ mutateAsync, isPending: false } as any);
      loaded();
      renderWithQuery(React.createElement(CacheManagement));
      expect(screen.queryByTestId('clear-old-cache-warning')).toBeNull();
      await user.click(screen.getByTestId('clear-old-cache-btn'));
      const warning = screen.getByRole('status');
      expect(warning).toHaveTextContent(WARNING);
      expect(warning).toHaveAttribute('id', 'clear-old-cache-warning');
      expect(screen.getByTestId('clear-old-cache-btn')).toHaveAttribute(
        'aria-describedby',
        'clear-old-cache-warning'
      );
      expect(mutateAsync).not.toHaveBeenCalled();
      await user.click(screen.getByTestId('clear-old-cache-cancel-btn'));
      expect(screen.queryByTestId('clear-old-cache-warning')).toBeNull();
    });

    it('the button’s name is the full sentence on every width (the phone shows a short label)', async () => {
      const user = userEvent.setup();
      loaded();
      renderWithQuery(React.createElement(CacheManagement));
      expect(screen.getByRole('button', { name: '清除 30 天前的快取' })).toBeInTheDocument();
      await user.click(screen.getByTestId('clear-old-cache-btn'));
      expect(screen.getByRole('button', { name: '確認清除 30 天前的快取' })).toBeInTheDocument();
    });

    it('sections are 16 apart', () => {
      loaded();
      renderWithQuery(React.createElement(CacheManagement));
      expect(screen.getByTestId('cache-management').className).toContain('space-y-4');
    });

    it('load failure: 重試 refetches; mid-refetch stays on the error page as 重試中…', async () => {
      const user = userEvent.setup();
      const refetch = vi.fn();
      mockUseCacheStats.mockReturnValue({
        data: undefined,
        isFetched: true,
        isFetching: true,
        error: null,
        refetch,
      } as any);
      renderWithQuery(React.createElement(CacheManagement));
      expect(screen.queryByTestId('cache-loading')).toBeNull();
      expect(screen.getByRole('button', { name: '重試中…' })).toBeInTheDocument();
      mockUseCacheStats.mockReturnValue({
        data: undefined,
        isFetched: true,
        isFetching: false,
        error: new Error('x'),
        refetch,
      } as any);
      renderWithQuery(React.createElement(CacheManagement));
      await user.click(screen.getAllByRole('button', { name: '重試' })[0]);
      expect(refetch).toHaveBeenCalled();
    });
  });

  describe('dsr-3e CR', () => {
    const stats = {
      totalSizeBytes: 1,
      cacheTypes: [{ type: 'ai', label: 'AI 解析快取', sizeBytes: 1, entryCount: 1 }],
    };
    it('the live region is there (empty) before the first press, so the warning is announced', async () => {
      const user = userEvent.setup();
      mockUseCacheStats.mockReturnValue({ data: stats, isLoading: false, error: null } as any);
      renderWithQuery(React.createElement(CacheManagement));
      const region = screen.getByTestId('clear-old-cache-warning-region');
      expect(region).toHaveAttribute('role', 'status');
      expect(region).toBeEmptyDOMElement();
      await user.click(screen.getByTestId('clear-old-cache-btn'));
      expect(screen.getByTestId('clear-old-cache-warning-region')).toBe(region);
      expect(region).not.toBeEmptyDOMElement();
    });

    it('while clearing, the button no longer points at a warning that is gone', async () => {
      const user = userEvent.setup();
      mockUseCacheStats.mockReturnValue({ data: stats, isLoading: false, error: null } as any);
      const { rerender } = renderWithQuery(React.createElement(CacheManagement));
      await user.click(screen.getByTestId('clear-old-cache-btn'));
      mockUseClearByAge.mockReturnValue({ mutateAsync: vi.fn(), isPending: true } as any);
      rerender(
        React.createElement(
          QueryClientProvider,
          { client: new QueryClient() },
          React.createElement(CacheManagement)
        )
      );
      expect(screen.getByTestId('clear-old-cache-btn')).not.toHaveAttribute('aria-describedby');
      expect(screen.queryByTestId('clear-old-cache-warning')).toBeNull();
    });
  });
});
