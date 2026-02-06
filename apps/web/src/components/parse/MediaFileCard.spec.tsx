/**
 * MediaFileCard Tests (Story 3.10 - Task 8)
 */

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import {
  MediaFileCard,
  MediaFileRow,
  MediaFileGrid,
  MediaFileList,
} from './MediaFileCard';
import type { MediaFile } from './MediaFileCard';

const mockFile: MediaFile = {
  id: 'file-123',
  filename: 'Demon.Slayer.S01E01.1080p.BluRay.x264.mkv',
  path: '/media/anime/Demon.Slayer.S01E01.1080p.BluRay.x264.mkv',
  size: 1500 * 1024 * 1024, // 1.5 GB
  mediaType: 'tv',
  parseStatus: 'success',
  parsedInfo: {
    title: 'Demon Slayer',
    year: 2019,
    season: 1,
    episode: 1,
  },
};

const pendingFile: MediaFile = {
  id: 'file-456',
  filename: 'Unknown.Movie.mkv',
  path: '/media/Unknown.Movie.mkv',
  size: 800 * 1024 * 1024,
  parseStatus: 'pending',
};

const failedFile: MediaFile = {
  id: 'file-789',
  filename: 'Strange.File.mkv',
  path: '/media/Strange.File.mkv',
  size: 500 * 1024 * 1024,
  parseStatus: 'failed',
  parseSteps: [
    { name: 'tmdb_search', label: '搜尋 TMDb', status: 'failed', error: 'Not found' },
    { name: 'douban_search', label: '搜尋豆瓣', status: 'failed' },
  ],
};

describe('MediaFileCard', () => {
  it('renders the card with parsed title', () => {
    render(<MediaFileCard file={mockFile} />);

    expect(screen.getByTestId('media-file-card')).toBeInTheDocument();
    expect(screen.getByText('Demon Slayer')).toBeInTheDocument();
  });

  it('displays year from parsed info', () => {
    render(<MediaFileCard file={mockFile} />);

    expect(screen.getByText('2019')).toBeInTheDocument();
  });

  it('displays season and episode', () => {
    render(<MediaFileCard file={mockFile} />);

    expect(screen.getByText('S1')).toBeInTheDocument();
    expect(screen.getByText('E1')).toBeInTheDocument();
  });

  it('shows media type badge', () => {
    render(<MediaFileCard file={mockFile} />);

    expect(screen.getByText('影集')).toBeInTheDocument();
  });

  it('displays file size', () => {
    render(<MediaFileCard file={mockFile} />);

    expect(screen.getByText('1.46 GB')).toBeInTheDocument();
  });

  it('shows success status badge', () => {
    render(<MediaFileCard file={mockFile} />);

    expect(screen.getByTestId('parse-status-badge')).toBeInTheDocument();
    expect(screen.getByTestId('parse-status-badge')).toHaveAttribute('data-status', 'success');
  });

  it('shows pending status badge', () => {
    render(<MediaFileCard file={pendingFile} />);

    expect(screen.getByTestId('parse-status-badge')).toHaveAttribute('data-status', 'pending');
  });

  it('shows parsing badge when isParsing', () => {
    render(<MediaFileCard file={pendingFile} isParsing />);

    expect(screen.getByTestId('parsing-status-badge')).toBeInTheDocument();
  });

  it('shows error summary for failed files', () => {
    render(<MediaFileCard file={failedFile} />);

    expect(screen.getByTestId('compact-error-summary')).toBeInTheDocument();
    expect(screen.getByText(/2 個來源失敗/)).toBeInTheDocument();
  });

  it('extracts title from filename when no parsed info', () => {
    render(<MediaFileCard file={pendingFile} />);

    expect(screen.getByText('Unknown Movie')).toBeInTheDocument();
  });

  it('calls onClick when card is clicked', async () => {
    const onClick = vi.fn();
    const user = userEvent.setup();

    render(<MediaFileCard file={mockFile} onClick={onClick} />);

    await user.click(screen.getByTestId('media-file-card'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('applies custom className', () => {
    render(<MediaFileCard file={mockFile} className="custom-class" />);

    expect(screen.getByTestId('media-file-card')).toHaveClass('custom-class');
  });

  it('has correct data-status attribute', () => {
    render(<MediaFileCard file={failedFile} />);

    expect(screen.getByTestId('media-file-card')).toHaveAttribute('data-status', 'failed');
  });
});

describe('MediaFileRow', () => {
  it('renders the row with title', () => {
    render(<MediaFileRow file={mockFile} />);

    expect(screen.getByTestId('media-file-row')).toBeInTheDocument();
    expect(screen.getByText('Demon Slayer')).toBeInTheDocument();
  });

  it('shows filename', () => {
    render(<MediaFileRow file={mockFile} />);

    expect(screen.getByText(mockFile.filename)).toBeInTheDocument();
  });

  it('shows status badge', () => {
    render(<MediaFileRow file={mockFile} />);

    expect(screen.getByTestId('parse-status-badge')).toBeInTheDocument();
  });

  it('shows parsing badge when isParsing', () => {
    render(<MediaFileRow file={mockFile} isParsing />);

    expect(screen.getByTestId('parsing-status-badge')).toBeInTheDocument();
  });

  it('calls onClick when clicked', async () => {
    const onClick = vi.fn();
    const user = userEvent.setup();

    render(<MediaFileRow file={mockFile} onClick={onClick} />);

    await user.click(screen.getByTestId('media-file-row'));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});

describe('MediaFileGrid', () => {
  const files = [mockFile, pendingFile, failedFile];

  it('renders all files in grid', () => {
    render(<MediaFileGrid files={files} />);

    expect(screen.getByTestId('media-file-grid')).toBeInTheDocument();
    expect(screen.getAllByTestId('media-file-card')).toHaveLength(3);
  });

  it('passes parsingIds to cards', () => {
    render(<MediaFileGrid files={files} parsingIds={['file-456']} />);

    expect(screen.getByTestId('parsing-status-badge')).toBeInTheDocument();
  });

  it('calls onFileClick with correct file', async () => {
    const onFileClick = vi.fn();
    const user = userEvent.setup();

    render(<MediaFileGrid files={files} onFileClick={onFileClick} />);

    const cards = screen.getAllByTestId('media-file-card');
    await user.click(cards[0]);

    expect(onFileClick).toHaveBeenCalledWith(mockFile);
  });
});

describe('MediaFileList', () => {
  const files = [mockFile, pendingFile, failedFile];

  it('renders all files in list', () => {
    render(<MediaFileList files={files} />);

    expect(screen.getByTestId('media-file-list')).toBeInTheDocument();
    expect(screen.getAllByTestId('media-file-row')).toHaveLength(3);
  });

  it('passes parsingIds to rows', () => {
    render(<MediaFileList files={files} parsingIds={['file-456']} />);

    expect(screen.getByTestId('parsing-status-badge')).toBeInTheDocument();
  });

  it('calls onFileClick with correct file', async () => {
    const onFileClick = vi.fn();
    const user = userEvent.setup();

    render(<MediaFileList files={files} onFileClick={onFileClick} />);

    const rows = screen.getAllByTestId('media-file-row');
    await user.click(rows[1]);

    expect(onFileClick).toHaveBeenCalledWith(pendingFile);
  });
});
