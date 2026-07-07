// Implements: <utility — no .pen counterpart>
/**
 * Pure state helpers for the generation WORKSPACE (Story ux3-ai-2 AC 2/3).
 * Kept separate from the component so the render-mode decision — the crux of the
 * state matrix — is unit-tested in isolation.
 */
import type { GenerationBatchHookStatus } from '../../hooks/useGenerationBatchProgress';

/**
 * The workspace's top-level render mode.
 *  - `loading`   — the on-mount status probe is still in flight.
 *  - `idle`      — no batch running, no single jobs → calm empty + preview + launcher.
 *  - `running`   — batch running AND its `items[]` are known (started this session,
 *                  cached by the launcher) → full queue.
 *  - `attach`    — batch running but `items[]` unknown (attached cold; the status
 *                  probe carries none — disc-2026-07-generation-batch-status-items,
 *                  backlog) → degraded: counters + in-flight card only, honest note.
 *  - `single`    — no batch, but detail-triggered single jobs are in flight → queue
 *                  of single-job rows (opportunistic; AC 5).
 *  - terminals   — `budget_ceiling` (F9-verbatim) / `complete` / `cancelled` / `error`.
 */
export type WorkspaceMode =
  | 'loading'
  | 'idle'
  | 'running'
  | 'attach'
  | 'single'
  | 'budget_ceiling'
  | 'complete'
  | 'cancelled'
  | 'error';

const BATCH_TERMINALS = new Set(['budget_ceiling', 'complete', 'cancelled', 'error']);

export function deriveWorkspaceMode(input: {
  /** True while the on-mount getGenerationBatchStatus() probe has not resolved. */
  probing: boolean;
  batchStatus: GenerationBatchHookStatus;
  /** Whether the batch's enumerated `items[]` are available (start-cache hit). */
  hasItems: boolean;
  /** Count of in-flight detail-triggered single jobs (useGenerationJobsFeed.singleJobs). */
  singleJobCount: number;
}): WorkspaceMode {
  const { probing, batchStatus, hasItems, singleJobCount } = input;
  if (probing) return 'loading';
  if (batchStatus === 'running') return hasItems ? 'running' : 'attach';
  if (BATCH_TERMINALS.has(batchStatus)) return batchStatus as WorkspaceMode;
  // batchStatus === 'idle'
  return singleJobCount > 0 ? 'single' : 'idle';
}

/** A mode where the live event-log pane + SSE indicator are meaningful (something is/was happening). */
export function modeShowsFeed(mode: WorkspaceMode): boolean {
  return mode !== 'loading' && mode !== 'idle';
}
