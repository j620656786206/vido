/**
 * LayeredProgressIndicator Tests (Story 3.10 - Task 5)
 */

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  LayeredProgressIndicator,
  InlineProgressIndicator,
  SourceChainIndicator,
} from './LayeredProgressIndicator';
import type { ParseStep } from './types';

const mockSteps: ParseStep[] = [
  { name: 'filename_extract', label: '解析檔名', status: 'success' },
  { name: 'tmdb_search', label: '搜尋 TMDb', status: 'in_progress' },
  { name: 'douban_search', label: '搜尋豆瓣', status: 'pending' },
  { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'pending' },
  { name: 'ai_retry', label: 'AI 重試', status: 'pending' },
  { name: 'download_poster', label: '下載海報', status: 'pending' },
];

const failedSteps: ParseStep[] = [
  { name: 'filename_extract', label: '解析檔名', status: 'success' },
  { name: 'tmdb_search', label: '搜尋 TMDb', status: 'failed', error: 'API timeout' },
  { name: 'douban_search', label: '搜尋豆瓣', status: 'failed', error: '無回應' },
  { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'skipped' },
  { name: 'ai_retry', label: 'AI 重試', status: 'pending' },
  { name: 'download_poster', label: '下載海報', status: 'pending' },
];

describe('LayeredProgressIndicator', () => {
  it('renders all steps', () => {
    render(<LayeredProgressIndicator steps={mockSteps} currentStep={1} />);

    expect(screen.getByTestId('layered-progress-indicator')).toBeInTheDocument();
    expect(screen.getByTestId('progress-step-filename_extract')).toBeInTheDocument();
    expect(screen.getByTestId('progress-step-tmdb_search')).toBeInTheDocument();
    expect(screen.getByTestId('progress-step-douban_search')).toBeInTheDocument();
  });

  it('displays step labels', () => {
    render(<LayeredProgressIndicator steps={mockSteps} currentStep={1} />);

    expect(screen.getByText('解析檔名')).toBeInTheDocument();
    expect(screen.getByText('搜尋 TMDb')).toBeInTheDocument();
    expect(screen.getByText('搜尋豆瓣')).toBeInTheDocument();
  });

  it('shows correct status for each step', () => {
    render(<LayeredProgressIndicator steps={mockSteps} currentStep={1} />);

    expect(screen.getByTestId('progress-step-filename_extract')).toHaveAttribute('data-status', 'success');
    expect(screen.getByTestId('progress-step-tmdb_search')).toHaveAttribute('data-status', 'in_progress');
    expect(screen.getByTestId('progress-step-douban_search')).toHaveAttribute('data-status', 'pending');
  });

  it('shows "搜尋中..." for in_progress step', () => {
    render(<LayeredProgressIndicator steps={mockSteps} currentStep={1} />);

    expect(screen.getByText('搜尋中...')).toBeInTheDocument();
  });

  it('shows error message for failed step', () => {
    render(<LayeredProgressIndicator steps={failedSteps} currentStep={2} />);

    expect(screen.getByText('API timeout')).toBeInTheDocument();
  });

  it('renders compact view without labels', () => {
    render(<LayeredProgressIndicator steps={mockSteps} currentStep={1} compact />);

    // Labels should not be present in compact mode
    expect(screen.queryByText('解析檔名')).not.toBeInTheDocument();
    expect(screen.queryByText('搜尋 TMDb')).not.toBeInTheDocument();
  });

  it('has correct ARIA attributes', () => {
    render(<LayeredProgressIndicator steps={mockSteps} currentStep={1} />);

    const container = screen.getByTestId('layered-progress-indicator');
    expect(container).toHaveAttribute('role', 'list');
    expect(container).toHaveAttribute('aria-label', '解析步驟進度');
  });

  it('applies custom className', () => {
    render(<LayeredProgressIndicator steps={mockSteps} currentStep={1} className="custom-class" />);

    expect(screen.getByTestId('layered-progress-indicator')).toHaveClass('custom-class');
  });
});

describe('InlineProgressIndicator', () => {
  it('renders step icons inline', () => {
    render(<InlineProgressIndicator steps={mockSteps} />);

    expect(screen.getByTestId('inline-progress-indicator')).toBeInTheDocument();
    // Should render 6 icons
    const container = screen.getByTestId('inline-progress-indicator');
    const icons = container.querySelectorAll('svg');
    expect(icons.length).toBe(6);
  });

  it('applies custom className', () => {
    render(<InlineProgressIndicator steps={mockSteps} className="custom-class" />);

    expect(screen.getByTestId('inline-progress-indicator')).toHaveClass('custom-class');
  });
});

describe('SourceChainIndicator', () => {
  it('shows source chain with arrows', () => {
    render(<SourceChainIndicator steps={mockSteps} />);

    expect(screen.getByTestId('source-chain-indicator')).toBeInTheDocument();
    expect(screen.getByText(/TMDb/)).toBeInTheDocument();
    expect(screen.getByText(/豆瓣/)).toBeInTheDocument();
    expect(screen.getByText(/Wikipedia/)).toBeInTheDocument();
    expect(screen.getByText(/AI/)).toBeInTheDocument();
  });

  it('shows success checkmark for successful sources', () => {
    const successSteps: ParseStep[] = [
      ...mockSteps.slice(0, 1),
      { name: 'tmdb_search', label: '搜尋 TMDb', status: 'success' },
      ...mockSteps.slice(2),
    ];

    render(<SourceChainIndicator steps={successSteps} />);

    expect(screen.getByText('TMDb ✓')).toBeInTheDocument();
  });

  it('shows failure mark for failed sources', () => {
    render(<SourceChainIndicator steps={failedSteps} />);

    expect(screen.getByText('TMDb ✗')).toBeInTheDocument();
    expect(screen.getByText('豆瓣 ✗')).toBeInTheDocument();
  });

  it('only shows search-related steps', () => {
    render(<SourceChainIndicator steps={mockSteps} />);

    // Should not show filename_extract or download_poster
    const container = screen.getByTestId('source-chain-indicator');
    expect(container).not.toHaveTextContent('解析檔名');
    expect(container).not.toHaveTextContent('下載海報');
  });
});
