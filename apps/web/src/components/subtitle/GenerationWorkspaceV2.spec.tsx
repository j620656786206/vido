import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
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
  { seq: 1, kind: 'done', mediaId: 'm0', title: '沙丘：第二部', seriesTitle: '' },
  {
    seq: 2,
    kind: 'stage',
    mediaId: 'm12',
    title: '奧本海默',
    seriesTitle: '',
    stage: 'translating',
    state: 'live',
    pipeline: false,
    percentage: 45,
  },
];

function props(over: Partial<GenerationWorkspaceV2Props> = {}): GenerationWorkspaceV2Props {
  return {
    mode: 'running',
    progress: progress(),
    feed,
    feedConnected: true,
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
          // Nothing logged this visit (dsr-6d-c-2 CR M5 keeps a non-empty log on idle).
          feed: [],
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
    // Scoped: the live log names the same film now (dsr-6d-c-2).
    expect(
      within(screen.getByTestId('workspace-attach')).getByText('奧本海默')
    ).toBeInTheDocument();
    expect(screen.queryByTestId('workspace-queue-row-m0')).not.toBeInTheDocument();
  });

  it('single: opportunistic single-job rows under 進行中任務', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'single',
          progress: progress({ items: null }),
          singleJobs: {
            s1: {
              mediaId: 's1',
              phase: 'transcribing',
              title: '媽的多重宇宙',
              message: '正在轉錄音訊',
              percentage: null,
            },
          },
        })}
      />
    );
    expect(screen.getByText('進行中任務')).toBeInTheDocument();
    expect(screen.getByTestId('workspace-queue-row-s1')).toHaveAttribute('data-state', 'running');
  });

  it('single: a job with no title shows 處理中的項目 — NEVER a raw UUID', () => {
    const uuid = '9ff0c000-dead-4bee-8f00-000000000999';
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'single',
          progress: progress({ items: null }),
          singleJobs: {
            [uuid]: {
              mediaId: uuid,
              phase: 'transcribing',
              title: '',
              message: '',
              percentage: null,
            },
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

  it('event log: renders feed rows and the honest footer; the LIST itself is not a live region', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const log = screen.getByTestId('workspace-event-log');
    expect(log).toHaveTextContent('即時活動');
    expect(log).toHaveTextContent('自開啟本頁起累積');
    expect(log).toHaveTextContent('僅狀態事件，不含逐字內容');
    expect(screen.getAllByTestId('workspace-feed-row')).toHaveLength(2);
    expect(within(log).getByText('45%').className).toContain('font-mono');
    // dsr-6d-c-2 🔴 #8: every percentage used to be read aloud.
    expect(screen.getByRole('list', { name: '生成事件日誌' })).not.toHaveAttribute('aria-live');
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
      // The overall strip's chip must be gone; only the event-log pane's remains
      // (its stream is still open — dsr-6d-c-2 judges it by `feedConnected`).
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
            s1: {
              mediaId: 's1',
              phase: 'transcribing',
              title: '媽的多重宇宙',
              message: '正在轉錄音訊',
              percentage: null,
            },
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

describe('GenerationWorkspaceV2 — the live log tells the truth (dsr-6d-c-2)', () => {
  const log = () => screen.getByTestId('workspace-event-log');
  const rows = () => within(log()).getAllByTestId('workspace-feed-row');

  it('[P0] the pane header carries the activity glyph; every row has a glyph and the · separator', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    expect(within(log()).getByTestId('workspace-log-header-icon')).toBeInTheDocument();
    for (const row of rows()) {
      expect(row.querySelector('svg')).not.toBeNull();
      expect(row).toHaveTextContent('·');
    }
  });

  it('[P0] each row names the film, and a stage row keeps its percentage only while live', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          feed: [
            {
              seq: 1,
              kind: 'stage',
              mediaId: 'e7',
              title: 'S04E07 第七章',
              seriesTitle: '怪奇物語',
              stage: 'transcribing',
              state: 'passed',
              pipeline: false,
              percentage: null,
            },
            {
              seq: 2,
              kind: 'stage',
              mediaId: 'e7',
              title: 'S04E07 第七章',
              seriesTitle: '怪奇物語',
              stage: 'translating',
              state: 'live',
              pipeline: false,
              percentage: 45,
            },
          ],
        })}
      />
    );
    const [past, live] = rows();
    expect(past).toHaveTextContent('轉錄中');
    expect(past).toHaveTextContent('怪奇物語 S04E07 第七章');
    expect(past.querySelector('.animate-spin')).toBeNull();
    expect(live).toHaveTextContent('翻譯中');
    expect(live).toHaveTextContent('45%');
    expect(live.querySelector('.animate-spin')).not.toBeNull();
  });

  it('[P0] an unknown title says 處理中的項目 — never a UUID', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          feed: [{ seq: 1, kind: 'done', mediaId: '3d87dcb5-uuid', title: '', seriesTitle: '' }],
        })}
      />
    );
    expect(rows()[0]).toHaveTextContent('處理中的項目');
    expect(log()).not.toHaveTextContent('3d87dcb5');
  });

  it('[P0] 🔴 #13 a failure speaks Chinese — the raw backend string (and its server path) never reaches the page', () => {
    const raw = 'save SRT: open /mnt/media/Movies/Dune.en.srt: permission denied';
    const { container } = render(
      <GenerationWorkspaceV2
        {...props({
          feed: [
            {
              seq: 1,
              kind: 'failed',
              mediaId: 'm1',
              title: '沙丘',
              seriesTitle: '',
              reason: null,
              error: raw,
            },
            {
              seq: 2,
              kind: 'failed',
              mediaId: 'm2',
              title: '芭比',
              seriesTitle: '',
              reason: null,
              error: 'select audio track: no audio track found in media file',
            },
            {
              seq: 3,
              kind: 'failed',
              mediaId: 'm3',
              title: '花月殺手',
              seriesTitle: '',
              reason: 'busy_elsewhere',
              error: null,
            },
          ],
        })}
      />
    );
    const [a, b, c] = rows();
    expect(a).toHaveTextContent('失敗');
    expect(a).toHaveTextContent('生成失敗');
    expect(b).toHaveTextContent('找不到可用的音軌');
    expect(c).toHaveTextContent('這部正在別處處理');
    expect(container.innerHTML).not.toContain('/mnt/media');
    expect(container.innerHTML).not.toContain('no audio track');
  });

  it('[P0] a single job stopped by the budget is 已停止 (ochre), not 失敗', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          feed: [
            {
              seq: 1,
              kind: 'failed',
              mediaId: 'm1',
              title: '芭比',
              seriesTitle: '',
              reason: null,
              error: 'translate: AI_BUDGET_EXCEEDED: ceiling',
            },
          ],
        })}
      />
    );
    const row = rows()[0];
    expect(row).toHaveTextContent('已停止');
    expect(row).toHaveTextContent('已達預算上限');
    expect(row).not.toHaveTextContent('失敗');
    expect(row.innerHTML).toContain('--warning-text');
  });

  it('[P0] batch rows say 本批次; the budget amount is neutral text-primary', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'budget_ceiling',
          progress: progress({ status: 'budget_ceiling' }),
          feed: [
            {
              seq: 1,
              kind: 'batch',
              batchId: 'b1',
              status: 'budget_ceiling',
              successCount: 1,
              failCount: 0,
              budgetUsd: 5,
            },
          ],
        })}
      />
    );
    const row = rows()[0];
    expect(row).toHaveTextContent('已達預算上限');
    expect(row).toHaveTextContent('本批次');
    const money = within(row).getByText('$5.00');
    expect(money.className).toContain('text-[var(--text-primary)]');
  });

  it('[P0] a finished batch with failures is not green; the numbers match the verdict line', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete', successCount: 1, failCount: 2 }),
          feed: [
            {
              seq: 1,
              kind: 'batch',
              batchId: 'b1',
              status: 'complete',
              successCount: 1,
              failCount: 2,
              budgetUsd: 5,
            },
          ],
        })}
      />
    );
    const row = rows()[0];
    expect(row).toHaveTextContent('批次完成');
    expect(row).toHaveTextContent('完成 1 部、失敗 2 部');
    expect(row.innerHTML).not.toContain('--success-text');
  });

  it('[P0] 🔴 #14 the log chip shows only while the jobs stream is connected', () => {
    const { rerender } = render(<GenerationWorkspaceV2 {...props({ feedConnected: false })} />);
    expect(within(log()).queryByTestId('workspace-sse-chip')).not.toBeInTheDocument();
    rerender(<GenerationWorkspaceV2 {...props({ feedConnected: true })} />);
    expect(within(log()).getByTestId('workspace-sse-chip')).toBeInTheDocument();
  });

  it('[P1] feedConnected defaults to false — unknown is not "connected"', () => {
    const p = props();
    delete (p as Partial<GenerationWorkspaceV2Props>).feedConnected;
    render(<GenerationWorkspaceV2 {...p} />);
    expect(within(log()).queryByTestId('workspace-sse-chip')).not.toBeInTheDocument();
  });

  it('[P0] F12: the second footer line only at the budget ceiling', () => {
    const { rerender } = render(
      <GenerationWorkspaceV2
        {...props({ mode: 'budget_ceiling', progress: progress({ status: 'budget_ceiling' }) })}
      />
    );
    expect(within(log()).getByText('已停止（達預算上限）')).toBeInTheDocument();
    rerender(
      <GenerationWorkspaceV2
        {...props({ mode: 'complete', progress: progress({ status: 'complete' }) })}
      />
    );
    expect(within(log()).queryByText('已停止（達預算上限）')).not.toBeInTheDocument();
  });

  it('[P0] 🔴 #8 the screen reader hears results and batch rows only — and from ONE region that survives the mode switch', () => {
    const stage: FeedRow = {
      seq: 1,
      kind: 'stage',
      mediaId: 'm1',
      title: '奧本海默',
      seriesTitle: '',
      stage: 'translating',
      state: 'live',
      pipeline: false,
      percentage: 45,
    };
    const { rerender } = render(<GenerationWorkspaceV2 {...props({ feed: [stage] })} />);
    const region = screen.getByTestId('workspace-log-announcer');
    expect(region).toHaveAttribute('aria-live', 'polite');
    expect(region).toHaveTextContent(''); // a stage row is not announced

    const done: FeedRow = {
      seq: 2,
      kind: 'done',
      mediaId: 'm1',
      title: '奧本海默',
      seriesTitle: '',
    };
    const batch: FeedRow = {
      seq: 3,
      kind: 'batch',
      batchId: 'b1',
      status: 'complete',
      successCount: 1,
      failCount: 0,
      budgetUsd: 5,
    };
    rerender(
      <GenerationWorkspaceV2
        {...props({
          mode: 'complete',
          progress: progress({ status: 'complete' }),
          feed: [{ ...stage, state: 'passed', pipeline: false, percentage: null }, done, batch],
        })}
      />
    );
    // Same DOM node: a region mounted together with its text is never read out.
    expect(screen.getByTestId('workspace-log-announcer')).toBe(region);
    expect(region).toHaveTextContent('批次完成');
    expect(document.querySelectorAll('[aria-live]')).toHaveLength(1);
  });

  it('[P1] it follows the newest row only while the reader is already at the bottom', () => {
    const many = (n: number): FeedRow[] =>
      Array.from({ length: n }, (_, i) => ({
        seq: i + 1,
        kind: 'done' as const,
        mediaId: `m${i}`,
        title: `片${i}`,
        seriesTitle: '',
      }));
    const { rerender } = render(<GenerationWorkspaceV2 {...props({ feed: many(3) })} />);
    const list = screen.getByRole('list', { name: '生成事件日誌' });
    Object.defineProperty(list, 'scrollHeight', { configurable: true, value: 1000 });
    Object.defineProperty(list, 'clientHeight', { configurable: true, value: 300 });

    // At the bottom → a new row pulls the view down.
    list.scrollTop = 700;
    list.dispatchEvent(new Event('scroll'));
    Object.defineProperty(list, 'scrollHeight', { configurable: true, value: 1040 });
    rerender(<GenerationWorkspaceV2 {...props({ feed: many(4) })} />);
    expect(list.scrollTop).toBe(1040);

    // Scrolled up to read → left alone.
    list.scrollTop = 100;
    list.dispatchEvent(new Event('scroll'));
    Object.defineProperty(list, 'scrollHeight', { configurable: true, value: 1080 });
    rerender(<GenerationWorkspaceV2 {...props({ feed: many(5) })} />);
    expect(list.scrollTop).toBe(100);
  });

  it('CR M5: a single job that just finished keeps its log on screen when the page falls back to idle', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'idle',
          progress: progress({ status: 'idle', items: null }),
          feed: [{ seq: 1, kind: 'done', mediaId: 'm9', title: '芭比', seriesTitle: '' }],
        })}
      />
    );
    expect(screen.getByTestId('workspace-idle')).toBeInTheDocument();
    expect(within(log()).getByText('芭比')).toBeInTheDocument();
  });

  it('[P1] idle with nothing logged still has no log pane', () => {
    render(
      <GenerationWorkspaceV2
        {...props({ mode: 'idle', progress: progress({ status: 'idle', items: null }), feed: [] })}
      />
    );
    expect(screen.queryByTestId('workspace-event-log')).not.toBeInTheDocument();
  });

  it('CR M7: a second identical result is a NEW node in the live region, so it is read again', () => {
    const batch = (seq: number): FeedRow => ({
      seq,
      kind: 'batch',
      batchId: `b${seq}`,
      status: 'complete',
      successCount: 1,
      failCount: 0,
      budgetUsd: 5,
    });
    const { rerender } = render(<GenerationWorkspaceV2 {...props({ feed: [batch(1)] })} />);
    const first = screen.getByTestId('workspace-log-announcer').firstElementChild;
    expect(first).toHaveTextContent('批次完成');
    rerender(<GenerationWorkspaceV2 {...props({ feed: [batch(1), batch(2)] })} />);
    const second = screen.getByTestId('workspace-log-announcer').firstElementChild;
    expect(second).toHaveTextContent('批次完成');
    expect(second).not.toBe(first);
  });

  it('CR L10: the row state is readable, not only coloured', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          feed: [
            {
              seq: 1,
              kind: 'stage',
              mediaId: 'm1',
              title: '奧本海默',
              seriesTitle: '',
              stage: 'transcribing',
              state: 'stopped',
              percentage: null,
              pipeline: false,
            },
          ],
        })}
      />
    );
    expect(rows()[0]).toHaveTextContent('（已中斷）');
    expect(rows()[0].querySelector('.animate-spin')).toBeNull();
  });
});

