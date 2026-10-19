package models

import "strings"

// Failure-reason codes for a failed subtitle run, as the activity record sends
// them (GET /api/v1/activity recent.events[].failure_reason —
// disc-2026-10-activity-subtitle-run-time-a AC #1 [@contract-v1]). Codes only:
// the words live on the client (apps/web generationEventCopy.ts
// FAILURE_COPY_BY_REASON), and the raw error_message never leaves the server —
// it carries NAS paths and ffmpeg stderr.
const (
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
//   - cancelled first — a cancel can interrupt any step, so its message may
//     also mention whatever that step was doing;
//   - target_not_writable before timeout — the write probe's timeout message
//     (fsprobe.ProbeWritableContext) says both "not writable" and "timed out",
//     and the user's fix is the folder, not patience.
//
// The match strings are the same set as the `failureCopy` TABLE in
// apps/web/src/components/subtitle/generationEventCopy.ts (the live log's
// raw-string matcher), plus cancelled and target_not_writable, which only this
// side knows. Adding a rule means changing both sides together.
var subtitleRunFailureRules = []struct {
	reason string
	match  func(msg string) bool
}{
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
// ten SubtitleRunFailure* codes. Pure: no I/O, no clock. An empty message or
// anything unrecognised (AI_INVALID_RESPONSE included — ⚖️ Sally 2026-10-19
// 追認 4) is SubtitleRunFailureOther.
//
// error_message is stored truncated (~1000 bytes, process_item.go /
// transcription_ledger.go) and may carry ffmpeg stderr: a cause written at the
// very end of a long path can be cut off (→ other), and stderr text can
// contain a later rule's words. Classifying at write time would remove both;
// today the stored text is all there is.
//
// Same match strings as `failureCopy`'s TABLE in
// apps/web/src/components/subtitle/generationEventCopy.ts — a new rule is
// added on both sides together.
func SubtitleRunFailureReason(errorMessage string) string {
	for _, r := range subtitleRunFailureRules {
		if r.match(errorMessage) {
			return r.reason
		}
	}
	return SubtitleRunFailureOther
}
