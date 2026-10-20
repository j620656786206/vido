import { describe, it, expect, vi } from 'vitest';
import { useEffect } from 'react';
import { act, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { GenerationWorkspaceV2, type GenerationWorkspaceV2Props } from './GenerationWorkspaceV2';
import type { GenerationBatchProgressState } from '../../hooks/useGenerationBatchProgress';
import type { GenerationBatchItemState } from '../../services/subtitleService';
import { useGenerationJobsFeed, type FeedRow } from '../../hooks/useGenerationJobsFeed';

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

  it('credit_exhausted: F9b-verbatim banner + billing link + 沒有額度 pill + 下次繼續', async () => {
    const onResume = vi.fn();
    render(
      <GenerationWorkspaceV2
        {...props({
          mode: 'credit_exhausted',
          progress: progress({
            status: 'credit_exhausted',
            successCount: 3,
            pausedCount: 2,
            totalItems: 5,
            creditProvider: 'anthropic',
            billingUrl: 'https://console.anthropic.com/settings/billing',
            items: [item('m0', '沙丘：第二部', 'done'), item('m12', '奧本海默', 'paused')],
          }),
          onResume,
        })}
      />
    );
    const banner = screen.getByTestId('workspace-credit-banner');
    expect(banner).toHaveTextContent(
      'Anthropic 帳號沒有額度了 — 已完成3部，剩餘2部已暫停，儲值後按「下次繼續」'
    );
    expect(banner.className).toContain('warning-tint');
    expect(screen.queryByTestId('workspace-budget-banner')).not.toBeInTheDocument();
    expect(screen.getByTestId('credit-stop-billing-link')).toHaveAttribute(
      'href',
      'https://console.anthropic.com/settings/billing'
    );
    expect(screen.getByTestId('workspace-status-pill')).toHaveTextContent('沒有額度');
    expect(screen.getByTestId('workspace-queue-row-m12')).toHaveAttribute('data-state', 'paused');
    expect(screen.getByText('已停止（AI 帳號沒有額度）')).toBeInTheDocument();
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

    // disc-2026-10-no-text-source-skip-b AC #4/#10 — overturns dsr-6d-b's
    // 「不准用已略過」: a skip is a neutral 已略過 badge (the 已取消 tint, no
    // glyph) with the reason on the line under the title. data-state stays
    // `failed` (the E2E reads it); data-skipped is the new hook.
    const skipped = screen.getByTestId('workspace-queue-row-m4');
    expect(skipped).toHaveAttribute('data-state', 'failed');
    expect(skipped).toHaveAttribute('data-skipped', 'true');
    const badge = within(skipped).getByText('已略過');
    expect(badge.className).toContain('bg-[var(--bg-tertiary)]');
    expect(badge.className).not.toContain('--error-tint');
    expect(badge.className).toContain('text-[var(--text-muted)]');
    expect(badge.querySelector('svg')).toBeNull();
    expect(skipped).not.toHaveTextContent('失敗');
    const sub = within(skipped).getByText('沒有可用的字幕來源');
    expect(sub.className).toContain('--text-secondary');
    // The real failure beside it is untouched: red-tinted 失敗 with its glyph.
    expect(busy).not.toHaveAttribute('data-skipped');
    const busyBadge = within(busy).getByText('失敗');
    expect(busyBadge.className).toContain('--error-tint');
    expect(busyBadge.querySelector('svg')).not.toBeNull();
  });

  it('[P0] a skipped film says the same sentence in the queue and in the live log (AC #10)', () => {
    const NO_SOURCE = '沒有可用的字幕來源，也沒有設定語音辨識';
    render(
      <GenerationWorkspaceV2
        {...props({
          progress: progress({
            items: [
              { ...item('m5', '星際效應', 'failed', 'skipped'), failureReason: 'no_text_source' },
              item('m12', '奧本海默', 'running'),
            ],
          }),
          feed: [
            ...feed,
            {
              seq: 3,
              kind: 'failed',
              mediaId: 'm5',
              title: '星際效應',
              seriesTitle: '',
              reason: 'skipped',
              failureReason: 'no_text_source',
            },
          ],
        })}
      />
    );
    const row = screen.getByTestId('workspace-queue-row-m5');
    expect(within(row).getByText('已略過')).toBeInTheDocument();
    expect(within(row).getByText(NO_SOURCE).className).toContain('--text-secondary');
    const logRow = within(screen.getByTestId('workspace-event-log'))
      .getAllByTestId('workspace-feed-row')
      .find((r) => r.textContent!.includes('星際效應'))!;
    expect(logRow).toHaveTextContent('已略過');
    expect(logRow).not.toHaveTextContent('失敗');
    expect(within(logRow).getByTestId('workspace-feed-row-detail').textContent).toBe(NO_SOURCE);
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

  it('[P0] a failed / skipped row wraps its reason whole; other rows still truncate (disc-2026-10-gen-queue-reason-wrap-desktop)', () => {
    // At 1024 wide the live log leaves the queue's text column 242px, and the
    // 19-character skip reason needs 266px — it used to end in 「…」.
    // F8-SPEC-ROW / F11-SPEC-LOG: a reason is always whole and wraps.
    render(
      <GenerationWorkspaceV2
        {...props({
          progress: progress({
            currentMediaId: 'm12',
            items: [
              item('m0', '沙丘：第二部', 'done'),
              { ...item('m4', '星際效應', 'failed', 'skipped'), failureReason: 'no_text_source' },
              {
                ...item('m5', '奧本海默', 'failed', 'error'),
                failureReason: 'target_not_writable',
              },
              item('m12', '花月殺手', 'running'),
              item('m20', '沙丘', 'queued'),
            ],
          }),
        })}
      />
    );
    const sub = (id: string, text: string) =>
      within(screen.getByTestId(`workspace-queue-row-${id}`)).getByText(text);

    for (const reason of [
      sub('m4', '沒有可用的字幕來源，也沒有設定語音辨識'),
      sub('m5', '影片所在的資料夾寫不進字幕'),
    ]) {
      expect(reason.className).not.toContain('truncate');
      expect(reason.className).toContain('whitespace-normal');
      expect(reason.className).not.toContain('line-clamp');
      expect(reason.className).toContain('--text-secondary');
    }
    // Everything else keeps one line on desktop: a long 翻譯中… line would
    // otherwise make the row (and the queue) jump as it changes.
    for (const line of [
      sub('m0', '字幕已寫入檔案'),
      sub('m20', '等待前面項目完成'),
      within(screen.getByTestId('workspace-queue-row-m12')).getByText(/…$/),
    ]) {
      expect(line.className).toContain('truncate');
      expect(line.className).toContain('max-sm:whitespace-normal');
    }
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
    // disc-2026-10-live-activity-failure-reasons-b: hook → page, end to end. The
    // backend still sends the raw `error` (the detail dialog shows it); the
    // hook keeps only the reason code, so nothing raw can reach this log.
    const raw =
      'place: placer: write failed: open /mnt/media/Movies/Dune/Dune.zh-TW.srt: permission denied';
    const sources: FakeEventSource[] = [];
    class FakeEventSource {
      readyState = 0;
      onopen: (() => void) | null = null;
      onerror: (() => void) | null = null;
      private listeners: Record<string, ((e: MessageEvent) => void)[]> = {};
      constructor() {
        sources.push(this);
      }
      addEventListener(type: string, cb: (e: MessageEvent) => void) {
        (this.listeners[type] ||= []).push(cb);
      }
      close() {
        this.readyState = 2;
      }
      emit(type: string, data: unknown) {
        const frame = { data: JSON.stringify({ id: 'x', type, data }) } as MessageEvent;
        (this.listeners[type] || []).forEach((cb) => cb(frame));
      }
    }
    vi.stubGlobal('EventSource', FakeEventSource);
    function Harness() {
      const { feed: live, startTracking } = useGenerationJobsFeed();
      useEffect(() => startTracking(), [startTracking]);
      return <GenerationWorkspaceV2 {...props({ feed: live })} />;
    }
    try {
      const { container } = render(<Harness />);
      act(() => {
        const es = sources[0];
        es.onopen?.();
        es.emit('transcription_failed', {
          media_id: 'm1',
          phase: 'failed',
          title: '沙丘',
          error: raw,
          message: raw,
          failure_reason: 'target_not_writable',
        });
        es.emit('transcription_failed', {
          media_id: 'm2',
          phase: 'failed',
          title: '芭比',
          error: 'select audio track: no audio track found in media file',
          failure_reason: 'no_audio_track',
        });
        // An older server: a raw error and no code at all.
        es.emit('transcription_failed', {
          media_id: 'm4',
          phase: 'failed',
          title: '墨西哥',
          error: 'save SRT: open /mnt/media/Movies/M/M.srt: permission denied',
        });
        es.emit('generation_batch_progress', {
          batch_id: 'b1',
          status: 'running',
          changed_item: {
            media_id: 'm3',
            title: '花月殺手',
            media_type: 'movie',
            series_title: '',
            status: 'failed',
            reason: 'busy_elsewhere',
            failure_reason: '',
          },
        });
      });
      const [a, b, d, c] = rows();
      expect(a).toHaveTextContent('失敗');
      expect(a).toHaveTextContent('影片所在的資料夾寫不進字幕');
      expect(b).toHaveTextContent('找不到可用的音軌');
      expect(d).toHaveTextContent('生成失敗');
      expect(c).toHaveTextContent('這部正在別處處理');
      expect(container.innerHTML).not.toContain('/mnt/media');
      expect(container.innerHTML).not.toContain('permission denied');
      expect(container.innerHTML).not.toContain('no audio track');
    } finally {
      vi.unstubAllGlobals();
    }
  });

  it('[P0] the live log and the activity record word a failure the same way (目標資料夾寫不進、中途停止)', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          feed: [
            {
              seq: 1,
              kind: 'failed',
              mediaId: 'm1',
              title: '和平使者 S01E01',
              seriesTitle: '',
              reason: 'error',
              failureReason: 'target_not_writable',
            },
            {
              seq: 2,
              kind: 'failed',
              mediaId: 'm2',
              title: '芭比',
              seriesTitle: '',
              reason: null,
              failureReason: 'cancelled',
            },
          ],
        })}
      />
    );
    const [a, b] = rows();
    expect(a).toHaveTextContent('失敗');
    expect(a).toHaveTextContent('和平使者 S01E01');
    expect(a).toHaveTextContent('影片所在的資料夾寫不進字幕');
    expect(a.innerHTML).toContain('--error-text');
    expect(b).toHaveTextContent('中途停止');
    expect(b).not.toHaveTextContent('已停止');
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
              failureReason: 'budget_exceeded',
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

  it('[P2] guard: a row sub-status may wrap on a phone; the title wraps to two lines at most', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const row = screen.getByTestId('workspace-queue-row-m0');
    // disc-2026-10-generation-queue-failure-reason AC #6: line-clamp-2 replaced
    // the one-line truncate (F11-M-v2 Ah8fN).
    expect(row.querySelector('.font-semibold.text-base')!.className).toBe(
      'line-clamp-2 break-words text-base font-semibold text-[var(--text-primary)]'
    );
    expect(within(row).getByText('字幕已寫入檔案').className).toContain('max-sm:whitespace-normal');
  });
});

