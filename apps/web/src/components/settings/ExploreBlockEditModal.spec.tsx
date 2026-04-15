import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import { ExploreBlockEditModal } from './ExploreBlockEditModal';
import type { ExploreBlock } from '../../services/exploreBlockService';

const createMutation = { mutateAsync: vi.fn(async () => undefined), isPending: false };
const updateMutation = { mutateAsync: vi.fn(async () => undefined), isPending: false };

vi.mock('../../hooks/useExploreBlocks', () => ({
  useCreateExploreBlock: () => createMutation,
  useUpdateExploreBlock: () => updateMutation,
}));

function renderModal(props: { block?: ExploreBlock } = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onClose = vi.fn();
  const result = render(
    React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(ExploreBlockEditModal, { block: props.block, onClose })
    )
  );
  return { ...result, onClose };
}

describe('ExploreBlockEditModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('create mode: submits new block and closes', async () => {
    const { onClose } = renderModal();

    fireEvent.change(screen.getByTestId('explore-block-name-input'), {
      target: { value: '新區塊' },
    });
    fireEvent.change(screen.getByTestId('explore-block-type-select'), {
      target: { value: 'tv' },
    });
    fireEvent.change(screen.getByTestId('explore-block-max-items-input'), {
      target: { value: '15' },
    });

    fireEvent.click(screen.getByTestId('explore-block-save-button'));
    // Promises flush with one microtask tick
    await Promise.resolve();
    await Promise.resolve();

    expect(createMutation.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({
        name: '新區塊',
        contentType: 'tv',
        maxItems: 15,
      })
    );
    expect(updateMutation.mutateAsync).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('edit mode: preloads block fields and calls update with the block id', async () => {
    const block: ExploreBlock = {
      id: 'blk-1',
      name: '既有',
      contentType: 'movie',
      genreIds: '28,12',
      language: 'zh-TW',
      region: 'TW',
      sortBy: 'vote_average.desc',
      maxItems: 10,
      sortOrder: 0,
      createdAt: '',
      updatedAt: '',
    };
    const { onClose } = renderModal({ block });

    const nameInput = screen.getByTestId('explore-block-name-input') as HTMLInputElement;
    expect(nameInput.value).toBe('既有');
    expect((screen.getByTestId('explore-block-region-input') as HTMLInputElement).value).toBe('TW');

    fireEvent.change(nameInput, { target: { value: '改名' } });
    fireEvent.click(screen.getByTestId('explore-block-save-button'));
    await Promise.resolve();
    await Promise.resolve();

    expect(updateMutation.mutateAsync).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'blk-1', name: '改名' })
    );
    expect(createMutation.mutateAsync).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('disables save button when name is empty', () => {
    renderModal();
    expect(screen.getByTestId('explore-block-save-button')).toBeDisabled();

    fireEvent.change(screen.getByTestId('explore-block-name-input'), {
      target: { value: '有名字' },
    });
    expect(screen.getByTestId('explore-block-save-button')).not.toBeDisabled();
  });

  it('calls onClose when 取消 is clicked', () => {
    const { onClose } = renderModal();
    fireEvent.click(screen.getByText('取消'));
    expect(onClose).toHaveBeenCalled();
  });
});
