import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { GenerationWorkspaceV2, type GenerationWorkspaceV2Props } from './GenerationWorkspaceV2';
import type { GenerationBatchProgressState } from '../../hooks/useGenerationBatchProgress';
import type { GenerationBatchItem } from '../../services/subtitleService';
import type { FeedRow } from '../../hooks/useGenerationJobsFeed';

const progress = (
  over: Partial<GenerationBatchProgressState> = {}
): GenerationBatchProgressState => ({
  batchId: 'b1',
  totalItems: 38,
  currentIndex: 12,
  currentMediaId: 'm12',
  currentItem: '奧本海默',
  successCount: 12,
  failCount: 0,
  pausedCount: 0,
  status: 'running',
  spentUsd: 0.42,
  budgetUsd: 5,
  ...over,
});

const items: GenerationBatchItem[] = [
  { mediaId: 'm0', title: '沙丘：第二部' },
  { mediaId: 'm12', title: '奧本海默' },
  { mediaId: 'm20', title: '花月殺手' },
];

const feed: FeedRow[] = [
  { seq: 1, tone: 'done', stage: '完成', mediaId: 'm0', message: '沙丘：第二部' },
  { seq: 2, tone: 'active', stage: '轉錄中', mediaId: 'm12', trail: '45%' },
];

function props(over: Partial<GenerationWorkspaceV2Props> = {}): GenerationWorkspaceV2Props {
  return {
    mode: 'running',
    progress: progress(),
    items,
    feed,
    onLaunch: vi.fn(),
    onConfirmCancelAll: vi.fn(),
    onResume: vi.fn(),
    onRetryData: vi.fn(),
    ...over,
  };
}

describe('GenerationWorkspaceV2 (ux3-ai-2 — state matrix)', () => {
  it('idle: 目前沒有進行中的生成 + Mono preview count + launch opens the dialog', async () => {
    const onLaunch = vi.fn();
    render(<GenerationWorkspaceV2 {...props({ mode: 'idle', previewCount: 38, onLaunch })} />);
    expect(screen.getByText('目前沒有進行中的生成')).toBeInTheDocument();
    expect(screen.getByText('38')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('workspace-launch'));
    expect(onLaunch).toHaveBeenCalled();
    // idle shows no feed pane
    expect(screen.queryByTestId('workspace-event-log')).not.toBeInTheDocument();
  });

  it('running: renders queue rows via deriveRowStates + 全部取消 (batch-wide) + overall progressbar', async () => {
    const onConfirmCancelAll = vi.fn();
    render(<GenerationWorkspaceV2 {...props({ onConfirmCancelAll })} />);
    expect(screen.getByTestId('workspace-queue-row-m0')).toHaveAttribute('data-state', 'done');
    expect(screen.getByTestId('workspace-queue-row-m12')).toHaveAttribute('data-state', 'active');
    expect(screen.getByTestId('workspace-queue-row-m20')).toHaveAttribute('data-state', 'queued');
    // overall bar exposes progressbar a11y
    const bar = screen.getByRole('progressbar', { name: '整批生成進度' });
    expect(bar).toHaveAttribute('aria-valuenow', String(Math.round((12 / 38) * 100)));
    await userEvent.click(screen.getByTestId('workspace-cancel-all'));
    expect(onConfirmCancelAll).toHaveBeenCalled();
    // NO per-row action controls (capability honor)
    expect(screen.queryByRole('button', { name: /暫停|重試/ })).not.toBeInTheDocument();
  });

  it('budget_ceiling: F9-verbatim banner + 下次繼續 resumes + paused rows', async () => {
    const onResume = vi.fn();
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'budget_ceiling',
          progress: progress({
            status: 'budget_ceiling',
            successCount: 12,
            pausedCount: 26,
            totalItems: 38,
          }),
          onResume,
        })}
      />
    );
    const banner = screen.getByTestId('workspace-budget-banner');
    expect(banner).toHaveTextContent('已達本次預算上限');
    expect(banner).toHaveTextContent('部下次繼續');
    // success semantics, not error tokens
    expect(banner.className).toContain('warning-tint');
    await userEvent.click(screen.getByTestId('workspace-resume'));
    expect(onResume).toHaveBeenCalled();
  });

  it('attach: degraded — in-flight card + honest "自本頁開啟起顯示" note, no fake full queue', () => {
    render(<GenerationWorkspaceV2 {...props({ mode: 'attach', items: [] })} />);
    expect(screen.getByTestId('workspace-attach')).toBeInTheDocument();
    expect(screen.getByText(/佇列明細自本頁開啟起顯示/)).toBeInTheDocument();
    expect(screen.getByText('奧本海默')).toBeInTheDocument(); // in-flight item from progress
    expect(screen.queryByTestId('workspace-queue-row-m0')).not.toBeInTheDocument();
  });

  it('single: opportunistic single-job rows under 進行中任務', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'single',
          items: [],
          singleJobs: {
            s1: { mediaId: 's1', phase: 'transcribing', message: '媽的多重宇宙', percentage: null },
          },
        })}
      />
    );
    expect(screen.getByText('進行中任務')).toBeInTheDocument();
    expect(screen.getByTestId('workspace-queue-row-s1')).toHaveAttribute('data-state', 'active');
  });

  it('fail-soft: data error shows inline 重試 without hard-failing the page', async () => {
    const onRetryData = vi.fn();
    render(<GenerationWorkspaceV2 {...props({ dataError: true, onRetryData })} />);
    expect(screen.getByTestId('workspace-data-error')).toHaveTextContent('無法載入生成狀態');
    await userEvent.click(screen.getByTestId('workspace-data-retry'));
    expect(onRetryData).toHaveBeenCalled();
    // page shell still renders (queue still there)
    expect(screen.getByTestId('generation-workspace')).toBeInTheDocument();
  });

  it('event log: renders feed rows, aria-live, and the honest "僅狀態事件，不含逐字內容" footer', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const log = screen.getByTestId('workspace-event-log');
    expect(log).toHaveTextContent('即時活動');
    expect(log).toHaveTextContent('自開啟本頁起累積');
    expect(log).toHaveTextContent('僅狀態事件，不含逐字內容');
    expect(screen.getAllByTestId('workspace-feed-row')).toHaveLength(2);
    expect(screen.getByText('45%').className).toContain('font-mono'); // Mono trail
    // the log list announces politely
    expect(screen.getByRole('list', { name: '生成事件日誌' })).toHaveAttribute(
      'aria-live',
      'polite'
    );
  });
});
