import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
  createRootRoute,
  createRoute,
  createRouter,
  createMemoryHistory,
  RouterProvider,
} from '@tanstack/react-router';

vi.mock('../../hooks/useActivity', () => ({ useActivity: vi.fn() }));

import { useActivity } from '../../hooks/useActivity';
import { ActivityHub } from './ActivityHub';
import type { ActivitySummary } from '../../services/activityService';

const mockUseActivity = vi.mocked(useActivity);

function summary(over: Partial<ActivitySummary> = {}): ActivitySummary {
  return {
    activeJobs: { status: 'ok', jobs: [] },
    pending: { status: 'ok', parseCount: 0 },
    downloads: { status: 'ok', downloading: 0, queued: 0, total: 0 },
    recent: { status: 'ok', events: [] },
    ...over,
  };
}

function result(over: Record<string, unknown> = {}) {
  return {
    data: undefined,
    isLoading: false,
    isError: false,
    refetch: vi.fn(),
    ...over,
  } as unknown as ReturnType<typeof useActivity>;
}

// ActivityHub renders Links to /library and /downloads — mount it in a router whose tree
// has those routes so href generation resolves.
function renderHub() {
  const rootRoute = createRootRoute({ component: () => React.createElement(ActivityHub) });
  const mk = (p: string) =>
    createRoute({ getParentRoute: () => rootRoute, path: p, component: () => null });
  const routeTree = rootRoute.addChildren([mk('/library'), mk('/downloads'), mk('/activity')]);
  const router = createRouter({
    routeTree,
    history: createMemoryHistory({ initialEntries: ['/'] }),
  });
  return render(React.createElement(RouterProvider, { router } as never));
}

describe('ActivityHub (v2 Activity hub — four states + fail-soft)', () => {
  beforeEach(() => vi.clearAllMocks());

  it('[P1] Loading — renders the row-shaped skeleton', async () => {
    mockUseActivity.mockReturnValue(result({ isLoading: true }));
    renderHub();
    expect(await screen.findByTestId('activity-skeleton')).toBeInTheDocument();
  });

  it('[P1] Page error — whole-request failure shows one retry banner (page never blanks)', async () => {
    const refetch = vi.fn();
    mockUseActivity.mockReturnValue(result({ isError: true, refetch }));
    renderHub();
    expect(await screen.findByTestId('activity-page-error')).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('activity-section-retry'));
    expect(refetch).toHaveBeenCalled();
  });

  it('[P1] Empty — all sections ok with no content shows the calm empty state', async () => {
    mockUseActivity.mockReturnValue(result({ data: summary() }));
    renderHub();
    expect(await screen.findByTestId('activity-empty')).toBeInTheDocument();
  });

  it('[P1] Data — active jobs map kind→title with progress; pending + downloads + recent render', async () => {
    mockUseActivity.mockReturnValue(
      result({
        data: summary({
          activeJobs: {
            status: 'ok',
            jobs: [
              { kind: 'scan', percentDone: 62, detail: '/media/movies', current: 1234 },
              { kind: 'subtitle_batch', percentDone: 40, detail: 'ep.mkv', current: 12, total: 30 },
            ],
          },
          pending: { status: 'ok', parseCount: 8 },
          downloads: { status: 'ok', downloading: 3, queued: 5, total: 8 },
          recent: {
            status: 'ok',
            events: [
              {
                kind: 'parse',
                result: 'completed',
                detail: 'done.mkv',
                at: '2026-06-15T10:00:00Z',
              },
              { kind: 'parse', result: 'failed', detail: 'bad.mkv', at: '2026-06-15T09:00:00Z' },
            ],
          },
        }),
      })
    );
    renderHub();

    expect(await screen.findByTestId('activity-job-scan')).toHaveTextContent('媒體庫掃描');
    expect(screen.getByTestId('activity-job-scan')).toHaveTextContent('62%');
    expect(screen.getByTestId('activity-job-subtitle_batch')).toHaveTextContent('批次字幕');
    expect(screen.getByTestId('activity-job-subtitle_batch')).toHaveTextContent('12 / 30');
    expect(screen.getByTestId('activity-pending-row')).toHaveTextContent('8 個項目待處理');
    expect(screen.getByTestId('activity-pending-cta')).toBeInTheDocument();
    expect(screen.getByTestId('activity-downloads-row')).toHaveTextContent('3 個進行中 · 5 個排隊');
    // Recent: completed → 解析完成, failed → 解析失敗.
    const recent = screen.getAllByTestId('activity-recent-row');
    expect(recent).toHaveLength(2);
    expect(recent[0]).toHaveTextContent('解析完成');
    expect(recent[1]).toHaveTextContent('解析失敗');
  });

  it('[P1] Per-section fail-soft — an unavailable section degrades alone; the page renders', async () => {
    mockUseActivity.mockReturnValue(
      result({
        data: summary({
          activeJobs: { status: 'unavailable', jobs: [], error: 'boom' },
          pending: { status: 'ok', parseCount: 4 },
        }),
      })
    );
    renderHub();
    // The failed section shows its inline banner...
    expect(await screen.findByTestId('activity-active-error')).toBeInTheDocument();
    // ...while a healthy section still renders, and the page is not the empty/error state.
    expect(screen.getByTestId('activity-pending-row')).toBeInTheDocument();
    expect(screen.queryByTestId('activity-page-error')).toBeNull();
    expect(screen.queryByTestId('activity-empty')).toBeNull();
  });

  it('[P2] zero-count sections are omitted (no empty pending/downloads headers)', async () => {
    mockUseActivity.mockReturnValue(
      result({
        data: summary({
          recent: {
            status: 'ok',
            events: [
              { kind: 'parse', result: 'completed', detail: 'x.mkv', at: '2026-06-15T10:00:00Z' },
            ],
          },
        }),
      })
    );
    renderHub();
    expect(await screen.findByTestId('activity-recent-row')).toBeInTheDocument();
    expect(screen.queryByTestId('activity-pending-row')).toBeNull();
    expect(screen.queryByTestId('activity-downloads-row')).toBeNull();
  });
});