// disc-2026-10-generation-queue-failure-reason — the queue on the left and the
// live log on the right word one failure the same way, and the reason no longer
// squeezes the film name (F11-D-v2 l8FsB f14sNh / evt-fail, F11-SPEC-LOG tT3JX).
describe('GenerationWorkspaceV2 — one failure, one sentence, both sides', () => {
  const PEACE = '和平使者 S01E01';
  const NOT_WRITABLE = '影片所在的資料夾寫不進字幕';
  const failedItem = (failureReason: string): GenerationBatchItemState => ({
    ...item('m7', PEACE, 'failed', 'error'),
    failureReason,
  });
  const failedRow: FeedRow = {
    seq: 3,
    kind: 'failed',
    mediaId: 'm7',
    title: PEACE,
    seriesTitle: '',
    reason: 'error',
    failureReason: 'target_not_writable',
  };
  const batchDone: FeedRow = {
    seq: 4,
    kind: 'batch',
    batchId: 'b1',
    status: 'complete',
    successCount: 3,
    failCount: 2,
    budgetUsd: 5,
    creditProvider: '',
  };
  const renderFailure = () =>
    render(
      <GenerationWorkspaceV2
        {...props({
          progress: progress({
            items: [failedItem('target_not_writable'), item('m12', '奧本海默', 'running')],
          }),
          feed: [...feed, failedRow, batchDone],
        })}
      />
    );
  const feedRows = () =>
    within(screen.getByTestId('workspace-event-log')).getAllByTestId('workspace-feed-row');

  it('[P0] (a) a failed queue row says the reason in secondary text, badge 失敗, two-line title, no stepper', () => {
    renderFailure();
    const row = screen.getByTestId('workspace-queue-row-m7');
    expect(row).toHaveAttribute('data-state', 'failed');
    expect(row).toHaveTextContent(NOT_WRITABLE);
    expect(row).not.toHaveTextContent('生成失敗');
    // The badge still says 失敗 (the state); the line under the title is the why.
    expect(within(row).getByText('失敗')).toBeInTheDocument();
    const sub = within(row).getByText(NOT_WRITABLE);
    expect(sub.className).toContain('--text-secondary');
    expect(sub.className).not.toContain('--error-text');
    const title = within(row).getByText(PEACE);
    expect(title).toHaveClass('line-clamp-2');
    expect(title).not.toHaveClass('truncate');
    // Only the running row draws a stepper.
    expect(within(row).queryByTestId('generation-progress-v2')).toBeNull();
  });

  it('[P0] (b) the queue row and the live-log row for the same film say the same sentence', () => {
    renderFailure();
    const queueText = screen.getByTestId('workspace-queue-row-m7').textContent!;
    const logRow = feedRows().find((r) => r.textContent!.includes(PEACE))!;
    expect(logRow).toBeDefined();
    expect(queueText).toContain(NOT_WRITABLE);
    expect(within(logRow).getByTestId('workspace-feed-row-detail').textContent).toBe(NOT_WRITABLE);
  });

  it('[P0] (c) the reason is the second line — whole, no ·, outside the name line; the name wraps to two lines', () => {
    renderFailure();
    const logRow = feedRows().find((r) => r.textContent!.includes(PEACE))!;
    const detail = within(logRow).getByTestId('workspace-feed-row-detail');
    expect(detail.textContent).toBe(NOT_WRITABLE);
    expect(detail.className).toContain('--text-secondary');
    expect(detail.className).not.toMatch(/\b(truncate|line-clamp-\d)\b/);
    const name = within(logRow).getByText(PEACE);
    expect(name).toHaveClass('line-clamp-2');
    expect(name).not.toHaveClass('truncate');
    // The name's line does not contain the reason.
    const nameLine = name.closest('div')!;
    expect(nameLine.contains(detail)).toBe(false);
    expect(nameLine).not.toHaveTextContent(NOT_WRITABLE);
    expect(nameLine).toHaveTextContent('失敗');
    // ⑦ 批次完成 stays one line: no second-line element.
    const batchRow = feedRows().find((r) => r.textContent!.includes('批次完成'))!;
    expect(batchRow).toHaveTextContent('完成 3 部、失敗 2 部');
    expect(within(batchRow).queryByTestId('workspace-feed-row-detail')).toBeNull();
    // Every film name in the log may wrap (two lines), never a one-line truncate.
    for (const r of feedRows()) expect(r.querySelector('.truncate')).toBeNull();
  });

  it('[P1] the glyph sits in a one-line-tall box at the top of the row, not centred on two lines', () => {
    renderFailure();
    const logRow = feedRows().find((r) => r.textContent!.includes(PEACE))!;
    expect(logRow).toHaveClass('items-start');
    const box = logRow.querySelector('svg')!.parentElement!;
    expect(box).toHaveClass('h-[1lh]', 'items-center', 'text-sm');
  });

  it('[P1] a failure with no code (older server) still says 生成失敗 on both sides', () => {
    render(
      <GenerationWorkspaceV2
        {...props({
          progress: progress({ items: [item('m7', PEACE, 'failed', 'error')] }),
          feed: [{ ...failedRow, failureReason: null }],
        })}
      />
    );
    expect(screen.getByTestId('workspace-queue-row-m7')).toHaveTextContent('生成失敗');
    expect(screen.getByTestId('workspace-feed-row-detail').textContent).toBe('生成失敗');
  });
});

