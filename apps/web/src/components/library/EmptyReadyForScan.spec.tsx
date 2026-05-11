import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { EmptyReadyForScan } from './EmptyReadyForScan';
import type { useTriggerScan } from '../../hooks/useScanner';

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

const mockMutateAsync = vi.fn();
type TriggerScanResult = ReturnType<typeof useTriggerScan>;
let mockTriggerScanReturn: Partial<TriggerScanResult> = {
  mutateAsync: mockMutateAsync,
  isPending: false,
};

vi.mock('../../hooks/useScanner', () => ({
  useTriggerScan: vi.fn(() => mockTriggerScanReturn),
}));

function renderComponent() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <EmptyReadyForScan />
    </QueryClientProvider>
  );
}

describe('EmptyReadyForScan (bugfix-10-5 Case C: ready, library empty)', () => {
  beforeEach(() => {
    mockMutateAsync.mockReset();
    mockTriggerScanReturn = { mutateAsync: mockMutateAsync, isPending: false };
  });

  it('renders the Case C heading exactly as specified in AC #4', () => {
    renderComponent();
    expect(screen.getByText('準備好了，等待第一筆媒體')).toBeInTheDocument();
  });

  it('renders the Case C subtitle exactly as specified', () => {
    renderComponent();
    expect(screen.getByText('下載完成或掃描到檔案後會自動出現在這裡')).toBeInTheDocument();
  });

  it('renders the root container with empty-ready-for-scan testid (AC #4)', () => {
    renderComponent();
    expect(screen.getByTestId('empty-ready-for-scan')).toBeInTheDocument();
  });

  it('renders secondary CTA linking to /downloads (AC #4)', () => {
    renderComponent();
    const link = screen.getByTestId('empty-ready-for-scan-downloads-btn');
    expect(link).toHaveAttribute('href', '/downloads');
    expect(link).toHaveTextContent('前往下載中');
  });

  it('renders primary CTA as a button with correct label (AC #4)', () => {
    renderComponent();
    const btn = screen.getByTestId('empty-ready-for-scan-trigger-btn');
    expect(btn.tagName).toBe('BUTTON');
    expect(btn).toHaveTextContent('立即掃描');
  });

  it('calls useTriggerScan().mutateAsync on primary CTA click (AC #8 spec a)', async () => {
    mockMutateAsync.mockResolvedValueOnce({});
    renderComponent();
    const btn = screen.getByTestId('empty-ready-for-scan-trigger-btn');
    fireEvent.click(btn);
    await waitFor(() => expect(mockMutateAsync).toHaveBeenCalledTimes(1));
  });

  it('disables primary CTA when triggerScan.isPending (AC #8 spec b)', () => {
    mockTriggerScanReturn = { mutateAsync: mockMutateAsync, isPending: true };
    renderComponent();
    const btn = screen.getByTestId('empty-ready-for-scan-trigger-btn');
    expect(btn).toBeDisabled();
    expect(btn).toHaveTextContent('掃描中…');
  });

  it('shows success notification when mutation resolves (AC #8 spec c)', async () => {
    mockMutateAsync.mockResolvedValueOnce({});
    renderComponent();
    fireEvent.click(screen.getByTestId('empty-ready-for-scan-trigger-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('empty-ready-for-scan-notification')).toHaveTextContent(
        '掃描已啟動'
      );
    });
  });

  it('shows error notification with AppError.message when mutation rejects (AC #8 spec d)', async () => {
    mockMutateAsync.mockRejectedValueOnce({ code: 'SCANNER_BUSY', message: '掃描伺服器忙線中' });
    renderComponent();
    fireEvent.click(screen.getByTestId('empty-ready-for-scan-trigger-btn'));
    await waitFor(() => {
      expect(screen.getByTestId('empty-ready-for-scan-notification')).toHaveTextContent(
        '掃描伺服器忙線中'
      );
    });
  });
});
