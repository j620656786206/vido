import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SubtitleSearchDialog } from './SubtitleSearchDialog';

// Mock the hook
vi.mock('../../hooks/useSubtitleSearch', () => ({
  useSubtitleSearch: () => ({
    search: vi.fn(),
    isSearching: false,
    searchError: null,
    results: [],
    resultCount: 0,
    sortBy: 'score' as const,
    sortOrder: 'desc' as const,
    toggleSort: vi.fn(),
    download: vi.fn(),
    downloadingIds: new Set<string>(),
    downloadedIds: new Set<string>(),
    downloadError: null,
    preview: vi.fn(),
    previewDataMap: {},
    previewingId: null,
    isPreviewing: false,
    downloadStage: null,
  }),
}));

function renderDialog(props?: Partial<Parameters<typeof SubtitleSearchDialog>[0]>) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

  const defaultProps = {
    mediaId: 'movie-1',
    mediaType: 'movie' as const,
    mediaTitle: 'Test Movie',
    mediaFilePath: '/media/movie.mkv',
    open: true,
    onOpenChange: vi.fn(),
  };

  return render(
    <QueryClientProvider client={queryClient}>
      <SubtitleSearchDialog {...defaultProps} {...props} />
    </QueryClientProvider>,
  );
}

describe('SubtitleSearchDialog', () => {
  it('renders when open', () => {
    renderDialog();
    expect(screen.getByTestId('subtitle-search-dialog')).toBeInTheDocument();
  });

  it('does not render when closed', () => {
    renderDialog({ open: false });
    expect(screen.queryByTestId('subtitle-search-dialog')).not.toBeInTheDocument();
  });

  it('pre-fills search input with media title', () => {
    renderDialog({ mediaTitle: '星際效應' });
    const input = screen.getByTestId('subtitle-search-input') as HTMLInputElement;
    expect(input.value).toBe('星際效應');
  });

  it('shows all provider checkboxes checked by default', () => {
    renderDialog();
    const assrt = screen.getByTestId('provider-assrt') as HTMLInputElement;
    const opensub = screen.getByTestId('provider-opensubtitles') as HTMLInputElement;
    const zimuku = screen.getByTestId('provider-zimuku') as HTMLInputElement;
    expect(assrt.checked).toBe(true);
    expect(opensub.checked).toBe(true);
    expect(zimuku.checked).toBe(true);
  });

  it('toggles provider checkbox on click', () => {
    renderDialog();
    const assrt = screen.getByTestId('provider-assrt') as HTMLInputElement;
    fireEvent.click(assrt);
    expect(assrt.checked).toBe(false);
  });

  it('shows empty state when no results', () => {
    renderDialog();
    expect(screen.getByTestId('subtitle-empty-state')).toBeInTheDocument();
  });

  it('shows 繁體轉換 toggle ON by default for non-CN content', () => {
    renderDialog({ productionCountry: 'US' });
    const toggle = screen.getByRole('switch');
    expect(toggle.getAttribute('aria-checked')).toBe('true');
  });

  it('shows 繁體轉換 toggle OFF for CN content', () => {
    renderDialog({ productionCountry: 'CN' });
    const toggle = screen.getByRole('switch');
    expect(toggle.getAttribute('aria-checked')).toBe('false');
  });

  it('calls onOpenChange when close button clicked', () => {
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });
    fireEvent.click(screen.getByLabelText('關閉'));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });

  it('calls onOpenChange on backdrop click', () => {
    const onOpenChange = vi.fn();
    renderDialog({ onOpenChange });
    fireEvent.click(screen.getByTestId('subtitle-search-dialog'));
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});
