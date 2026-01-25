/**
 * MetadataEditorDialog Tests (Story 3.8 - AC1, AC4)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MetadataEditorDialog } from './MetadataEditorDialog';
import type { MediaMetadata } from './MetadataEditorDialog';

// Mock the hooks
const mockMutateAsync = vi.fn();
vi.mock('../../hooks/useMetadataEditor', () => ({
  useUpdateMetadata: vi.fn(() => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
    error: null,
  })),
  useUploadPoster: vi.fn(() => ({
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  })),
}));

function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
    },
  });
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createTestQueryClient();
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

describe('MetadataEditorDialog', () => {
  const defaultInitialData: MediaMetadata = {
    id: 'test-media-id',
    mediaType: 'movie',
    title: '鬼滅之刃',
    titleEnglish: 'Demon Slayer',
    year: 2019,
    genres: ['animation', 'action'],
    director: '外崎春雄',
    cast: ['花江夏樹', '鬼頭明里'],
    overview: '大正時代的日本，善良的少年炭治郎...',
    posterUrl: 'https://example.com/poster.jpg',
  };

  const defaultProps = {
    isOpen: true,
    onClose: vi.fn(),
    mediaId: 'test-media-id',
    mediaType: 'movie' as const,
    initialData: defaultInitialData,
    onSuccess: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
    mockMutateAsync.mockResolvedValue({});
  });

  it('renders dialog when isOpen is true', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);
    expect(screen.getByTestId('metadata-editor-dialog')).toBeTruthy();
  });

  it('does not render when isOpen is false', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId('metadata-editor-dialog')).toBeNull();
  });

  it('renders with initial data pre-filled', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    expect(screen.getByDisplayValue('鬼滅之刃')).toBeTruthy();
    expect(screen.getByDisplayValue('Demon Slayer')).toBeTruthy();
    expect(screen.getByDisplayValue('2019')).toBeTruthy();
    expect(screen.getByDisplayValue('外崎春雄')).toBeTruthy();
    expect(screen.getByText('花江夏樹')).toBeTruthy();
    expect(screen.getByText('鬼頭明里')).toBeTruthy();
  });

  it('calls onClose when close button is clicked', async () => {
    const onClose = vi.fn();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} onClose={onClose} />);

    const closeButton = screen.getByLabelText('關閉');
    await userEvent.click(closeButton);

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when cancel button is clicked', async () => {
    const onClose = vi.fn();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} onClose={onClose} />);

    const cancelButton = screen.getByRole('button', { name: '取消' });
    await userEvent.click(cancelButton);

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('calls onClose when Escape key is pressed', async () => {
    const onClose = vi.fn();
    renderWithProviders(<MetadataEditorDialog {...defaultProps} onClose={onClose} />);

    fireEvent.keyDown(document, { key: 'Escape' });

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledTimes(1);
    });
  });

  it('shows validation error when title is empty', async () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    // Clear the title field
    const titleInput = screen.getByDisplayValue('鬼滅之刃');
    await userEvent.clear(titleInput);

    // Submit the form
    const saveButton = screen.getByRole('button', { name: /儲存/ });
    await userEvent.click(saveButton);

    await waitFor(() => {
      expect(screen.getByText('標題為必填')).toBeTruthy();
    });
  });

  it('shows validation error for invalid year', async () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    const yearInput = screen.getByDisplayValue('2019');
    await userEvent.clear(yearInput);
    await userEvent.type(yearInput, '1800');

    const saveButton = screen.getByRole('button', { name: /儲存/ });
    await userEvent.click(saveButton);

    await waitFor(() => {
      expect(screen.getByText('年份必須大於 1900')).toBeTruthy();
    });
  });

  it('toggles genre selection', async () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    // Find the 動畫 genre button (should be selected)
    const animationButton = screen.getByRole('button', { name: '動畫' });
    expect(animationButton.className).toContain('bg-blue-600');

    // Click to deselect
    await userEvent.click(animationButton);
    expect(animationButton.className).not.toContain('bg-blue-600');

    // Click to select again
    await userEvent.click(animationButton);
    expect(animationButton.className).toContain('bg-blue-600');
  });

  it('adds and removes cast members', async () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    // Add a new cast member
    const castInput = screen.getByPlaceholderText(/輸入演員名稱後按 Enter/);
    await userEvent.type(castInput, '下野紘{enter}');

    expect(screen.getByText('下野紘')).toBeTruthy();

    // Remove a cast member - find the X button inside the cast member span
    const castSpan = screen.getByText('下野紘').closest('span');
    const removeButton = castSpan?.querySelector('button');
    if (removeButton) {
      await userEvent.click(removeButton);
    }

    expect(screen.queryByText('下野紘')).toBeNull();
  });

  it('submits form with updated data', async () => {
    const onSuccess = vi.fn();
    const onClose = vi.fn();
    renderWithProviders(
      <MetadataEditorDialog {...defaultProps} onSuccess={onSuccess} onClose={onClose} />
    );

    // Modify the title
    const titleInput = screen.getByDisplayValue('鬼滅之刃');
    await userEvent.clear(titleInput);
    await userEvent.type(titleInput, '鬼滅之刃：無限城篇');

    // Submit the form
    const saveButton = screen.getByRole('button', { name: /儲存/ });
    await userEvent.click(saveButton);

    await waitFor(() => {
      expect(mockMutateAsync).toHaveBeenCalledWith(
        expect.objectContaining({
          id: 'test-media-id',
          mediaType: 'movie',
          title: '鬼滅之刃：無限城篇',
        })
      );
    });

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });

  it('disables save button when form is not dirty', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    const saveButton = screen.getByRole('button', { name: /儲存/ });
    expect(saveButton.hasAttribute('disabled')).toBe(true);
  });

  it('renders all form fields per AC1', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    // Title (Chinese)
    expect(screen.getByText('標題（中文）')).toBeTruthy();
    // Title (English)
    expect(screen.getByText('標題（英文）')).toBeTruthy();
    // Year
    expect(screen.getByText('年份')).toBeTruthy();
    // Genres
    expect(screen.getByText('類型')).toBeTruthy();
    // Director
    expect(screen.getByText('導演')).toBeTruthy();
    // Cast
    expect(screen.getByText('演員')).toBeTruthy();
    // Overview
    expect(screen.getByText('簡介')).toBeTruthy();
    // Poster URL
    expect(screen.getByText('海報圖片網址')).toBeTruthy();
  });

  it('renders genre options', () => {
    renderWithProviders(<MetadataEditorDialog {...defaultProps} />);

    // Check some genre options are rendered
    expect(screen.getByRole('button', { name: '動作' })).toBeTruthy();
    expect(screen.getByRole('button', { name: '動畫' })).toBeTruthy();
    expect(screen.getByRole('button', { name: '劇情' })).toBeTruthy();
    expect(screen.getByRole('button', { name: '科幻' })).toBeTruthy();
  });
});
