import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MetadataExport } from './MetadataExport';

vi.mock('../../hooks/useBackups', () => ({
  useExport: vi.fn(),
}));

import { useExport } from '../../hooks/useBackups';

const mockUseExport = vi.mocked(useExport);

function renderWithQuery(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(React.createElement(QueryClientProvider, { client: queryClient }, ui));
}

beforeEach(() => {
  mockUseExport.mockReturnValue({
    mutateAsync: vi.fn().mockResolvedValue({
      exportId: 'e1',
      format: 'json',
      status: 'completed',
      itemCount: 5,
      message: '匯出完成，共 5 個項目',
    }),
    isPending: false,
  } as any);
});

describe('MetadataExport', () => {
  it('renders format selector with 3 options', () => {
    renderWithQuery(React.createElement(MetadataExport));
    expect(screen.getByTestId('metadata-export')).toBeInTheDocument();
    expect(screen.getByTestId('export-format-json')).toBeInTheDocument();
    expect(screen.getByTestId('export-format-yaml')).toBeInTheDocument();
    expect(screen.getByTestId('export-format-nfo')).toBeInTheDocument();
  });

  it('renders export button', () => {
    renderWithQuery(React.createElement(MetadataExport));
    expect(screen.getByTestId('export-btn')).toBeInTheDocument();
    expect(screen.getByText('匯出')).toBeInTheDocument();
  });

  it('shows success message after export', async () => {
    const user = userEvent.setup();
    renderWithQuery(React.createElement(MetadataExport));
    await user.click(screen.getByTestId('export-btn'));
    expect(screen.getByTestId('export-message')).toBeInTheDocument();
    expect(screen.getByText(/匯出完成/)).toBeInTheDocument();
  });

  it('shows download link for JSON export', async () => {
    const user = userEvent.setup();
    renderWithQuery(React.createElement(MetadataExport));
    await user.click(screen.getByTestId('export-btn'));
    expect(screen.getByTestId('export-download-link')).toBeInTheDocument();
  });

  it('does not show download link for NFO export', async () => {
    const user = userEvent.setup();
    mockUseExport.mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue({
        exportId: 'e2',
        format: 'nfo',
        status: 'completed',
        itemCount: 3,
        message: 'NFO 匯出完成',
      }),
      isPending: false,
    } as any);

    renderWithQuery(React.createElement(MetadataExport));

    // Select NFO format
    await user.click(screen.getByTestId('export-format-nfo'));
    await user.click(screen.getByTestId('export-btn'));

    expect(screen.queryByTestId('export-download-link')).not.toBeInTheDocument();
  });

  it('shows error message on failure', async () => {
    const user = userEvent.setup();
    mockUseExport.mockReturnValue({
      mutateAsync: vi.fn().mockResolvedValue({
        status: 'failed',
        error: 'EXPORT_FAILED: no media',
      }),
      isPending: false,
    } as any);

    renderWithQuery(React.createElement(MetadataExport));
    await user.click(screen.getByTestId('export-btn'));
    expect(screen.getByTestId('export-message')).toBeInTheDocument();
    expect(screen.getByText(/匯出失敗/)).toBeInTheDocument();
  });

  it('shows format descriptions', () => {
    renderWithQuery(React.createElement(MetadataExport));
    expect(screen.getByText(/JSON 格式/)).toBeInTheDocument();
    expect(screen.getByText(/YAML 格式/)).toBeInTheDocument();
    expect(screen.getByText(/Kodi\/Plex\/Jellyfin/)).toBeInTheDocument();
  });
});
