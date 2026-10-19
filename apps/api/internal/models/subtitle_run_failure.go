package models

import "strings"

// Failure-reason codes for a failed subtitle run, as the activity record sends
// them (GET /api/v1/activity recent.events[].failure_reason —
// disc-2026-10-activity-subtitle-run-time-a AC #1 [@contract-v2]; v2 added
// queue_timeout — disc-2026-10-run-timeout-not-cancel). Codes only:
// the words live on the client (apps/web generationEventCopy.ts
// FAILURE_COPY_BY_REASON), and the raw error_message never leaves the server —
// it carries NAS paths and ffmpeg stderr.
const (
	SubtitleRunFailureQueueTimeout      = "queue_timeout"
	SubtitleRunFailureCancelled         = "cancelled"
	SubtitleRunFailureBudgetExceeded    = "budget_exceeded"
	SubtitleRunFailureCreditExhausted   = "credit_exhausted"
	SubtitleRunFailureTargetNotWritable = "target_not_writable"
	SubtitleRunFailureNoAudioTrack      = "no_audio_track"
	SubtitleRunFailureFFmpegMissing     = "ffmpeg_missing"
	SubtitleRunFailureASRUnconfigured   = "asr_unconfigured"
	SubtitleRunFailureUnauthorized      = "unauthorized"
	SubtitleRunFailureTimeout           = "timeout"
	SubtitleRunFailureOther             = "other"
)

// subtitleRunFailureRules is checked top to bottom; the first rule that
// matches wins. The order is load-bearing:
//   - queue_timeout first, even before cancelled — rows written before
//     disc-2026-10-run-timeout-not-cancel recorded a queue timeout under the
//     `cancelled: ` prefix, so an old row carries BOTH the cancelled prefix
//     and the deadline words, and it is a timeout. A real cancel while queued
//     ends in "context canceled", never "deadline exceeded", so it still falls
//     through to cancelled;
//   - cancelled next — a cancel can interrupt any step, so its message may
//     also mention whatever that step was doing;
//   - target_not_writable before timeout — the write probe's timeout message
//     (fsprobe.ProbeWritableContext) says both "not writable" and "timed out",
//     and the user's fix is the folder, not patience.
//
// The match strings are the same set as the `failureCopy` TABLE in
// apps/web/src/components/subtitle/generationEventCopy.ts (the live log's
// raw-string matcher), plus queue_timeout, cancelled and target_not_writable,
// which only this side knows. The live log can still show a cancel or a write
// failure — it just words them 生成失敗, because its raw-string matcher does
// not know them (disc-2026-10-live-activity-failure-reasons-b moves it onto
// these codes). queue_timeout never reaches the live log at all: only the free
// lane can time out in the extraction queue, and it does not emit
// transcription_failed. Until -b lands, a rule the live log can see is added
// on both sides together.
var subtitleRunFailureRules = []struct {
	reason string
	match  func(msg string) bool
}{
	{SubtitleRunFailureQueueTimeout, isSubtitleRunQueueTimeout},
	{SubtitleRunFailureCancelled, func(m string) bool {
		return strings.HasPrefix(m, SubtitleRunCancelledPrefix) || strings.Contains(m, "context canceled")
	}},
	{SubtitleRunFailureBudgetExceeded, containsAny("AI_BUDGET_EXCEEDED")},
	{SubtitleRunFailureCreditExhausted, containsAny("AI_CREDIT_EXHAUSTED", "帳號沒有額度")},
	{SubtitleRunFailureTargetNotWritable, func(m string) bool {
		if containsAny("SUBTITLE_TARGET_NOT_WRITABLE", "target directory is not writable")(m) {
			return true
		}
		// The placer found out only at write time — after the paid work ran
		// (正式機 和平使者 S01E01: permission denied, $0.196 spent).
		return strings.Contains(m, "placer: write failed") &&
			containsAny("permission denied", "read-only file system")(m)
	}},
	{SubtitleRunFailureNoAudioTrack, containsAny("no audio track found")},
	{SubtitleRunFailureFFmpegMissing, containsAny("ffmpeg not available")},
	{SubtitleRunFailureASRUnconfigured, containsAny("transcription unavailable", "API key not configured")},
	{SubtitleRunFailureUnauthorized, containsAny("AI_UNAUTHORIZED", "status 401")},
	{SubtitleRunFailureTimeout, containsAny("AI_TIMEOUT", "timed out", "deadline exceeded")},
}

