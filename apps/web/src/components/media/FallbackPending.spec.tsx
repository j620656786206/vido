import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { FallbackPending } from './FallbackPending';

describe('FallbackPending', () => {
  const filename = '[Leopard-Raws] Kimi no Na wa (BD).mkv';

  it('renders spinner animation', () => {
    render(<FallbackPending filename={filename} />);
    expect(screen.getByTestId('pending-spinner')).toBeInTheDocument();
  });

  it('displays primary message in zh-TW', () => {
    render(<FallbackPending filename={filename} />);
    expect(screen.getByText('正在搜尋電影資訊⋯')).toBeInTheDocument();
  });

  it('displays secondary description', () => {
    render(<FallbackPending filename={filename} />);
    expect(screen.getByText('系統正在比對檔案名稱與 TMDb 資料庫')).toBeInTheDocument();
  });

  it('renders progress bar', () => {
    render(<FallbackPending filename={filename} />);
    expect(screen.getByTestId('pending-progress')).toBeInTheDocument();
  });

  it('shows filename hint', () => {
    render(<FallbackPending filename={filename} />);
    const hint = screen.getByTestId('pending-filename');
    expect(hint).toHaveTextContent(filename);
    expect(hint).toHaveAttribute('title', filename);
  });

  it('has the correct container testid', () => {
    render(<FallbackPending filename={filename} />);
    expect(screen.getByTestId('fallback-pending')).toBeInTheDocument();
  });
});
