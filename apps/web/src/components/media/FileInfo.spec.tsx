import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { FileInfo, formatFileSize, parseQuality } from './FileInfo';

describe('formatFileSize', () => {
  it('formats bytes', () => {
    expect(formatFileSize(0)).toBe('0 B');
    expect(formatFileSize(500)).toBe('500 B');
  });

  it('formats kilobytes', () => {
    expect(formatFileSize(1024)).toBe('1 KB');
    expect(formatFileSize(1536)).toBe('1.5 KB');
  });

  it('formats megabytes', () => {
    expect(formatFileSize(1048576)).toBe('1 MB');
  });

  it('formats gigabytes', () => {
    expect(formatFileSize(1073741824)).toBe('1 GB');
    expect(formatFileSize(4831838208)).toBe('4.5 GB');
  });
});

describe('parseQuality', () => {
  it('detects 1080p', () => {
    expect(parseQuality('Movie.2024.1080p.BluRay.mkv')).toBe('1080P');
  });

  it('detects 4K', () => {
    expect(parseQuality('Movie.2024.4K.mkv')).toBe('4K');
  });

  it('detects 2160p as 4K', () => {
    expect(parseQuality('Movie.2024.2160p.mkv')).toBe('4K');
  });

  it('detects 720p', () => {
    expect(parseQuality('Movie.720p.mkv')).toBe('720P');
  });

  it('returns null for no quality info', () => {
    expect(parseQuality('movie.mkv')).toBeNull();
  });
});

describe('FileInfo', () => {
  it('renders filename and file size', () => {
    render(<FileInfo filePath="/media/Movie.2024.1080p.mkv" fileSize={4831838208} />);

    expect(screen.getByTestId('file-info')).toBeInTheDocument();
    expect(screen.getByTestId('file-name')).toHaveTextContent('Movie.2024.1080p.mkv');
    expect(screen.getByTestId('file-size')).toHaveTextContent('4.5 GB');
    expect(screen.getByTestId('file-quality')).toHaveTextContent('1080P');
  });

  it('truncates long filenames', () => {
    render(
      <FileInfo
        filePath="/media/This.Is.A.Very.Long.Movie.Title.2024.1080p.BluRay.x264-GROUP.mkv"
        fileSize={1000}
      />
    );

    const nameEl = screen.getByTestId('file-name');
    expect(nameEl.textContent!.length).toBeLessThanOrEqual(43); // 40 + "..."
  });

  it('shows full path in tooltip', () => {
    const fullPath = '/media/movies/Movie.mkv';
    render(<FileInfo filePath={fullPath} />);

    const nameContainer = screen.getByTestId('file-name').closest('[title]');
    expect(nameContainer).toHaveAttribute('title', fullPath);
  });

  it('returns null when no file info', () => {
    const { container } = render(<FileInfo />);
    expect(container.firstChild).toBeNull();
  });

  it('renders without file size', () => {
    render(<FileInfo filePath="/media/movie.mkv" />);
    expect(screen.getByTestId('file-name')).toBeInTheDocument();
    expect(screen.queryByTestId('file-size')).not.toBeInTheDocument();
  });
});
