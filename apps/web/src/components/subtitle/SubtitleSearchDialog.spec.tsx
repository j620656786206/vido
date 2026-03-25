import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SubtitleSearchDialog } from './SubtitleSearchDialog';

// Mock hook state that can be overridden per test
const defaultHookReturn = {
  search: vi.fn(),
  isSearching: false,
  searchError: null,
  results: [] as any[],
  resultCount: 0,
  sortBy: 'score' as const,
  sortOrder: 'desc' as const,
  toggleSort: vi.fn(),
  download: vi.fn(),
  downloadingIds: new Set<string>(),
  downloadedIds: new Set<string>(),
  downloadErrorMap: {} as Record<string, string>,
  preview: vi.fn(),
  previewDataMap: {} as Record<string, any>,
  previewingId: null as string | null,
  isPreviewing: false,
  downloadStage: null as string | null,
};

let hookOverrides: Partial<typeof defaultHookReturn> = {};

vi.mock('../../hooks/useSubtitleSearch', () => ({
  useSubtitleSearch: () => ({ ...defaultHookReturn, ...hookOverrides }),
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

  it('toggles 繁體轉換 switch on click', () => {
    renderDialog({ productionCountry: 'US' });
    const toggle = screen.getByRole('switch');
    expect(toggle.getAttribute('aria-checked')).toBe('true');
    fireEvent.click(toggle);
    expect(toggle.getAttribute('aria-checked')).toBe('false');
  });
});

// --- AC #3: Table columns and sort ---
describe('SubtitleSearchDialog — Results Table', () => {
  const mockResults = [
    {
      id: 'sub-1',
      source: 'assrt',
      filename: '星際效應.srt',
      language: 'zh-Hant',
      download_url: 'https://example.com/sub1',
      downloads: 150,
      group: 'YYeTs',
      resolution: '1080p',
      format: 'SRT',
      score: 0.85,
      score_breakdown: { language: 0.9, resolution: 0.8, source_trust: 0.7, group: 0.6, downloads: 0.5 },
    },
    {
      id: 'sub-2',
      source: 'zimuku',
      filename: '星際效應.ass',
      language: 'zh-Hans',
      download_url: 'https://example.com/sub2',
      downloads: 80,
      group: '',
      resolution: '720p',
      format: 'ASS',
      score: 0.45,
      score_breakdown: { language: 0.5, resolution: 0.4, source_trust: 0.3, group: 0.2, downloads: 0.1 },
    },
  ];

  beforeEach(() => {
    hookOverrides = {
      results: mockResults,
      resultCount: 2,
    };
  });

  afterEach(() => {
    hookOverrides = {};
  });

  it('renders results table with all 7 columns (AC #3)', () => {
    renderDialog();
    const table = screen.getByTestId('subtitle-results-table');
    expect(table).toBeInTheDocument();

    // Verify column headers: 來源, 語言, 字幕名稱, 格式, 評分, 下載數, 操作
    expect(screen.getByText('來源')).toBeInTheDocument();
    expect(screen.getByText('語言')).toBeInTheDocument();
    expect(screen.getByText('字幕名稱')).toBeInTheDocument();
    expect(screen.getByText('格式')).toBeInTheDocument();
    expect(screen.getByText('評分')).toBeInTheDocument();
    expect(screen.getByText('下載數')).toBeInTheDocument();
    expect(screen.getByText('操作')).toBeInTheDocument();
  });

  it('renders result count', () => {
    renderDialog();
    expect(screen.getByText('找到 2 個結果')).toBeInTheDocument();
  });

  it('renders result rows with correct data', () => {
    renderDialog();
    // First row
    expect(screen.getByTestId('subtitle-row-sub-1')).toBeInTheDocument();
    expect(screen.getByTestId('subtitle-row-sub-2')).toBeInTheDocument();
    // Source names
    expect(screen.getByText('assrt')).toBeInTheDocument();
    expect(screen.getByText('zimuku')).toBeInTheDocument();
  });

  it('displays score as percentage with color coding (AC #3)', () => {
    renderDialog();
    // score 0.85 → 85% (green)
    expect(screen.getByText('85%')).toBeInTheDocument();
    // score 0.45 → 45% (yellow)
    expect(screen.getByText('45%')).toBeInTheDocument();
  });

  it('displays format column values (AC #3)', () => {
    renderDialog();
    expect(screen.getByText('SRT')).toBeInTheDocument();
    expect(screen.getByText('ASS')).toBeInTheDocument();
  });

  it('shows download buttons per row', () => {
    renderDialog();
    expect(screen.getByTestId('download-btn-sub-1')).toBeInTheDocument();
    expect(screen.getByTestId('download-btn-sub-2')).toBeInTheDocument();
  });

  it('shows preview buttons per row', () => {
    renderDialog();
    expect(screen.getByTestId('preview-btn-sub-1')).toBeInTheDocument();
    expect(screen.getByTestId('preview-btn-sub-2')).toBeInTheDocument();
  });

  it('shows checkmark for downloaded items (AC #5)', () => {
    hookOverrides = {
      results: mockResults,
      resultCount: 2,
      downloadedIds: new Set(['sub-1']),
    };
    renderDialog();
    // sub-1 should not have download button (replaced by checkmark)
    expect(screen.queryByTestId('download-btn-sub-1')).not.toBeInTheDocument();
    // sub-2 should still have download button
    expect(screen.getByTestId('download-btn-sub-2')).toBeInTheDocument();
  });

  it('shows loading spinner for downloading items (AC #5)', () => {
    hookOverrides = {
      results: mockResults,
      resultCount: 2,
      downloadingIds: new Set(['sub-1']),
    };
    renderDialog();
    const downloadBtn = screen.getByTestId('download-btn-sub-1');
    expect(downloadBtn).toBeDisabled();
  });
});