// SubtitleRunCancelledPrefix is the error_message prefix of a run the CALLER
// cancelled. It lives here so the classifier and the writer share one value:
// subtitle.CancelledRunPrefix (process_item.go) is defined as this constant.
const SubtitleRunCancelledPrefix = "cancelled: "

// SubtitleRunQueueTimeoutPrefix is the error_message prefix of a run whose
// deadline ran out while it was QUEUED for the embedded-subtitle extraction
// slot (or lost so much of its budget there that ffmpeg was stopped by the
// caller's deadline). Like the cancelled prefix it says nothing about the
// file, so the free lane's three-strikes parking skips it; unlike a cancel,
// nobody stopped it — the activity record says 處理逾時, not 中途停止.
// subtitle.QueueTimeoutRunPrefix is defined as this constant.
const SubtitleRunQueueTimeoutPrefix = "queue-timeout: "

// subtitleExtractWaitAbortedSentinel is subtitle.ErrSubtitleExtractWaitAborted's
// code, written out here because models cannot import subtitle (subtitle
// imports models). A cross-package test in subtitle
// (TestQueueTimeoutClassifierTracksTheWaitAbortedSentinel) fails if the
// sentinel's text changes without this one.
const subtitleExtractWaitAbortedSentinel = "SUBTITLE_EXTRACT_WAIT_ABORTED"

// isSubtitleRunQueueTimeout is the queue_timeout rule. The prefix is the
// reliable signal (decided at write time from the real error type). The
// string fallback catches what has no prefix — an error still in memory, or a
// row written before the prefix existed (`cancelled: …WAIT_ABORTED…context
// deadline exceeded`). It needs BOTH words: "deadline exceeded" alone is an
// ordinary per-item timeout (timeout), and WAIT_ABORTED alone may be a cancel.
// An old row truncated before its trailing "deadline exceeded" cannot be told
// apart and stays cancelled — a known, accepted limit.
func isSubtitleRunQueueTimeout(m string) bool {
	if strings.HasPrefix(m, SubtitleRunQueueTimeoutPrefix) {
		return true
	}
	return strings.Contains(m, subtitleExtractWaitAbortedSentinel) &&
		strings.Contains(m, "deadline exceeded")
}

func containsAny(needles ...string) func(string) bool {
	return func(m string) bool {
		for _, n := range needles {
			if strings.Contains(m, n) {
				return true
			}
		}
		return false
	}
}

// SubtitleRunFailureReason maps a failed run's raw error_message to one of the
// eleven SubtitleRunFailure* codes. Pure: no I/O, no clock. An empty message or
// anything unrecognised (AI_INVALID_RESPONSE included — ⚖️ Sally 2026-10-19
// 追認 4) is SubtitleRunFailureOther.
//
// error_message is stored truncated (~1000 bytes, process_item.go /
// transcription_ledger.go) and may carry ffmpeg stderr: a cause written at the
// very end of a long path can be cut off (→ other), and stderr text can
// contain a later rule's words. The two cases decided at WRITE time — the
// cancelled and queue-timeout prefixes, which failItem sets from the real
// error type — survive truncation; for every other rule the stored text is
// all there is.
//
// Apart from those prefixes and target_not_writable, the match strings are
// the same as `failureCopy`'s TABLE in
// apps/web/src/components/subtitle/generationEventCopy.ts (see the note on
// subtitleRunFailureRules).
func SubtitleRunFailureReason(errorMessage string) string {
	for _, r := range subtitleRunFailureRules {
		if r.match(errorMessage) {
			return r.reason
		}
	}
	return SubtitleRunFailureOther
}
