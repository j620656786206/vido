/**
 * ErrorDetailsPanel Tests (Story 3.10 - Task 7)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ErrorDetailsPanel, CompactErrorSummary } from './ErrorDetailsPanel';
import type { ParseStep } from './types';

const failedSteps: ParseStep[] = [
  { name: 'filename_extract', label: '解析檔名', status: 'success' },
  { name: 'tmdb_search', label: '搜尋 TMDb', status: 'failed', error: 'API timeout' },
  { name: 'douban_search', label: '搜尋豆瓣', status: 'failed', error: '無法連線' },
  { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'skipped' },
  { name: 'ai_retry', label: 'AI 重試', status: 'pending' },
  { name: 'download_poster', label: '下載海報', status: 'pending' },
];

const allSuccessSteps: ParseStep[] = [
  { name: 'filename_extract', label: '解析檔名', status: 'success' },
  { name: 'tmdb_search', label: '搜尋 TMDb', status: 'success' },
  { name: 'douban_search', label: '搜尋豆瓣', status: 'skipped' },
  { name: 'wikipedia_search', label: '搜尋 Wikipedia', status: 'skipped' },
  { name: 'ai_retry', label: 'AI 重試', status: 'skipped' },
  { name: 'download_poster', label: '下載海報', status: 'success' },
];

describe('ErrorDetailsPanel', () => {
  it('renders the panel', () => {
    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" />);

    expect(screen.getByTestId('error-details-panel')).toBeInTheDocument();
  });

  it('displays failure reasons header', () => {
    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" />);

    expect(screen.getByText('失敗原因')).toBeInTheDocument();
  });

  it('lists all failed steps with errors', () => {
    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" />);

    expect(screen.getByTestId('failed-steps-list')).toBeInTheDocument();
    expect(screen.getByText(/搜尋 TMDb/)).toBeInTheDocument();
    expect(screen.getByText(/API timeout/)).toBeInTheDocument();
    expect(screen.getByText(/搜尋豆瓣/)).toBeInTheDocument();
    expect(screen.getByText(/無法連線/)).toBeInTheDocument();
  });

  it('shows source chain with success/failure indicators', () => {
    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" />);

    const sourceChain = screen.getByTestId('source-chain');
    expect(sourceChain).toBeInTheDocument();
    expect(sourceChain).toHaveTextContent('TMDb ✗');
    expect(sourceChain).toHaveTextContent('豆瓣 ✗');
  });

  it('renders action buttons', () => {
    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" />);

    expect(screen.getByTestId('manual-search-button')).toBeInTheDocument();
    expect(screen.getByTestId('edit-filename-button')).toBeInTheDocument();
    expect(screen.getByTestId('skip-button')).toBeInTheDocument();
  });

  it('displays correct button labels', () => {
    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" />);

    expect(screen.getByText('手動搜尋')).toBeInTheDocument();
    expect(screen.getByText('編輯檔名後重試')).toBeInTheDocument();
    expect(screen.getByText('跳過此檔案')).toBeInTheDocument();
  });

  it('calls onManualSearch when manual search button clicked', async () => {
    const onManualSearch = vi.fn();
    const user = userEvent.setup();

    render(
      <ErrorDetailsPanel steps={failedSteps} filename="test.mkv" onManualSearch={onManualSearch} />
    );

    await user.click(screen.getByTestId('manual-search-button'));
    expect(onManualSearch).toHaveBeenCalledTimes(1);
  });

  it('calls onEditFilename when edit button clicked', async () => {
    const onEditFilename = vi.fn();
    const user = userEvent.setup();

    render(
      <ErrorDetailsPanel steps={failedSteps} filename="test.mkv" onEditFilename={onEditFilename} />
    );

    await user.click(screen.getByTestId('edit-filename-button'));
    expect(onEditFilename).toHaveBeenCalledTimes(1);
  });

  it('calls onSkip when skip button clicked', async () => {
    const onSkip = vi.fn();
    const user = userEvent.setup();

    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" onSkip={onSkip} />);

    await user.click(screen.getByTestId('skip-button'));
    expect(onSkip).toHaveBeenCalledTimes(1);
  });

  it('does not show failure reasons section when no failures', () => {
    render(<ErrorDetailsPanel steps={allSuccessSteps} filename="test.mkv" />);

    expect(screen.queryByText('失敗原因')).not.toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<ErrorDetailsPanel steps={failedSteps} filename="test.mkv" className="custom-class" />);

    expect(screen.getByTestId('error-details-panel')).toHaveClass('custom-class');
  });

  it('shows "無回應" for failed steps without error message', () => {
    const stepsWithoutError: ParseStep[] = [
      { name: 'tmdb_search', label: '搜尋 TMDb', status: 'failed' },
    ];

    render(<ErrorDetailsPanel steps={stepsWithoutError} filename="test.mkv" />);

    expect(screen.getByText(/無回應/)).toBeInTheDocument();
  });
});

describe('CompactErrorSummary', () => {
  it('shows count of failed sources', () => {
    render(<CompactErrorSummary steps={failedSteps} />);

    expect(screen.getByTestId('compact-error-summary')).toBeInTheDocument();
    expect(screen.getByText(/2 個來源失敗/)).toBeInTheDocument();
  });

  it('shows first error message', () => {
    render(<CompactErrorSummary steps={failedSteps} />);

    expect(screen.getByText(/API timeout/)).toBeInTheDocument();
  });

  it('returns null when no failures', () => {
    const { container } = render(<CompactErrorSummary steps={allSuccessSteps} />);

    expect(container).toBeEmptyDOMElement();
  });

  it('applies custom className', () => {
    render(<CompactErrorSummary steps={failedSteps} className="custom-class" />);

    expect(screen.getByTestId('compact-error-summary')).toHaveClass('custom-class');
  });
});
