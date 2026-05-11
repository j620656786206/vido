import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { EmptyNoFolder } from './EmptyNoFolder';

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

describe('EmptyNoFolder (bugfix-10-5 Case B: qBT OK, no media folder)', () => {
  it('renders the Case B heading exactly as specified in AC #3', () => {
    render(<EmptyNoFolder />);
    expect(screen.getByText('指定一個媒體資料夾即可開始')).toBeInTheDocument();
  });

  it('renders the Case B subtitle exactly as specified', () => {
    render(<EmptyNoFolder />);
    expect(screen.getByText('Vido 會掃描資料夾中的影片並自動匹配 TMDb 資訊')).toBeInTheDocument();
  });

  it('renders primary CTA linking to /settings/libraries (AC #3)', () => {
    render(<EmptyNoFolder />);
    const link = screen.getByTestId('empty-no-folder-libraries-btn');
    expect(link).toHaveAttribute('href', '/settings/libraries');
    expect(link).toHaveTextContent('設定媒體資料夾');
  });

  it('renders secondary CTA linking to /setup (AC #3)', () => {
    render(<EmptyNoFolder />);
    const link = screen.getByTestId('empty-no-folder-wizard-btn');
    expect(link).toHaveAttribute('href', '/setup');
    expect(link).toHaveTextContent('開啟設定精靈');
  });

  it('renders the root container with empty-no-folder testid (AC #3)', () => {
    render(<EmptyNoFolder />);
    expect(screen.getByTestId('empty-no-folder')).toBeInTheDocument();
  });
});
