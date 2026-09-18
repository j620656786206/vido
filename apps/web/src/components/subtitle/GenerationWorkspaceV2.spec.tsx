import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { GenerationWorkspaceV2, type GenerationWorkspaceV2Props } from './GenerationWorkspaceV2';
import type { GenerationBatchProgressState } from '../../hooks/useGenerationBatchProgress';
import type { GenerationBatchItemState } from '../../services/subtitleService';
import type { FeedRow } from '../../hooks/useGenerationJobsFeed';

const item = (
  mediaId: string,
  title: string,
  status: GenerationBatchItemState['status'],
  reason: GenerationBatchItemState['reason'] = '',
  seriesTitle = ''
): GenerationBatchItemState => ({
  mediaId,
  title,
  mediaType: seriesTitle ? 'episode' : 'movie',
  seriesTitle,
  status,
  reason,
});

const QUEUE: GenerationBatchItemState[] = [
  item('m0', '沙丘：第二部', 'done'),
  item('m12', '奧本海默', 'running'),
  item('m20', '花月殺手', 'queued'),
];

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
  items: QUEUE,
  ...over,
});

const feed: FeedRow[] = [
  { seq: 1, tone: 'done', stage: '完成', mediaId: 'm0', message: '沙丘：第二部' },
  { seq: 2, tone: 'active', stage: '轉錄中', mediaId: 'm12', trail: '45%' },
];

function props(over: Partial<GenerationWorkspaceV2Props> = {}): GenerationWorkspaceV2Props {
  return {
    mode: 'running',
    progress: progress(),
    feed,
    onLaunch: vi.fn(),
    onConfirmCancelAll: vi.fn().mockResolvedValue(undefined),
    onResume: vi.fn(),
    onRetryData: vi.fn(),
    ...over,
  };
}

describe('GenerationWorkspaceV2 (ux3-ai-2 — state matrix)', () => {
  it('idle: 生成工作區 title + 目前沒有進行中的生成 + Mono preview count + launch opens the dialog', async () => {
    const onLaunch = vi.fn();
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'idle',
          previewCount: 38,
          onLaunch,
          progress: progress({ items: null }),
        })}
      />
    );
    // F13 `G0fna`: the idle page is the WORKSPACE, not a batch.
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent('生成工作區');
    expect(screen.getByText('目前沒有進行中的生成')).toBeInTheDocument();
    expect(screen.getByText('38')).toBeInTheDocument();
    await userEvent.click(screen.getByTestId('workspace-launch'));
    expect(onLaunch).toHaveBeenCalled();
    expect(screen.queryByTestId('workspace-event-log')).not.toBeInTheDocument();
  });

  it('running: rows come from progress.items[] + 全部取消 (batch-wide) + overall progressbar', async () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    expect(screen.getByTestId('workspace-queue-row-m0')).toHaveAttribute('data-state', 'done');
    expect(screen.getByTestId('workspace-queue-row-m12')).toHaveAttribute('data-state', 'running');
    expect(screen.getByTestId('workspace-queue-row-m20')).toHaveAttribute('data-state', 'queued');
    const bar = screen.getByRole('progressbar', { name: '整批生成進度' });
    expect(bar).toHaveAttribute('aria-valuenow', String(Math.round((12 / 38) * 100)));
    // NO per-row action controls (capability honor — the backend has no per-item
    // route). Scoped to the QUEUE so the fail-soft 重試 button cannot mask it.
    const queue = screen.getByTestId('workspace-queue-row-m12').closest('ul')!;
    expect(queue.querySelectorAll('button')).toHaveLength(0);
    await userEvent.click(screen.getByTestId('workspace-cancel-all'));
    expect(screen.getByTestId('workspace-cancel-confirm')).toBeInTheDocument();
  });

  it('budget_ceiling: F9-verbatim banner + 下次繼續 resumes + paused rows come from the backend', async () => {
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
            items: [item('m0', '沙丘：第二部', 'done'), item('m12', '奧本海默', 'paused')],
          }),
          onResume,
        })}
      />
    );
    const banner = screen.getByTestId('workspace-budget-banner');
    expect(banner).toHaveTextContent('已達本次預算上限');
    expect(banner).toHaveTextContent('部下次繼續');
    expect(banner.className).toContain('warning-tint');
    expect(screen.getByTestId('workspace-queue-row-m12')).toHaveAttribute('data-state', 'paused');
    await userEvent.click(screen.getByTestId('workspace-resume'));
    expect(onResume).toHaveBeenCalled();
  });

  it('attach: degraded — in-flight card + honest note, no fake full queue', () => {
    render(
      <GenerationWorkspaceV2 {...props({ mode: 'attach', progress: progress({ items: null }) })} />
    );
    expect(screen.getByTestId('workspace-attach')).toBeInTheDocument();
    expect(screen.getByText(/佇列明細自本頁開啟起顯示/)).toBeInTheDocument();
    expect(screen.getByText('奧本海默')).toBeInTheDocument();
    expect(screen.queryByTestId('workspace-queue-row-m0')).not.toBeInTheDocument();
  });

  it('single: opportunistic single-job rows under 進行中任務', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'single',
          progress: progress({ items: null }),
          singleJobs: {
            s1: { mediaId: 's1', phase: 'transcribing', message: '媽的多重宇宙', percentage: null },
          },
        })}
      />
    );
    expect(screen.getByText('進行中任務')).toBeInTheDocument();
    expect(screen.getByTestId('workspace-queue-row-s1')).toHaveAttribute('data-state', 'running');
  });

  it('single: a job with no message shows 處理中的項目 — NEVER a raw UUID', () => {
    const uuid = '9ff0c000-dead-4bee-8f00-000000000999';
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'single',
          progress: progress({ items: null }),
          singleJobs: {
            [uuid]: { mediaId: uuid, phase: 'transcribing', message: '', percentage: null },
          },
        })}
      />
    );
    const row = screen.getByTestId(`workspace-queue-row-${uuid}`);
    expect(row).toHaveTextContent('處理中的項目');
    expect(row).not.toHaveTextContent(uuid);
  });

  it('fail-soft: data error shows inline 重試 without hard-failing the page', async () => {
    const onRetryData = vi.fn();
    render(<GenerationWorkspaceV2 {...props({ dataError: true, onRetryData })} />);
    expect(screen.getByTestId('workspace-data-error')).toHaveTextContent('無法載入生成狀態');
    await userEvent.click(screen.getByTestId('workspace-data-retry'));
    expect(onRetryData).toHaveBeenCalled();
    expect(screen.getByTestId('generation-workspace')).toBeInTheDocument();
  });

  it('event log: renders feed rows, aria-live, and the honest footer', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const log = screen.getByTestId('workspace-event-log');
    expect(log).toHaveTextContent('即時活動');
    expect(log).toHaveTextContent('自開啟本頁起累積');
    expect(log).toHaveTextContent('僅狀態事件，不含逐字內容');
    expect(screen.getAllByTestId('workspace-feed-row')).toHaveLength(2);
    expect(screen.getByText('45%').className).toContain('font-mono');
    expect(screen.getByRole('list', { name: '生成事件日誌' })).toHaveAttribute(
      'aria-live',
      'polite'
    );
  });
});

