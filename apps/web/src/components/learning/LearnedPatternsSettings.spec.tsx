/**
 * LearnedPatternsSettings Tests (Story 3.9 - AC3)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { LearnedPatternsSettings } from './LearnedPatternsSettings';
import { learningService } from '../../services/learning';

// Mock the learning service
vi.mock('../../services/learning', () => ({
  learningService: {
    listPatterns: vi.fn(),
    deletePattern: vi.fn(),
  },
}));

const mockPatterns = [
  {
    id: 'pattern-1',
    pattern: '[Leopard-Raws] Kimetsu no Yaiba',
    patternType: 'fansub',
    fansubGroup: 'Leopard-Raws',
    titlePattern: 'Kimetsu no Yaiba',
    metadataType: 'series',
    metadataId: 'series-123',
    tmdbId: 85937,
    confidence: 1.0,
    useCount: 12,
    createdAt: '2026-01-20T10:00:00Z',
  },
  {
    id: 'pattern-2',
    pattern: 'Breaking Bad',
    patternType: 'standard',
    titlePattern: 'Breaking Bad',
    metadataType: 'series',
    metadataId: 'series-456',
    tmdbId: 1396,
    confidence: 1.0,
    useCount: 5,
    createdAt: '2026-01-18T10:00:00Z',
  },
];

const mockStats = {
  totalPatterns: 2,
  totalApplied: 17,
  mostUsedPattern: '[Leopard-Raws] Kimetsu no Yaiba',
  mostUsedCount: 12,
};

describe('LearnedPatternsSettings', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state initially', () => {
    vi.mocked(learningService.listPatterns).mockImplementation(
      () => new Promise(() => {})
    );

    render(<LearnedPatternsSettings />);

    expect(screen.getByTestId('patterns-loading')).toBeInTheDocument();
  });

  it('displays patterns count in Chinese (AC3)', async () => {
    vi.mocked(learningService.listPatterns).mockResolvedValue({
      patterns: mockPatterns,
      totalCount: 2,
      stats: mockStats,
    });

    render(<LearnedPatternsSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('patterns-count')).toHaveTextContent(
        '已記住 2 個自訂規則'
      );
    });
  });

  it('displays pattern statistics', async () => {
    vi.mocked(learningService.listPatterns).mockResolvedValue({
      patterns: mockPatterns,
      totalCount: 2,
      stats: mockStats,
    });

    render(<LearnedPatternsSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('patterns-stats')).toBeInTheDocument();
    });

    expect(screen.getByText('共套用 17 次')).toBeInTheDocument();
    expect(screen.getByText('[Leopard-Raws] Kimetsu no Yaiba')).toBeInTheDocument();
    expect(screen.getByText('(12 次)')).toBeInTheDocument();
  });

  it('displays pattern list with details', async () => {
    vi.mocked(learningService.listPatterns).mockResolvedValue({
      patterns: mockPatterns,
      totalCount: 2,
      stats: mockStats,
    });

    render(<LearnedPatternsSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('patterns-list')).toBeInTheDocument();
    });

    expect(screen.getByTestId('pattern-item-pattern-1')).toBeInTheDocument();
    expect(screen.getByTestId('pattern-item-pattern-2')).toBeInTheDocument();

    // Pattern names should be visible
    expect(screen.getByText('[Leopard-Raws] Kimetsu no Yaiba')).toBeInTheDocument();
    expect(screen.getByText('Breaking Bad')).toBeInTheDocument();

    // Pattern types should be visible
    expect(screen.getByText('fansub')).toBeInTheDocument();
    expect(screen.getByText('standard')).toBeInTheDocument();
  });

  it('expands pattern to show details when clicked', async () => {
    const user = userEvent.setup();
    vi.mocked(learningService.listPatterns).mockResolvedValue({
      patterns: mockPatterns,
      totalCount: 2,
      stats: mockStats,
    });

    render(<LearnedPatternsSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('patterns-list')).toBeInTheDocument();
    });

    // Click to expand first pattern
    await user.click(screen.getByText('[Leopard-Raws] Kimetsu no Yaiba'));

    // Should show details
    expect(screen.getByTestId('pattern-details-pattern-1')).toBeInTheDocument();
    expect(screen.getByText('字幕組：')).toBeInTheDocument();
    expect(screen.getByText('Leopard-Raws')).toBeInTheDocument();
    expect(screen.getByText('TMDb ID：')).toBeInTheDocument();
    expect(screen.getByText('85937')).toBeInTheDocument();
  });

  it('deletes pattern when delete button is clicked', async () => {
    const user = userEvent.setup();
    vi.mocked(learningService.listPatterns).mockResolvedValue({
      patterns: mockPatterns,
      totalCount: 2,
      stats: mockStats,
    });
    vi.mocked(learningService.deletePattern).mockResolvedValue(undefined);

    render(<LearnedPatternsSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('patterns-list')).toBeInTheDocument();
    });

    // Expand pattern to see delete button
    await user.click(screen.getByText('[Leopard-Raws] Kimetsu no Yaiba'));

    // Click delete
    await user.click(screen.getByTestId('delete-pattern-pattern-1'));

    await waitFor(() => {
      expect(learningService.deletePattern).toHaveBeenCalledWith('pattern-1');
    });

    // Pattern should be removed from list
    expect(screen.queryByTestId('pattern-item-pattern-1')).not.toBeInTheDocument();
    expect(screen.getByTestId('patterns-count')).toHaveTextContent(
      '已記住 1 個自訂規則'
    );
  });

  it('shows empty state when no patterns exist', async () => {
    vi.mocked(learningService.listPatterns).mockResolvedValue({
      patterns: [],
      totalCount: 0,
      stats: { totalPatterns: 0, totalApplied: 0 },
    });

    render(<LearnedPatternsSettings />);

    await waitFor(() => {
      expect(screen.getByTestId('empty-patterns')).toBeInTheDocument();
    });

    expect(screen.getByText('尚無自訂規則')).toBeInTheDocument();
    expect(
      screen.getByText('在手動配對檔案後，可選擇學習規則以便未來自動套用')
    ).toBeInTheDocument();
  });

  it('calls onError when fetching patterns fails', async () => {
    const error = new Error('Failed to fetch');
    const onError = vi.fn();
    vi.mocked(learningService.listPatterns).mockRejectedValue(error);

    render(<LearnedPatternsSettings onError={onError} />);

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith(error);
    });
  });

  it('calls onError when deleting pattern fails', async () => {
    const user = userEvent.setup();
    const error = new Error('Failed to delete');
    const onError = vi.fn();

    vi.mocked(learningService.listPatterns).mockResolvedValue({
      patterns: mockPatterns,
      totalCount: 2,
      stats: mockStats,
    });
    vi.mocked(learningService.deletePattern).mockRejectedValue(error);

    render(<LearnedPatternsSettings onError={onError} />);

    await waitFor(() => {
      expect(screen.getByTestId('patterns-list')).toBeInTheDocument();
    });

    // Expand and try to delete
    await user.click(screen.getByText('[Leopard-Raws] Kimetsu no Yaiba'));
    await user.click(screen.getByTestId('delete-pattern-pattern-1'));

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith(error);
    });
  });
});
