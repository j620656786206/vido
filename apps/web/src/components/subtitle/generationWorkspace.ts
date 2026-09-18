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
 *  - `running`   — batch running AND the BACKEND's `progress.items[]` is known
 *                  (dsr-6d-c-1: from the status probe / SSE, no longer the
 *                  launcher's 202 cache) → full queue.
 *  - `attach`    — batch running but no queue at all: a pre-dsr-6d-a server, or the
 *                  moments before the first snapshot lands → degraded: counters +
 *                  in-flight card only, honest note.
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
  /** Whether the BACKEND's queue (`progress.items[]`) is available (dsr-6d-c-1). */
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
