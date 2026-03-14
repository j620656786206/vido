import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { EmptyLibrary } from './EmptyLibrary';

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

describe('EmptyLibrary', () => {
  it('renders empty state heading', () => {
    render(<EmptyLibrary />);
    expect(screen.getByText('你的媒體庫還是空的')).toBeInTheDocument();
  });

  it('renders guidance text', () => {
    render(<EmptyLibrary />);
    expect(screen.getByText(/透過 qBittorrent/)).toBeInTheDocument();
  });

  it('renders connect qBittorrent button linking to settings', () => {
    render(<EmptyLibrary />);
    const link = screen.getByTestId('connect-qbittorrent-btn');
    expect(link).toHaveAttribute('href', '/settings/qbittorrent');
    expect(link).toHaveTextContent('連接 qBittorrent');
  });

  it('renders learn more button linking to search', () => {
    render(<EmptyLibrary />);
    const link = screen.getByTestId('learn-more-btn');
    expect(link).toHaveAttribute('href', '/search');
    expect(link).toHaveTextContent('了解更多');
  });
});
