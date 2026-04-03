import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { FallbackFailed } from './FallbackFailed';

// Mock TanStack Router — same pattern as PosterCard.spec.tsx
vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    search,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    search?: Record<string, string>;
    [key: string]: unknown;
  }) => {
    const qs = search ? '?' + new URLSearchParams(search).toString() : '';
    return (
      <a href={`${to}${qs}`} {...props}>
        {children}
      </a>
    );
  },
}));

const defaultProps = {
  title: '[Leopard-Raws] Kimi no Na wa (BD)',
  mediaType: 'movie' as const,
  filePath: '/volume1/Movies/Anime/[Leopard-Raws] Kimi no Na wa (BD).mkv',
  fileSize: 4509715660, // ~4.2 GB
  createdAt: '2026-03-28T14:32:00Z',
  parseStatus: 'failed',
  onEditClick: vi.fn(),
};

describe('FallbackFailed', () => {
  it('renders search-x icon and title message for movie', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByTestId('fallback-failed-title')).toHaveTextContent(
      '我們找不到這部電影的資料'
    );
  });

  it('renders correct title message for TV show', () => {
    render(<FallbackFailed {...defaultProps} mediaType="tv" />);
    expect(screen.getByTestId('fallback-failed-title')).toHaveTextContent(
      '我們找不到這部電視節目的資料'
    );
  });

  it('renders secondary description', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByText('你可以手動搜尋，或等待系統自動比對')).toBeInTheDocument();
  });

  // AC #4: File info section
  it('displays file name', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByTestId('file-info-name')).toHaveTextContent(
      '[Leopard-Raws] Kimi no Na wa (BD).mkv'
    );
  });

  it('displays file directory path', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByTestId('file-info-path')).toHaveTextContent('/volume1/Movies/Anime/');
  });

  it('displays file size in GB', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByTestId('file-info-size')).toHaveTextContent('4.2 GB');
  });

  it('displays added date in zh-TW locale', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByTestId('file-info-date')).toBeInTheDocument();
  });

  it('displays parse status with warning color for "failed"', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByTestId('file-info-status')).toHaveTextContent('比對失敗');
  });

  it('shows "尚未比對" for empty parseStatus', () => {
    render(<FallbackFailed {...defaultProps} parseStatus="" />);
    expect(screen.getByTestId('file-info-status')).toHaveTextContent('尚未比對');
  });

  it('displays section header "檔案資訊"', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByText('檔案資訊')).toBeInTheDocument();
  });

  // AC #5: CTA buttons
  it('renders "搜尋 Metadata" primary CTA with correct link including q= param', () => {
    render(<FallbackFailed {...defaultProps} />);
    const link = screen.getByTestId('cta-search-metadata');
    expect(link).toHaveTextContent('搜尋 Metadata');
    expect(link.getAttribute('href')).toMatch(/\/search\?q=.+/);
  });

  it('renders "手動編輯" secondary action', () => {
    render(<FallbackFailed {...defaultProps} />);
    expect(screen.getByTestId('cta-manual-edit')).toHaveTextContent('手動編輯');
  });

  it('calls onEditClick when "手動編輯" is clicked', () => {
    const onEditClick = vi.fn();
    render(<FallbackFailed {...defaultProps} onEditClick={onEditClick} />);
    fireEvent.click(screen.getByTestId('cta-manual-edit'));
    expect(onEditClick).toHaveBeenCalledOnce();
  });

  it('hides file size when zero', () => {
    render(<FallbackFailed {...defaultProps} fileSize={0} />);
    expect(screen.queryByTestId('file-info-size')).not.toBeInTheDocument();
  });
});
