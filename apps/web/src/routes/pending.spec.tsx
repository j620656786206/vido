import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => (opts: Record<string, unknown>) => opts,
}));

vi.mock('lucide-react', () => ({
  Clock: () => <svg data-testid="clock-icon" />,
  FileText: () => <svg data-testid="file-icon" />,
}));

// Import triggers createFileRoute mock, Route.component is the real component
import { Route } from './pending';

const PendingPage = (Route as { component: React.FC }).component;

describe('Pending Page', () => {
  it('[P0] renders the page heading', () => {
    render(<PendingPage />);
    expect(screen.getByText('待解析')).toBeInTheDocument();
  });

  it('[P0] renders empty state message', () => {
    render(<PendingPage />);
    expect(screen.getByText('尚未有待解析的媒體檔案')).toBeInTheDocument();
  });

  it('[P1] renders helper text', () => {
    render(<PendingPage />);
    expect(screen.getByText('當有新的媒體檔案需要解析時，它們會顯示在這裡')).toBeInTheDocument();
  });
});