// disc-2026-10-gen-live-column-squeeze (F11-SPEC-WIDTH V2oywc): at 1024 a running
// row's stepper used to hold the queue at 650 and squeeze 即時活動 to 54px. The
// right pane now never gives way; the queue does, and the running row's stepper
// reads the ROW's width (616) to go vertical. jsdom has no layout — the widths
// themselves are measured in a real browser (story AC #3).
describe('GenerationWorkspaceV2 — 即時活動 keeps 400 on a narrow desktop', () => {
  it('[P0] the live pane never shrinks; the queue column may (min-w-0)', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const pane = screen.getByTestId('workspace-event-log');
    expect(pane.className.split(/\s+/)).toEqual(
      expect.arrayContaining(['lg:w-[400px]', 'lg:shrink-0'])
    );
    // The queue column is the pane's sibling inside the body row.
    let queueCol = screen.getByTestId('workspace-queue-row-m12') as HTMLElement;
    while (queueCol.parentElement && queueCol.parentElement !== pane.parentElement) {
      queueCol = queueCol.parentElement;
    }
    expect(queueCol.parentElement).toBe(pane.parentElement);
    expect(queueCol.className.split(/\s+/)).toEqual(expect.arrayContaining(['flex-1', 'min-w-0']));
  });

  it('[P0] the running row is a size container and its stepper reads the ROW, not the viewport', () => {
    render(<GenerationWorkspaceV2 {...props()} />);
    const row = screen.getByTestId('workspace-queue-row-m12');
    expect(row.className.split(/\s+/)).toContain('@container');
    const stepper = within(row).getByTestId('generation-progress-v2');
    const indent = stepper.parentElement!;
    expect(indent.className.split(/\s+/)).toContain('@min-[616px]:pl-[54px]');
    expect(indent.className).not.toContain('sm:pl-[54px]');
    const ol = within(row).getByRole('list', { name: '字幕生成進度' });
    expect(ol.className).toContain('@min-[616px]:flex-row');
    expect(ol.className).not.toMatch(/(^|\s)sm:flex-row/);
    expect(within(row).getByTestId('gen-stage-提取音訊').className).toContain('@max-[616px]:p-1');
  });
});
