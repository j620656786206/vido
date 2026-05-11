import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { EmptyNoQBT } from './EmptyNoQBT';

vi.mock('@tanstack/react-router', () => ({
  Link: ({
    children,
    to,
    ...props
  }: {
    children: React.ReactNode;
    to: string;
    [key: string]: unknown;
  }) => (
    <a href={to} {...props}>
      {children}
    </a>
  ),
}));

describe('EmptyNoQBT (bugfix-10-5 Case A: qBT disconnected)', () => {
  it('renders the Case A heading exactly as specified in AC #2', () => {
    render(<EmptyNoQBT />);
    expect(screen.getByText('連接 qBittorrent 開始下載')).toBeInTheDocument();
  });

  it('renders the Case A subtitle exactly as specified', () => {
    render(<EmptyNoQBT />);
    expect(screen.getByText('Vido 會自動追蹤你的下載並建立媒體庫')).toBeInTheDocument();
  });

  it('renders primary CTA linking to /settings/qbittorrent (AC #2)', () => {
    render(<EmptyNoQBT />);
    const link = screen.getByTestId('empty-no-qbt-connect-btn');
    expect(link).toHaveAttribute('href', '/settings/qbittorrent');
    expect(link).toHaveTextContent('連接 qBittorrent');
  });

  it('renders secondary CTA linking to /settings/libraries (AC #2)', () => {
    render(<EmptyNoQBT />);
    const link = screen.getByTestId('empty-no-qbt-folder-btn');
    expect(link).toHaveAttribute('href', '/settings/libraries');
    expect(link).toHaveTextContent('已有檔案？設定資料夾');
  });

  it('renders the root container with empty-no-qbt testid (AC #2)', () => {
    render(<EmptyNoQBT />);
    expect(screen.getByTestId('empty-no-qbt')).toBeInTheDocument();
  });
});