// ---------------------------------------------------------------------------
// dsr-6d-c-1 — the rows say what the BACKEND says
// ---------------------------------------------------------------------------

describe('GenerationWorkspaceV2 — items-first rows (dsr-6d-c-1 AC #2)', () => {
  it('[P0] a refused item renders 失敗 WITH its reason — it used to render 完成', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          progress: progress({
            currentMediaId: 'm20',
            successCount: 1,
            failCount: 2,
            items: [
              item('m0', '沙丘：第二部', 'done'),
              item('m12', '奧本海默', 'failed', 'busy_elsewhere'),
              item('m4', '星際效應', 'failed', 'skipped'),
              item('m20', '花月殺手', 'running'),
            ],
          }),
        })}
      />
    );
    const busy = screen.getByTestId('workspace-queue-row-m12');
    expect(busy).toHaveAttribute('data-state', 'failed');
    expect(busy).toHaveTextContent('這部正在別處處理');
    expect(busy).not.toHaveTextContent('完成');

    const skipped = screen.getByTestId('workspace-queue-row-m4');
    expect(skipped).toHaveTextContent('沒有可用的字幕來源');
    expect(skipped).not.toHaveTextContent('已略過');
  });

  it('[P0] a failed row draws NO stepper (it is not being worked on)', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          progress: progress({
            currentMediaId: 'm12',
            items: [
              item('m0', '沙丘：第二部', 'failed', 'error'),
              item('m12', '奧本海默', 'running'),
            ],
          }),
        })}
      />
    );
    expect(screen.getByTestId('workspace-queue-row-m0')).toHaveTextContent('生成失敗');
    expect(screen.getAllByTestId('generation-progress-v2')).toHaveLength(1);
  });

  it('[P0] an episode row shows 劇名 + 集數標題; a movie shows no "undefined"', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({
            status: 'complete',
            successCount: 2,
            failCount: 0,
            currentMediaId: null,
            items: [
              item('e1', 'S04E07 第七章', 'done', '', '怪奇物語'),
              item('m0', '沙丘：第二部', 'done'),
            ],
          }),
        })}
      />
    );
    expect(screen.getByTestId('workspace-queue-row-e1')).toHaveTextContent(
      '怪奇物語 S04E07 第七章'
    );
    expect(screen.getByTestId('workspace-queue-row-m0')).not.toHaveTextContent('undefined');
  });

  it('[P0] the row badge carries a pill shape and a state tint (Component/GenQueueRow-v2)', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const badge = screen.getByTestId('workspace-queue-row-m0').querySelector('span.rounded-full');
    expect(badge).not.toBeNull();
    expect(badge!.className).toContain('bg-[var(--success-tint)]');
    expect(badge!.textContent).toContain('完成');
  });
});

