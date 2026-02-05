/**
 * LearnPatternPrompt Tests (Story 3.9 - AC1, AC2)
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  LearnPatternPrompt,
  PatternAppliedToast,
  LearnSuccessToast,
  type ExtractedPattern,
} from './LearnPatternPrompt';
import { learningService } from '../../services/learning';

// Mock the learning service
vi.mock('../../services/learning', () => ({
  learningService: {
    learnPattern: vi.fn(),
  },
}));

const mockExtractedPattern: ExtractedPattern = {
  fansubGroup: 'Leopard-Raws',
  titlePattern: 'Kimetsu no Yaiba',
  patternType: 'fansub',
};

const mockLearnedPattern = {
  id: 'pattern-456',
  pattern: '[Leopard-Raws] Kimetsu no Yaiba',
  patternType: 'fansub',
  fansubGroup: 'Leopard-Raws',
  titlePattern: 'Kimetsu no Yaiba',
  metadataType: 'series',
  metadataId: 'series-123',
  tmdbId: 85937,
  confidence: 1.0,
  useCount: 0,
  createdAt: '2026-01-20T10:00:00Z',
};

describe('LearnPatternPrompt', () => {
  const defaultProps = {
    filename: '[Leopard-Raws] Kimetsu no Yaiba - 26.mkv',
    extractedPattern: mockExtractedPattern,
    metadataId: 'series-123',
    metadataType: 'series' as const,
    tmdbId: 85937,
    onConfirm: vi.fn(),
    onSkip: vi.fn(),
    onError: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders prompt with pattern preview', () => {
    render(<LearnPatternPrompt {...defaultProps} />);

    expect(screen.getByTestId('learn-pattern-prompt')).toBeInTheDocument();
    expect(screen.getByText('學習此規則？')).toBeInTheDocument();
    expect(screen.getByText('系統偵測到以下規則，是否記住以便未來自動套用？')).toBeInTheDocument();
  });

  it('shows fansub group in blue and title in green', () => {
    render(<LearnPatternPrompt {...defaultProps} />);

    const preview = screen.getByTestId('pattern-preview');
    expect(preview).toHaveTextContent('[Leopard-Raws]');
    expect(preview).toHaveTextContent('Kimetsu no Yaiba');
  });

  it('shows pattern without fansub group when not present', () => {
    const propsWithoutFansub = {
      ...defaultProps,
      extractedPattern: {
        titlePattern: 'Breaking Bad',
        patternType: 'standard' as const,
      },
    };

    render(<LearnPatternPrompt {...propsWithoutFansub} />);

    const preview = screen.getByTestId('pattern-preview');
    expect(preview).toHaveTextContent('Breaking Bad');
    expect(preview).not.toHaveTextContent('[');
  });

  it('renders confirm and skip buttons', () => {
    render(<LearnPatternPrompt {...defaultProps} />);

    expect(screen.getByTestId('confirm-learn-button')).toBeInTheDocument();
    expect(screen.getByText('記住此規則')).toBeInTheDocument();

    expect(screen.getByTestId('skip-learn-button')).toBeInTheDocument();
    expect(screen.getByText('這次不用')).toBeInTheDocument();
  });

  it('calls onSkip when skip button is clicked', async () => {
    const user = userEvent.setup();
    render(<LearnPatternPrompt {...defaultProps} />);

    await user.click(screen.getByTestId('skip-learn-button'));

    expect(defaultProps.onSkip).toHaveBeenCalledTimes(1);
  });

  it('calls learningService and onConfirm when confirm button is clicked', async () => {
    const user = userEvent.setup();
    vi.mocked(learningService.learnPattern).mockResolvedValue(mockLearnedPattern);

    render(<LearnPatternPrompt {...defaultProps} />);

    await user.click(screen.getByTestId('confirm-learn-button'));

    await waitFor(() => {
      expect(learningService.learnPattern).toHaveBeenCalledWith({
        filename: '[Leopard-Raws] Kimetsu no Yaiba - 26.mkv',
        metadataId: 'series-123',
        metadataType: 'series',
        tmdbId: 85937,
      });
    });

    expect(defaultProps.onConfirm).toHaveBeenCalledWith(mockLearnedPattern);
  });

  it('shows loading state when confirming', async () => {
    const user = userEvent.setup();
    // Make the promise never resolve to keep loading state
    vi.mocked(learningService.learnPattern).mockImplementation(() => new Promise(() => {}));

    render(<LearnPatternPrompt {...defaultProps} />);

    await user.click(screen.getByTestId('confirm-learn-button'));

    expect(screen.getByTestId('confirm-learn-button')).toBeDisabled();
    expect(screen.getByTestId('skip-learn-button')).toBeDisabled();
  });

  it('calls onError when learning fails', async () => {
    const user = userEvent.setup();
    const error = new Error('Learning failed');
    vi.mocked(learningService.learnPattern).mockRejectedValue(error);

    render(<LearnPatternPrompt {...defaultProps} />);

    await user.click(screen.getByTestId('confirm-learn-button'));

    await waitFor(() => {
      expect(defaultProps.onError).toHaveBeenCalledWith(error);
    });

    expect(defaultProps.onConfirm).not.toHaveBeenCalled();
  });
});

describe('PatternAppliedToast', () => {
  it('renders toast with UX-5 feedback', () => {
    render(<PatternAppliedToast patternTitle="Kimetsu no Yaiba" />);

    expect(screen.getByTestId('pattern-applied-toast')).toBeInTheDocument();
    expect(screen.getByText('✓ 已套用你之前的設定')).toBeInTheDocument();
    expect(screen.getByText('（Kimetsu no Yaiba）')).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();

    render(<PatternAppliedToast patternTitle="Test" onClose={onClose} />);

    await user.click(screen.getByLabelText('關閉'));

    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('does not show close button when onClose is not provided', () => {
    render(<PatternAppliedToast patternTitle="Test" />);

    expect(screen.queryByLabelText('關閉')).not.toBeInTheDocument();
  });
});

describe('LearnSuccessToast', () => {
  it('renders toast with pattern name', () => {
    render(<LearnSuccessToast pattern="[Leopard-Raws] Kimetsu no Yaiba" />);

    expect(screen.getByTestId('learn-success-toast')).toBeInTheDocument();
    expect(screen.getByText('已學習此規則')).toBeInTheDocument();
    expect(screen.getByText('（[Leopard-Raws] Kimetsu no Yaiba）')).toBeInTheDocument();
  });

  it('calls onClose when close button is clicked', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();

    render(<LearnSuccessToast pattern="Test" onClose={onClose} />);

    await user.click(screen.getByLabelText('關閉'));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
