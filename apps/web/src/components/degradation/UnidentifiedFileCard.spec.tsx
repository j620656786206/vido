import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { UnidentifiedFileCard } from './UnidentifiedFileCard';

describe('UnidentifiedFileCard', () => {
  const defaultProps = {
    filename: '[SubGroup] Unknown Movie.mkv',
    onManualSearch: vi.fn(),
    onEditFilename: vi.fn(),
    onSkip: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders filename', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    expect(screen.getByText('[SubGroup] Unknown Movie.mkv')).toBeInTheDocument();
  });

  it('renders "無法自動識別" message', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    expect(screen.getByText('無法自動識別')).toBeInTheDocument();
  });

  it('renders all three action buttons', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    expect(screen.getByRole('button', { name: /手動搜尋/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /編輯檔名/ })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /稍後處理/ })).toBeInTheDocument();
  });

  it('calls onManualSearch when manual search button is clicked', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    fireEvent.click(screen.getByRole('button', { name: /手動搜尋/ }));
    expect(defaultProps.onManualSearch).toHaveBeenCalledTimes(1);
  });

  it('calls onEditFilename when edit filename button is clicked', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    fireEvent.click(screen.getByRole('button', { name: /編輯檔名/ }));
    expect(defaultProps.onEditFilename).toHaveBeenCalledTimes(1);
  });

  it('calls onSkip when skip button is clicked', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    fireEvent.click(screen.getByRole('button', { name: /稍後處理/ }));
    expect(defaultProps.onSkip).toHaveBeenCalledTimes(1);
  });

  it('renders attempted sources when provided', () => {
    render(<UnidentifiedFileCard {...defaultProps} attemptedSources={['tmdb', 'douban', 'ai']} />);
    expect(screen.getByText('已嘗試：')).toBeInTheDocument();
    expect(screen.getByText('TMDb')).toBeInTheDocument();
    expect(screen.getByText('Douban')).toBeInTheDocument();
    expect(screen.getByText('AI')).toBeInTheDocument();
  });

  it('does not render attempted sources when not provided', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    expect(screen.queryByText('已嘗試：')).not.toBeInTheDocument();
  });

  it('has accessible label', () => {
    render(<UnidentifiedFileCard {...defaultProps} />);
    expect(screen.getByRole('article', { name: /無法識別的檔案/ })).toBeInTheDocument();
  });

  it('applies custom className', () => {
    render(<UnidentifiedFileCard {...defaultProps} className="custom-class" />);
    expect(screen.getByRole('article')).toHaveClass('custom-class');
  });

  it('shows filename in title attribute for long filenames', () => {
    const longFilename =
      '[Very Long Subgroup Name] Some Very Long Movie Title 2024 1080p BluRay x264.mkv';
    render(<UnidentifiedFileCard {...defaultProps} filename={longFilename} />);
    expect(screen.getByText(longFilename)).toHaveAttribute('title', longFilename);
  });
});