// dsr-6f-4 — the phone page (F11-M-v2 `PXB0z`). jsdom evaluates no media queries,
// so these pin STATE, ARIA and the presence of the `max-sm:` tokens; what a 390px
// screen actually draws is measured in tests/e2e/generation-workspace-mobile.spec.ts.
describe('GenerationWorkspaceV2 (dsr-6f-4 — phone page)', () => {
  const many = (n: number): FeedRow[] =>
    Array.from({ length: n }, (_, i) => ({
      seq: i + 1,
      kind: 'done' as const,
      mediaId: `m${i}`,
      title: `片${i}`,
      seriesTitle: '',
    }));
  const list = () => screen.getByRole('list', { name: '生成事件日誌' });
  const toggle = () => screen.getByTestId('workspace-log-toggle');

  it('[P0] the back button exists only when there is somewhere to go back to', async () => {
    const onBack = vi.fn();
    const { rerender } = render(<GenerationWorkspaceV2 {...props({ onBack })} />);
    const back = screen.getByRole('button', { name: '返回活動' });
    expect(back).toBe(screen.getByTestId('workspace-back'));
    expect(back.className).toContain('sm:hidden');
    await userEvent.click(back);
    expect(onBack).toHaveBeenCalledTimes(1);

    rerender(<GenerationWorkspaceV2 {...props()} />);
    expect(screen.queryByTestId('workspace-back')).not.toBeInTheDocument();
  });

  it('[P0] the title row leads on a phone and the breadcrumb keeps its DOM place (order-first, not an order on the nav)', () => {
    render(<GenerationWorkspaceV2 {...props({ onBack: vi.fn() })} />);
    const h1 = screen.getByRole('heading', { level: 1 });
    expect(h1.className).toContain('max-sm:text-lg');
    const titleRow = screen.getByTestId('workspace-title-row');
    expect(titleRow.className).toContain('max-sm:order-first');
    expect(titleRow.className).toContain('max-sm:justify-between');
    const crumb = screen.getByRole('navigation', { name: '麵包屑' });
    expect(crumb.className).not.toMatch(/order-/);
    expect(crumb.className).toContain('max-sm:text-xs');
  });

  it('[P0] the overall strip re-flows in place: one strip, one progressbar, full-width bar, chip pushed right on phones only', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    expect(screen.getAllByTestId('workspace-overall')).toHaveLength(1);
    const bar = screen.getByRole('progressbar', { name: '整批生成進度' });
    expect(bar.className).toContain('max-sm:w-full');
    expect(bar.className).toContain('w-[180px]');
    const chip = within(screen.getByTestId('workspace-overall')).getByTestId('workspace-sse-chip');
    expect(chip.className).toContain('max-sm:ml-auto');
    // Unprefixed ml-auto would shove the chip to the far right on desktop.
    expect(chip.className.split(/\s+/)).not.toContain('ml-auto');
    // The log footer's chip must not inherit the strip's placement.
    const logChip = within(screen.getByTestId('workspace-event-log')).getByTestId(
      'workspace-sse-chip'
    );
    expect(logChip.className).not.toContain('max-sm:ml-auto');
  });

  it('[P0] the log starts collapsed; the toggle names the list it controls and flips aria-expanded', async () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    expect(toggle()).toHaveAttribute('aria-expanded', 'false');
    expect(toggle()).toHaveAttribute('aria-controls', list().id);
    expect(list().id).not.toBe('');
    expect(toggle().className).toContain('sm:hidden');
    // The stretched hit area hangs off the HEADER; a relative button would shrink it.
    expect(toggle().className.split(/\s+/)).not.toContain('relative');
    expect(list().className).toContain('max-sm:hidden');

    await userEvent.click(toggle());
    expect(toggle()).toHaveAttribute('aria-expanded', 'true');
    expect(list().className).not.toContain('max-sm:hidden');
    expect(list().className).toContain('max-sm:max-h-80');

    await userEvent.click(toggle());
    expect(toggle()).toHaveAttribute('aria-expanded', 'false');
    expect(list().className).toContain('max-sm:hidden');
  });

  it('[P0] collapsing never unmounts the announcer: same node, still the only live region, still updated', async () => {
    const { rerender } = render(<GenerationWorkspaceV2 {...props({ feed: [] })} />);
    const region = screen.getByTestId('workspace-log-announcer');
    // Collapsed: a result arrives and is announced anyway.
    rerender(<GenerationWorkspaceV2 {...props({ feed: many(1) })} />);
    expect(screen.getByTestId('workspace-log-announcer')).toBe(region);
    expect(region.textContent).not.toBe('');
    expect(document.querySelectorAll('[aria-live]')).toHaveLength(1);
    await userEvent.click(toggle());
    expect(screen.getByTestId('workspace-log-announcer')).toBe(region);
    // Open is where a second region would most likely sneak in (e.g. on the list).
    expect(document.querySelectorAll('[aria-live]')).toHaveLength(1);
    await userEvent.click(toggle());
    expect(screen.getByTestId('workspace-log-announcer')).toBe(region);
    expect(document.querySelectorAll('[aria-live]')).toHaveLength(1);
    // The list is hidden, not gone.
    expect(list()).toBeInTheDocument();
  });

  it('[P1] collapsed shows the hint instead of the 僅狀態事件 note; the budget stop line survives both states', async () => {
    render(
      <GenerationWorkspaceV2
        {...props({ mode: 'budget_ceiling', progress: progress({ status: 'budget_ceiling' }) })}
      />
    );
    const hint = screen.getByTestId('workspace-log-hint');
    const note = screen.getByTestId('workspace-log-note');
    expect(hint).not.toHaveAttribute('hidden');
    expect(hint).toHaveTextContent('展開查看即時事件（不含逐字內容、無時間戳）');
    expect(note.className).toContain('max-sm:hidden');
    expect(screen.getByTestId('workspace-event-log')).toHaveTextContent('已停止（達預算上限）');

    await userEvent.click(toggle());
    expect(hint).toHaveAttribute('hidden');
    expect(note.className).not.toContain('max-sm:hidden');
    expect(screen.getByTestId('workspace-event-log')).toHaveTextContent('已停止（達預算上限）');
  });

  it('[P1] collapsed with no stream: the footer row is empty, so it gives up its gap (CR M1)', async () => {
    const { rerender } = render(<GenerationWorkspaceV2 {...props({ feedConnected: false })} />);
    const row = () => screen.getByTestId('workspace-log-footer-row');
    expect(row().className).toContain('max-sm:hidden');
    await userEvent.click(toggle());
    expect(row().className).not.toContain('max-sm:hidden');
    await userEvent.click(toggle());
    expect(row().className).toContain('max-sm:hidden');
    // Connected: the chip lives in this row, collapsed or not.
    rerender(<GenerationWorkspaceV2 {...props({ feedConnected: true })} />);
    expect(row().className).not.toContain('max-sm:hidden');
  });

  it('[P1] the toggle is named by the header it sits in, and an empty list draws no box', () => {
    render(<GenerationWorkspaceV2 {...props({ feed: [] })} />);
    expect(screen.getByRole('button', { name: '即時活動' })).toBe(toggle());
    expect(list().className).toContain('max-sm:empty:hidden');
  });

  it('[P1] opening lands on the newest row — even after the reader had scrolled up before collapsing', async () => {
    render(<GenerationWorkspaceV2 {...props({ feed: many(40) })} />);
    const el = list();
    Object.defineProperty(el, 'scrollHeight', { configurable: true, value: 1000 });
    Object.defineProperty(el, 'clientHeight', { configurable: true, value: 320 });
    el.scrollTop = 0;
    await userEvent.click(toggle());
    expect(el.scrollTop).toBe(1000);

    // Reader scrolls up, collapses, re-opens.
    el.scrollTop = 100;
    el.dispatchEvent(new Event('scroll'));
    await userEvent.click(toggle());
    el.scrollTop = 0;
    await userEvent.click(toggle());
    expect(el.scrollTop).toBe(1000);
  });

  it('[P1] cancel-confirm and the footer carry the phone tokens (sentence on its own line, equal buttons)', async () => {
    const { rerender } = render(
      <GenerationWorkspaceV2
        {...props({ onConfirmCancelAll: vi.fn().mockRejectedValue(new Error('x')) })}
      />
    );
    await userEvent.click(screen.getByTestId('workspace-cancel-all'));
    const confirm = screen.getByTestId('workspace-cancel-confirm');
    expect(confirm.className).toContain('max-sm:w-full');
    expect(screen.getByText('確定要取消整個批次嗎？已完成的字幕會保留。').className).toContain(
      'max-sm:w-full'
    );
    const confirmBtn = screen.getByTestId('workspace-cancel-confirm-btn');
    expect(confirmBtn.className).toContain('max-sm:flex-1');
    expect(screen.getByRole('button', { name: '繼續生成' }).className).toContain('max-sm:flex-1');
    await userEvent.click(confirmBtn);
    expect((await screen.findByTestId('workspace-cancel-error')).className).toContain(
      'max-sm:w-full'
    );

    rerender(
      <GenerationWorkspaceV2
        {...props({
          mode: 'budget_ceiling',
          progress: progress({ status: 'budget_ceiling' }),
          onDismiss: vi.fn().mockRejectedValue(new Error('x')),
        })}
      />
    );
    expect(screen.getByTestId('workspace-close').className).toContain('max-sm:flex-1');
    expect(screen.getByTestId('workspace-resume').className).toContain('max-sm:flex-1');
    await userEvent.click(screen.getByTestId('workspace-close'));
    expect((await screen.findByTestId('workspace-dismiss-error')).className).toContain(
      'max-sm:w-full'
    );
  });

  it('[P2] guard: a row sub-status may wrap on a phone, the title span is untouched', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const row = screen.getByTestId('workspace-queue-row-m0');
    expect(row.querySelector('.font-semibold.text-base')!.className).toBe(
      'truncate text-base font-semibold text-[var(--text-primary)]'
    );
    expect(within(row).getByText('已完成，字幕已寫入檔案').className).toContain(
      'max-sm:whitespace-normal'
    );
  });
});
