import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { ExploreBlocksSettings } from './ExploreBlocksSettings';
import { MOVIE_SORT_OPTIONS, sortLabel } from './exploreBlockSort';
import type { ExploreBlock } from '../../services/exploreBlockService';

const listResult = {
  data: undefined as { blocks: ExploreBlock[] } | undefined,
  isLoading: false,
  isError: false,
};

const createMutation = {
  mutateAsync: vi.fn(async () => undefined),
  isPending: false,
};
const updateMutation = {
  mutateAsync: vi.fn(async () => undefined),
  isPending: false,
};
const deleteMutation = {
  mutateAsync: vi.fn(async () => undefined),
  isPending: false,
};
const reorderMutation = {
  mutateAsync: vi.fn(async () => undefined),
  isPending: false,
};

vi.mock('../../hooks/useExploreBlocks', () => ({
  useExploreBlocks: () => listResult,
  useCreateExploreBlock: () => createMutation,
  useUpdateExploreBlock: () => updateMutation,
  useDeleteExploreBlock: () => deleteMutation,
  useReorderExploreBlocks: () => reorderMutation,
}));

function makeBlock(overrides: Partial<ExploreBlock>): ExploreBlock {
  return {
    id: 'b1',
    name: 'Block',
    contentType: 'movie',
    genreIds: '',
    language: '',
    region: '',
    sortBy: 'popularity.desc',
    maxItems: 20,
    sortOrder: 0,
    createdAt: '',
    updatedAt: '',
    ...overrides,
  };
}

function renderSettings() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(ExploreBlocksSettings)
    )
  );
}