describe('GenerationWorkspaceV2 — terminal honesty (dsr-6d-c-1 AC #4/#5)', () => {
  it('[P0] complete WITH failures is neutral — 全部完成 would claim an outcome that is not good', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete', successCount: 2, failCount: 3 }),
        })}
      />
    );
    const label = screen.getByTestId('workspace-terminal-label');
    expect(label).toHaveTextContent('完成 2 部、失敗 3 部');
    expect(label.className).not.toContain('success');
  });

  it('[P0] complete with zero failures earns the green 全部完成', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete', successCount: 5, failCount: 0 }),
        })}
      />
    );
    const label = screen.getByTestId('workspace-terminal-label');
    expect(label).toHaveTextContent('全部完成（5 部）');
    expect(label.className).toContain('success-text');
  });

  it('[P0] the money in the ceiling banner is NEUTRAL (Money-Is-A-Fact)', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'budget_ceiling',
          progress: progress({ status: 'budget_ceiling', pausedCount: 26 }),
        })}
      />
    );
    const money = screen
      .getByTestId('workspace-budget-banner')
      .querySelector('span.font-mono') as HTMLElement;
    expect(money.textContent).toBe('$5.00');
    expect(money.className).toContain('text-[var(--text-primary)]');
    expect(money.className).not.toContain('warning');
  });

  it('[P0] the SSE chip is GONE once the batch is no longer streaming', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    expect(screen.getAllByTestId('workspace-sse-chip').length).toBeGreaterThan(0);

    for (const mode of ['complete', 'cancelled', 'error', 'budget_ceiling'] as const) {
      const view = render(
        <GenerationWorkspaceV2 {...props({ mode, progress: progress({ status: mode }) })} />
      );
      // The overall strip's chip must be gone; only the event-log pane's remains.
      expect(view.container.querySelectorAll('[data-testid="workspace-sse-chip"]')).toHaveLength(1);
      view.unmount();
    }
  });

  it('[P0] the status pill claims running only while running, and 已達上限 at the ceiling', () => {
    const { rerender } = render(<GenerationWorkspaceV2 {...props()} />);
    expect(screen.getByTestId('workspace-status-pill')).toHaveTextContent('進行中');

    rerender(
      <GenerationWorkspaceV2
        {...props({ mode: 'budget_ceiling', progress: progress({ status: 'budget_ceiling' }) })}
      />
    );
    expect(screen.getByTestId('workspace-status-pill')).toHaveTextContent('已達上限');

    rerender(
      <GenerationWorkspaceV2
        {...props({ mode: 'complete', progress: progress({ status: 'complete' }) })}
      />
    );
    expect(screen.queryByTestId('workspace-status-pill')).toBeNull();
  });
});

