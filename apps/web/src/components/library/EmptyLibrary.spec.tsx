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
  it('renders welcome heading', () => {
    render(<EmptyLibrary />);
    expect(screen.getByText('歡迎來到你的媒體庫')).toBeInTheDocument();
  });

  it('renders guidance text', () => {
    render(<EmptyLibrary />);
    expect(screen.getByText(/你的媒體庫目前是空的/)).toBeInTheDocument();
  });

  it('renders link to search page', () => {
    render(<EmptyLibrary />);
    const link = screen.getByRole('link', { name: '搜尋媒體' });
    expect(link).toHaveAttribute('href', '/search');
  });

  it('renders emoji with accessible label', () => {
    render(<EmptyLibrary />);
    expect(screen.getByRole('img', { name: '媒體庫' })).toBeInTheDocument();
  });
});
