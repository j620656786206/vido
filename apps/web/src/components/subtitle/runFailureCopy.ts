// Implements: <utility — no .pen counterpart>
/**
 * The failure words for a subtitle run, keyed by the backend's reason code
 * (`models.SubtitleRunFailureReason`). A leaf module on purpose
 * (disc-2026-10-generation-queue-failure-reason AC #1): the live log
 * (generationEventCopy), the activity record (recentEventCopy) AND the
 * generation queue (generationQueueRow) all read it, and generationEventCopy
 * already imports generationQueueRow — keeping the words here means none of
 * them import each other just to share a sentence. Do not import those
 * modules from this file.
 */

export interface FailureCopy {
  text: string;
  /**
   * The run stopped for money — the AI budget ceiling, or an AI account with no
   * credit left. That is "you asked and it did not happen" (ochre), not "it
   * broke" (cinnabar) — the log renders it as 已停止 (F11-SPEC-LOG ⑤ / ⑤b).
   */
  stopped: boolean;
  /**
   * 沒做成，但不是壞掉、也不是錢的問題——畫成灰色「已略過」
   * (disc-2026-10-no-text-source-skip-b AC #1): the film is fine, there was
   * simply nothing to generate from. Never `true` together with `stopped`.
   */
  skipped: boolean;
}

/**
 * The reason codes the backend sends as `failure_reason` — on a finished
 * subtitle run in GET /api/v1/activity (disc-2026-10-activity-subtitle-run-
 * time-a AC #1 `[@contract-v2]`; v2 added `queue_timeout`) and on the live
 * generation events (disc-2026-10-live-activity-failure-reasons-a AC #1
 * `[@contract-v1]`). One classifier decides them all
 * (`models.SubtitleRunFailureReason`); the client only looks them up.
 * `no_text_source` is the ONLY code for "not done, but nothing is broken" —
 * no usable subtitle in the file and no speech recognition set up
 * (disc-2026-10-no-text-source-skip-a AC #1 `[@contract-v2]`); it is the one
 * entry with `skipped: true`.
 * (`queue_timeout` in practice never reaches the live log: only the automatic
 * free lane can time out in the extraction queue, and it does not send
 * transcription_failed.)
 */
export type RunFailureReason =
  | 'queue_timeout'
  | 'cancelled'
  | 'budget_exceeded'
  | 'credit_exhausted'
  | 'target_not_writable'
  | 'no_audio_track'
  | 'ffmpeg_missing'
  | 'asr_unconfigured'
  | 'unauthorized'
  | 'timeout'
  | 'no_text_source'
  | 'other';

/**
 * One sentence per reason — the single copy of the failure words, read by both
 * the live log (feedRowView) and the activity record (recentEventCopy), so the
 * two surfaces never word the same reason two ways.
 */
export const FAILURE_COPY_BY_REASON: Readonly<Record<RunFailureReason, FailureCopy>> = {
  // disc-2026-10-run-timeout-not-cancel (⚖️ Sally 2026-10-19): waited too long
  // for the extraction slot. Same words and colour as `timeout` — nobody
  // stopped it, so never 中途停止.
  queue_timeout: Object.freeze({ text: '處理逾時', stopped: false, skipped: false }),
  // ⚖️ Sally 2026-10-19: a user cancel or a container restart mid-run.
  cancelled: Object.freeze({ text: '中途停止', stopped: false, skipped: false }),
  budget_exceeded: Object.freeze({ text: '已達預算上限', stopped: true, skipped: false }),
  // An empty provider account (disc-2026-09-asr-quota-429-treated-as-transient):
  // not a failure — the film is fine, the account needs a top-up (F11-SPEC-LOG ⑤b).
  credit_exhausted: Object.freeze({ text: 'AI 帳號沒有額度', stopped: true, skipped: false }),
  // The first half of the 409 sentence (transcription_handler.go). Never the
  // second half 「這次沒有花到錢」: a run can fail to write AFTER translating
  // (正式機 和平使者 S01E01 spent $0.196).
  target_not_writable: Object.freeze({
    text: '影片所在的資料夾寫不進字幕',
    stopped: false,
    skipped: false,
  }),
  no_audio_track: Object.freeze({ text: '找不到可用的音軌', stopped: false, skipped: false }),
  ffmpeg_missing: Object.freeze({ text: '伺服器缺少 ffmpeg', stopped: false, skipped: false }),
  asr_unconfigured: Object.freeze({ text: '語音辨識未設定', stopped: false, skipped: false }),
  unauthorized: Object.freeze({ text: 'API 金鑰無效', stopped: false, skipped: false }),
  timeout: Object.freeze({ text: '處理逾時', stopped: false, skipped: false }),
  // disc-2026-10-no-text-source-skip-b: no subtitle to take and no ASR to fall
  // back on. Not a failure — a grey 已略過 whose second line says what is
  // missing. The backend's sentence (solo_runner.go) minus 「這部影片」.
  no_text_source: Object.freeze({
    text: '沒有可用的字幕來源，也沒有設定語音辨識',
    stopped: false,
    skipped: true,
  }),
  other: Object.freeze({ text: '生成失敗', stopped: false, skipped: false }),
};

/** Copy for a wire reason code; an unknown or missing code says 生成失敗. */
export function failureCopyForReason(reason: string | null | undefined): FailureCopy {
  return Object.prototype.hasOwnProperty.call(FAILURE_COPY_BY_REASON, reason ?? '')
    ? FAILURE_COPY_BY_REASON[reason as RunFailureReason]
    : FAILURE_COPY_BY_REASON.other;
}
