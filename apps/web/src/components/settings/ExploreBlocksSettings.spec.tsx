import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { ExploreBlocksSettings } from './ExploreBlocksSettings';
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
});
