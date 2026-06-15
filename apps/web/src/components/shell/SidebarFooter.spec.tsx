import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { SidebarFooter } from './SidebarFooter';
import { useStatusSummary } from '../../hooks/useStatusSummary';
import type { StatusSummary } from '../../services/statusSummaryService';

vi.mock('../../hooks/useStatusSummary', () => ({ useStatusSummary: vi.fn() }));
// Render the Base UI tooltip wrapper as a passthrough (its trigger child is the dot).
vi.mock('../ui/Tooltip', () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => children,
}));

const mockHook = vi.mocked(useStatusSummary);

function summary(over: Partial<StatusSummary> = {}): StatusSummary {
  return {
    diskHeadroom: { status: 'ok', usedBytes: 3.2e12, totalBytes: 8e12, volumes: 1 },
    activeScan: { status: 'ok', active: true, percentDone: 42 },
    downloadQueue: { status: 'ok', downloading: 5, total: 9 },
    serviceHealth: {
      status: 'ok',
      services: [
        {
          name: 'qbittorrent',
          displayName: 'qBittorrent',
          status: 'connected',
          message: '',
          lastSuccessAt: null,
          lastCheckAt: '',
          responseTimeMs: 0,
        },
      ],
    },
    ...over,
  };
}

function setData(data: StatusSummary | undefined) {
  mockHook.mockReturnValue({ data } as unknown as ReturnType<typeof useStatusSummary>);
}

describe('SidebarFooter (status strip)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('renders disk / scan / queue / health dot when all sections are ok', () => {
    setData(summary());
    render(<SidebarFooter />);
    expect(screen.getByTestId('status-disk')).toHaveTextContent('3.2 / 8.0 TB');
    expect(screen.getByTestId('status-scan')).toHaveTextContent('掃描中');
    expect(screen.getByTestId('status-queue')).toHaveTextContent('佇列 5');
    expect(screen.getByTestId('status-dot-qbittorrent')).toBeInTheDocument();
  });

  it('fails soft: disk unavailable → em-dash placeholder, no throw', () => {
    setData(
      summary({
        diskHeadroom: {
          status: 'unavailable',
          usedBytes: 0,
          totalBytes: 0,
          volumes: 0,
          error: 'x',
        },
      })
    );
    render(<SidebarFooter />);
    expect(screen.getByTestId('status-disk')).toHaveTextContent('—');
  });

  it('fails soft: service health unavailable → muted placeholder dots (no named dot)', () => {
    setData(summary({ serviceHealth: { status: 'unavailable', services: [], error: 'x' } }));
    render(<SidebarFooter />);
    expect(screen.queryByTestId('status-dot-qbittorrent')).not.toBeInTheDocument();
    expect(screen.getByTestId('sidebar-footer-status')).toBeInTheDocument();
  });

  it('hides scan + queue when inactive / empty', () => {
    setData(
      summary({
        activeScan: { status: 'ok', active: false, percentDone: 0 },
        downloadQueue: { status: 'ok', downloading: 0, total: 0 },
      })
    );
    render(<SidebarFooter />);
    expect(screen.queryByTestId('status-scan')).not.toBeInTheDocument();
    expect(screen.queryByTestId('status-queue')).not.toBeInTheDocument();
  });

  it('collapsed rail shows dots only (no disk block)', () => {
    setData(summary());
    render(<SidebarFooter collapsed />);
    expect(screen.getByTestId('sidebar-footer-status')).toBeInTheDocument();
    expect(screen.queryByTestId('status-disk')).not.toBeInTheDocument();
    expect(screen.getByTestId('status-dot-qbittorrent')).toBeInTheDocument();
  });

  it('no data (loading/error) → placeholder, never throws', () => {
    setData(undefined);
    expect(() => render(<SidebarFooter />)).not.toThrow();
    expect(screen.getByTestId('status-disk')).toHaveTextContent('—');
  });
});
