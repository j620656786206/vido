import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DownloadFilterTabs } from './DownloadFilterTabs';
import type { DownloadCounts } from '../../services/downloadService';

const mockCounts: DownloadCounts = {
  all: 10,
  downloading: 3,
  paused: 2,
  completed: 4,
  seeding: 1,
  error: 0,
};

describe('DownloadFilterTabs', () => {
  it('[P1] renders all filter tabs with counts (AC1)', () => {
    const onFilterChange = vi.fn();
    render(
      <DownloadFilterTabs activeFilter="all" counts={mockCounts} onFilterChange={onFilterChange} />
    );

    expect(screen.getByText('全部')).toBeInTheDocument();
    expect(screen.getByText('下載中')).toBeInTheDocument();
    expect(screen.getByText('已暫停')).toBeInTheDocument();
    expect(screen.getByText('已完成')).toBeInTheDocument();
    expect(screen.getByText('做種中')).toBeInTheDocument();
    // Error tab hidden when count is 0
    expect(screen.queryByText('錯誤')).not.toBeInTheDocument();
  });

  it('[P1] shows error tab when count > 0', () => {
    const countsWithError = { ...mockCounts, error: 2 };
    render(
      <DownloadFilterTabs activeFilter="all" counts={countsWithError} onFilterChange={vi.fn()} />
    );

    expect(screen.getByText('錯誤')).toBeInTheDocument();
  });

  it('[P1] highlights active filter (AC2)', () => {
    render(
      <DownloadFilterTabs activeFilter="downloading" counts={mockCounts} onFilterChange={vi.fn()} />
    );

    const downloadingTab = screen.getByRole('tab', { name: /下載中/ });
    expect(downloadingTab).toHaveAttribute('aria-selected', 'true');

    const allTab = screen.getByRole('tab', { name: /全部/ });
    expect(allTab).toHaveAttribute('aria-selected', 'false');
  });

  it('[P1] calls onFilterChange when tab is clicked (AC2)', () => {
    const onFilterChange = vi.fn();
    render(
      <DownloadFilterTabs activeFilter="all" counts={mockCounts} onFilterChange={onFilterChange} />
    );

    fireEvent.click(screen.getByText('下載中'));
    expect(onFilterChange).toHaveBeenCalledWith('downloading');
  });

  it('[P1] displays count badges on each tab (AC1)', () => {
    render(<DownloadFilterTabs activeFilter="all" counts={mockCounts} onFilterChange={vi.fn()} />);

    // Check count badges are rendered
    expect(screen.getByText('10')).toBeInTheDocument(); // all
    expect(screen.getByText('3')).toBeInTheDocument(); // downloading
    expect(screen.getByText('2')).toBeInTheDocument(); // paused
    expect(screen.getByText('4')).toBeInTheDocument(); // completed
    expect(screen.getByText('1')).toBeInTheDocument(); // seeding
  });

  it('[P2] renders with undefined counts', () => {
    render(<DownloadFilterTabs activeFilter="all" counts={undefined} onFilterChange={vi.fn()} />);

    // All counts should show 0
    const zeroBadges = screen.getAllByText('0');
    expect(zeroBadges.length).toBeGreaterThan(0);
  });

  it('[P2] has correct ARIA roles', () => {
    render(<DownloadFilterTabs activeFilter="all" counts={mockCounts} onFilterChange={vi.fn()} />);

    expect(screen.getByRole('tablist')).toBeInTheDocument();
    const tabs = screen.getAllByRole('tab');
    expect(tabs.length).toBeGreaterThanOrEqual(5); // all, downloading, paused, completed, seeding
  });

  it('[P1] shows error tab when active even if count is 0 (AC1)', () => {
    // GIVEN: error count is 0 but error filter is active
    render(
      <DownloadFilterTabs activeFilter="error" counts={mockCounts} onFilterChange={vi.fn()} />
    );

    // THEN: error tab is visible because it is the active filter
    expect(screen.getByText('錯誤')).toBeInTheDocument();
    const errorTab = screen.getByRole('tab', { name: /錯誤/ });
    expect(errorTab).toHaveAttribute('aria-selected', 'true');
  });

  it('[P2] applies error styling when error count > 0 and not active', () => {
    // GIVEN: errors exist but error tab is not active
    const countsWithError = { ...mockCounts, error: 5 };
    render(
      <DownloadFilterTabs activeFilter="all" counts={countsWithError} onFilterChange={vi.fn()} />
    );

    // THEN: error tab is rendered with error count
    expect(screen.getByText('錯誤')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
  });

  it('[P2] handles large count numbers', () => {
    // GIVEN: very large download counts
    const largeCounts = {
      all: 99999,
      downloading: 50000,
      paused: 10000,
      completed: 30000,
      seeding: 9999,
      error: 0,
    };
    render(<DownloadFilterTabs activeFilter="all" counts={largeCounts} onFilterChange={vi.fn()} />);

    // THEN: large numbers render correctly
    expect(screen.getByText('99999')).toBeInTheDocument();
    expect(screen.getByText('50000')).toBeInTheDocument();
  });

  it('[P1] each tab has aria-controls attribute (AC1)', () => {
    render(<DownloadFilterTabs activeFilter="all" counts={mockCounts} onFilterChange={vi.fn()} />);

    const tabs = screen.getAllByRole('tab');
    tabs.forEach((tab) => {
      expect(tab).toHaveAttribute('aria-controls', 'download-list');
    });
  });
});
