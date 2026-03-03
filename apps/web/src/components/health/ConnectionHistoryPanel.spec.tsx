import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { ConnectionHistoryPanel } from './ConnectionHistoryPanel';

vi.mock('../../hooks/useConnectionHealth', () => ({
  useConnectionHistory: vi.fn(),
}));

import { useConnectionHistory } from '../../hooks/useConnectionHealth';

const mockUseConnectionHistory = vi.mocked(useConnectionHistory);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
}

const mockEvents = [
  {
    id: 'evt-1',
    service: 'qbittorrent',
    eventType: 'disconnected' as const,
    status: 'down',
    message: 'connection refused',
    createdAt: new Date(Date.now() - 2 * 60 * 1000).toISOString(),
  },
  {
    id: 'evt-2',
    service: 'qbittorrent',
    eventType: 'connected' as const,
    status: 'healthy',
    createdAt: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
  },
  {
    id: 'evt-3',
    service: 'qbittorrent',
    eventType: 'recovered' as const,
    status: 'healthy',
    createdAt: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
  },
];

describe('ConnectionHistoryPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state', () => {
    mockUseConnectionHistory.mockReturnValue({
      data: undefined,
      isLoading: true,
    } as ReturnType<typeof useConnectionHistory>);

    renderWithQuery(
      <ConnectionHistoryPanel isOpen={true} onClose={vi.fn()} />
    );
    expect(screen.getByText('載入中...')).toBeInTheDocument();
  });

  it('shows empty state when no history', () => {
    mockUseConnectionHistory.mockReturnValue({
      data: [],
      isLoading: false,
    } as ReturnType<typeof useConnectionHistory>);

    renderWithQuery(
      <ConnectionHistoryPanel isOpen={true} onClose={vi.fn()} />
    );
    expect(screen.getByText('沒有連線記錄')).toBeInTheDocument();
  });

  it('renders history events', () => {
    mockUseConnectionHistory.mockReturnValue({
      data: mockEvents,
      isLoading: false,
    } as ReturnType<typeof useConnectionHistory>);

    renderWithQuery(
      <ConnectionHistoryPanel isOpen={true} onClose={vi.fn()} />
    );
    // Each label appears both in filter buttons and in event items
    expect(screen.getAllByText('已斷線').length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText('已連線').length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText('已恢復').length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText('connection refused')).toBeInTheDocument();
  });

  it('shows filter buttons', () => {
    mockUseConnectionHistory.mockReturnValue({
      data: mockEvents,
      isLoading: false,
    } as ReturnType<typeof useConnectionHistory>);

    renderWithQuery(
      <ConnectionHistoryPanel isOpen={true} onClose={vi.fn()} />
    );
    expect(screen.getByText('全部')).toBeInTheDocument();
    // Filter buttons for event types
    expect(screen.getAllByText('已連線').length).toBeGreaterThanOrEqual(1);
    expect(screen.getAllByText('已斷線').length).toBeGreaterThanOrEqual(1);
  });

  it('filters events by type', async () => {
    const user = userEvent.setup();
    mockUseConnectionHistory.mockReturnValue({
      data: mockEvents,
      isLoading: false,
    } as ReturnType<typeof useConnectionHistory>);

    renderWithQuery(
      <ConnectionHistoryPanel isOpen={true} onClose={vi.fn()} />
    );

    // Click the "已斷線" filter button (the one in the filter group)
    const filterButtons = screen.getAllByText('已斷線');
    // The filter button is the one with role group ancestor
    await user.click(filterButtons[0]);

    // After filtering, should only show disconnected events
    const listItems = screen.getAllByRole('listitem');
    expect(listItems).toHaveLength(1);
  });

  it('does not fetch when panel is closed', () => {
    mockUseConnectionHistory.mockReturnValue({
      data: undefined,
      isLoading: false,
    } as ReturnType<typeof useConnectionHistory>);

    renderWithQuery(
      <ConnectionHistoryPanel isOpen={false} onClose={vi.fn()} />
    );
    expect(mockUseConnectionHistory).toHaveBeenCalledWith(
      'qbittorrent',
      false
    );
  });
});