describe('ExploreBlocksSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listResult.data = { blocks: [] };
    listResult.isLoading = false;
    listResult.isError = false;
  });

  it('shows empty state when no blocks configured', () => {
    listResult.data = { blocks: [] };
    renderSettings();
    expect(screen.getByTestId('explore-blocks-empty')).toBeInTheDocument();
  });

  it('renders a row per configured block', () => {
    listResult.data = {
      blocks: [
        makeBlock({ id: 'a', name: '熱門電影' }),
        makeBlock({ id: 'b', name: '熱門影集', contentType: 'tv' }),
      ],
    };
    renderSettings();

    expect(screen.getByTestId('explore-block-row-a')).toBeInTheDocument();
    expect(screen.getByTestId('explore-block-row-b')).toBeInTheDocument();
    expect(screen.getByText('熱門電影')).toBeInTheDocument();
    expect(screen.getByText('熱門影集')).toBeInTheDocument();

    // bugfix-10-6 AC #4 — the content-type marker is a lucide icon + plain
    // label (Film/Tv), never the 🎬/📺 emoji.
    const movieMeta = screen.getByText(/^電影 ·/);
    const tvMeta = screen.getByText(/^影集 ·/);
    expect(movieMeta).toBeInTheDocument();
    expect(tvMeta).toBeInTheDocument();
    // dsr-3d: the icon moved OUT of the description line to its own 18px
    // column (C10 XmC7a); the words 電影／影集 stay in the line as plain text.
    expect(movieMeta.querySelector('svg')).toBeNull();
    expect(screen.getByTestId('explore-block-type-icon-a').tagName.toLowerCase()).toBe('svg');
    expect(screen.getByTestId('explore-block-type-icon-b').tagName.toLowerCase()).toBe('svg');
    expect(screen.queryByText(/🎬|📺/)).toBeNull();
  });

  it('opens create modal when 新增區塊 clicked', () => {
    listResult.data = { blocks: [] };
    renderSettings();

    fireEvent.click(screen.getByTestId('explore-blocks-add-button'));
    expect(screen.getByTestId('explore-block-edit-modal')).toBeInTheDocument();
  });

  it('opens edit modal with block data pre-populated', () => {
    const block = makeBlock({ id: 'x', name: '某區塊' });
    listResult.data = { blocks: [block] };
    renderSettings();

    fireEvent.click(screen.getByTestId('explore-block-edit-x'));
    expect(screen.getByTestId('explore-block-edit-modal')).toBeInTheDocument();
    const nameInput = screen.getByTestId('explore-block-name-input') as HTMLInputElement;
    expect(nameInput.value).toBe('某區塊');
  });

  it('shows delete confirmation before calling delete mutation', async () => {
    listResult.data = { blocks: [makeBlock({ id: 'del', name: '刪我' })] };
    renderSettings();

    fireEvent.click(screen.getByTestId('explore-block-delete-del'));
    expect(screen.getByTestId('explore-block-delete-confirm')).toBeInTheDocument();
    expect(deleteMutation.mutateAsync).not.toHaveBeenCalled();

    fireEvent.click(screen.getByTestId('explore-block-delete-confirm-button'));
    expect(deleteMutation.mutateAsync).toHaveBeenCalledWith('del');
  });

  it('calls reorder with new order when user moves a block up (AC #3)', async () => {
    listResult.data = {
      blocks: [
        makeBlock({ id: 'a', name: 'A', sortOrder: 0 }),
        makeBlock({ id: 'b', name: 'B', sortOrder: 1 }),
        makeBlock({ id: 'c', name: 'C', sortOrder: 2 }),
      ],
    };
    renderSettings();

    fireEvent.click(screen.getByTestId('explore-block-move-up-c'));
    expect(reorderMutation.mutateAsync).toHaveBeenCalledWith(['a', 'c', 'b']);
  });

  it('disables up arrow on the first row and down arrow on the last row', () => {
    listResult.data = {
      blocks: [makeBlock({ id: 'a' }), makeBlock({ id: 'b' })],
    };
    renderSettings();

    expect(screen.getByTestId('explore-block-move-up-a')).toBeDisabled();
    expect(screen.getByTestId('explore-block-move-down-b')).toBeDisabled();
  });

  it('shows error when delete mutation fails (H2 fix)', async () => {
    deleteMutation.mutateAsync = vi.fn(async () => {
      throw new Error('刪除失敗');
    });
    listResult.data = { blocks: [makeBlock({ id: 'err', name: '失敗' })] };
    renderSettings();

    fireEvent.click(screen.getByTestId('explore-block-delete-err'));
    fireEvent.click(screen.getByTestId('explore-block-delete-confirm-button'));

    await waitFor(() => {
      expect(screen.getByTestId('explore-blocks-operation-error')).toHaveTextContent('刪除失敗');
    });
  });

  it('shows error when reorder mutation fails (M1 fix)', async () => {
    reorderMutation.mutateAsync = vi.fn(async () => {
      throw new Error('排序失敗');
    });
    listResult.data = {
      blocks: [makeBlock({ id: 'a', sortOrder: 0 }), makeBlock({ id: 'b', sortOrder: 1 })],
    };
    renderSettings();

    fireEvent.click(screen.getByTestId('explore-block-move-up-b'));

    await waitFor(() => {
      expect(screen.getByTestId('explore-blocks-operation-error')).toHaveTextContent('排序失敗');
    });
  });

  it('closes delete confirmation on Escape key (L1 fix)', () => {
    listResult.data = { blocks: [makeBlock({ id: 'esc', name: 'Esc' })] };
    renderSettings();

    fireEvent.click(screen.getByTestId('explore-block-delete-esc'));
    expect(screen.getByTestId('explore-block-delete-confirm')).toBeInTheDocument();

    fireEvent.keyDown(document, { key: 'Escape' });
    expect(screen.queryByTestId('explore-block-delete-confirm')).toBeNull();
  });

  describe('dsr-3d', () => {
    const blocks = [
      makeBlock({
        id: 'a',
        name: '熱門電影',
        sortBy: 'popularity.desc',
        maxItems: 20,
        language: 'zh-TW',
        region: 'TW',
      }),
      makeBlock({
        id: 'b',
        name: '高分動畫',
        sortBy: 'vote_average.desc',
        maxItems: 15,
        genreIds: '16',
      }),
    ];

    it('the header counts the blocks', () => {
      listResult.data = { blocks };
      renderSettings();
      expect(screen.getByTestId('explore-blocks-count')).toHaveTextContent('2 個區塊');
    });

    it('each row says 電影／影集 · the sort in words · N 部 · language · region · genre', () => {
      listResult.data = { blocks };
      renderSettings();
      expect(screen.getByTestId('explore-block-desc-a')).toHaveTextContent(
        `電影 · ${sortLabel('popularity.desc')} · 20 部 · zh-TW · 地區 TW`
      );
      expect(screen.getByTestId('explore-block-desc-b')).toHaveTextContent(
        `電影 · ${sortLabel('vote_average.desc')} · 15 部 · 類型 16`
      );
      expect(screen.queryByText(/個項目/)).toBeNull();
    });

    it('rows are 12 apart, cards radius-lg, name semibold; buttons are solid squares', () => {
      listResult.data = { blocks };
      renderSettings();
      const row = screen.getByTestId('explore-block-row-a');
      expect(row.parentElement!.className).toContain('space-y-3');
      expect(row.className).toContain('rounded-[var(--radius-lg)]');
      expect(row.querySelector('h3')!.className).toContain('font-semibold');
      for (const id of ['move-up', 'move-down', 'edit', 'delete']) {
        const btn = screen.getByTestId(`explore-block-${id}-a`);
        expect(btn.className).toContain('bg-[var(--bg-tertiary)]');
        expect(btn.className).toContain('sm:size-8');
        expect(btn.className).toContain('size-11');
      }
      expect(screen.getByTestId('explore-blocks-settings').className).toContain('space-y-4');
    });

    it('ends with the one true sentence about owned titles', () => {
      listResult.data = { blocks };
      renderSettings();
      expect(screen.getByTestId('explore-blocks-owned-note')).toHaveTextContent(
        /^已擁有的作品不會出現在首頁。$/
      );
    });

    it('sort words come from the same list the edit modal offers', () => {
      for (const opt of MOVIE_SORT_OPTIONS) expect(sortLabel(opt.value)).toBe(opt.label);
      // An unknown stored value is shown as-is, not dropped.
      expect(sortLabel('weird.asc')).toBe('weird.asc');
    });
  });
});
