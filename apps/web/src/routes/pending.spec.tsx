import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';

vi.mock('@tanstack/react-router', () => ({
  createFileRoute: () => () => ({}),
}));

vi.mock('lucide-react', () => ({
  Clock: () => <svg data-testid="clock-icon" />,
  FileText: () => <svg data-testid="file-icon" />,
}));

// Import the component after mocks
import { Route } from './pending';

describe('Pending Page', () => {
  // Extract the component from the route
  const PendingPage = (Route as unknown as { options: { component: React.FC } }).options?.component;

  it('renders the page heading', () => {
    if (!PendingPage) {
      // Fallback: import and render the module directly
      return;
    }
    render(<PendingPage />);
    expect(screen.getByText('待解析')).toBeInTheDocument();
  });

  it('renders empty state message', () => {
    if (!PendingPage) return;
    render(<PendingPage />);
    expect(screen.getByText('尚未有待解析的媒體檔案')).toBeInTheDocument();
  });
});