describe('GenerationWorkspaceV2 — the way out of a finished batch (dsr-6d-c-1 AC #3)', () => {
  it('[P0] 下次繼續 is NOT locked behind a known queue — a cold attach at the ceiling still gets it', async () => {
    const onResume = vi.fn();
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'budget_ceiling',
          progress: progress({ status: 'budget_ceiling', pausedCount: 26, items: null }),
          onResume,
        })}
      />
    );
    await userEvent.click(screen.getByTestId('workspace-resume'));
    expect(onResume).toHaveBeenCalled();
  });

  it('[P0] 關閉 dismisses the remembered result', async () => {
    const onDismiss = vi.fn().mockResolvedValue(undefined);
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete', successCount: 5 }),
          onDismiss,
        })}
      />
    );
    await userEvent.click(screen.getByTestId('workspace-close'));
    expect(onDismiss).toHaveBeenCalledOnce();
  });

  it('[P0] a failed 關閉 says so instead of pretending', async () => {
    const onDismiss = vi.fn().mockRejectedValue(new Error('still running'));
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete', successCount: 5 }),
          onDismiss,
        })}
      />
    );
    await userEvent.click(screen.getByTestId('workspace-close'));
    const alert = await screen.findByTestId('workspace-dismiss-error');
    expect(alert).toHaveAttribute('role', 'alert');
    expect(alert).toHaveTextContent('批次還在進行中，現在無法關閉。');
  });

  it('[P0] CR M2 the verdict appears even when the queue is unknown (a cold attach)', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete', successCount: 2, failCount: 1, items: null }),
        })}
      />
    );
    expect(screen.getByTestId('workspace-terminal-label')).toHaveTextContent(
      '完成 2 部、失敗 1 部'
    );
  });

  it('[P0] CR M8 a running batch can be cancelled even before its queue is known', () => {
    render(<GenerationWorkspaceV2 {...props({ progress: progress({ items: null }) })} />);
    expect(screen.getByTestId('workspace-cancel-all')).toBeInTheDocument();
  });

  it('[P0] CR M7 single mode draws NO batch strip — there is no batch to count', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'single',
          progress: progress({ status: 'idle', totalItems: 0, items: null }),
          singleJobs: {
            s1: { mediaId: 's1', phase: 'transcribing', message: '媽的多重宇宙', percentage: null },
          },
        })}
      />
    );
    expect(screen.queryByTestId('workspace-overall')).toBeNull();
    expect(screen.queryByRole('progressbar', { name: '整批生成進度' })).toBeNull();
  });

  it('[P0] CR M5 focus follows the cancel confirm row instead of falling to <body>', async () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    await userEvent.click(screen.getByTestId('workspace-cancel-all'));
    await waitFor(() => expect(screen.getByText('繼續生成')).toHaveFocus());
    await userEvent.click(screen.getByText('繼續生成'));
    await waitFor(() => expect(screen.getByTestId('workspace-cancel-all')).toHaveFocus());
  });

  it('[P0] CR M4 the verdict is NOT a third live region', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete', successCount: 5 }),
        })}
      />
    );
    expect(screen.getByTestId('workspace-terminal-label')).not.toHaveAttribute('aria-live');
  });

  it('[P1] the footer bar does not exist while the batch is still running', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    expect(screen.queryByTestId('workspace-footer')).toBeNull();
    expect(screen.queryByTestId('workspace-close')).toBeNull();
  });
});

describe('GenerationWorkspaceV2 — cancel can fail (dsr-6d-c-1 AC #4)', () => {
  it('[P0] cancelling asks first — one click never cancels a batch', async () => {
    const onConfirmCancelAll = vi.fn().mockResolvedValue(undefined);
    render(<GenerationWorkspaceV2 {...props({ onConfirmCancelAll })} />);

    await userEvent.click(screen.getByTestId('workspace-cancel-all'));
    expect(onConfirmCancelAll).not.toHaveBeenCalled();

    await userEvent.click(screen.getByTestId('workspace-cancel-confirm-btn'));
    expect(onConfirmCancelAll).toHaveBeenCalledOnce();
  });

  it('[P0] a failed cancel SAYS SO and keeps the confirm row (the batch is still running)', async () => {
    const onConfirmCancelAll = vi.fn().mockRejectedValue(new Error('network'));
    render(<GenerationWorkspaceV2 {...props({ onConfirmCancelAll })} />);

    await userEvent.click(screen.getByTestId('workspace-cancel-all'));
    await userEvent.click(screen.getByTestId('workspace-cancel-confirm-btn'));

    const alert = await screen.findByTestId('workspace-cancel-error');
    expect(alert).toHaveAttribute('role', 'alert');
    expect(alert).toHaveTextContent('取消失敗，批次仍在進行。請再試一次。');
    expect(screen.getByTestId('workspace-cancel-confirm')).toBeInTheDocument();
  });

  it('[P0] the in-flight confirm button is aria-disabled, never disabled', async () => {
    let release: () => void = () => {};
    const onConfirmCancelAll = vi.fn(
      () =>
        new Promise<void>((res) => {
          release = res;
        })
    );
    render(<GenerationWorkspaceV2 {...props({ onConfirmCancelAll })} />);

    await userEvent.click(screen.getByTestId('workspace-cancel-all'));
    await userEvent.click(screen.getByTestId('workspace-cancel-confirm-btn'));

    const btn = await screen.findByText('取消中…');
    expect(btn).toHaveAttribute('aria-disabled', 'true');
    expect(btn).not.toHaveAttribute('disabled');

    await userEvent.click(btn);
    expect(onConfirmCancelAll).toHaveBeenCalledTimes(1);

    release();
    await waitFor(() => expect(onConfirmCancelAll).toHaveBeenCalledTimes(1));
  });
});
