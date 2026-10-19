package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The four 正式機 failures are pasted byte for byte from
// `sqlite3 -readonly /mnt/cache/appdata/vido/vido.db` (2026-10-19), including
// the server paths and ffmpeg's carriage returns (sqlite3 prints them as ^M) — that is exactly why the raw text never goes on the wire
// and only the code does.
const (
	prodInvalidResponse = `subtitle pipeline: episode c5eed5aa-4a23-4826-ad39-0ffe5d8f2a76: translate: SUBTITLE_TRANSLATE_FAILED: LLM subtitle translation failed: cues 151-200: translate chunk [151-200]: AI_INVALID_RESPONSE: Cannot parse AI response`
	prodAITimeout       = `subtitle pipeline: movie bea8e0b8-7bbd-4b41-a88c-63f779b8180b: translate: SUBTITLE_TRANSLATE_FAILED: LLM subtitle translation failed: cues 478-489: translate chunk [478-489]: AI_TIMEOUT: AI parsing timeout exceeded`
	prodFFmpegTimeout   = "subtitle pipeline: movie 55bcc4cd-f9a8-4c3e-ac86-e444c00078e9: route: subtitle route: extract /media/movies/GoodFellas (1990)/Goodfellas.1990.BDRemux.2160p.HDR-DV.mkv: SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed: ffmpeg timed out after 10m0s on Goodfellas.1990.BDRemux.2160p.HDR-DV.mkv (ffmpeg stderr: s/s speed=2.85x    \rsize=      34kB time=00:28:10.85 bitrate=   0.2kbits/s speed=2.85x    \rsize=      34kB time=00:28:13.23 bitrate=   0.2kbits/s speed=2.85x    \rsize=      34kB time=00:28:15.98 bitrate=   0.2kbits/s speed=2.85x    \rsize=      35kB time=00:28:17.94 bitrate=   0.2kbits/s speed=2.85x    \rsize=      35kB time=00:28:22.70 bitrate=   0.2kbits/s speed=2.85x    \rsize=      35kB time=00:28:25.24 bitrate=   0.2kbits/s speed=2.85x    \rsize=      35kB time=00:28:28.45 bitrate=   0.2kbits/s speed=2.85x): context deadline exceeded"
	prodPlacerDenied    = `subtitle pipeline: episode 2c5d56d4-9c3c-487d-871b-7d2d59ca753b: place: placer: write failed: write temp file: open /media/tv/Peacemaker/Season01/.Peacemaker.S01E01.2160p.MAX.WEB-DL.DDPA5.1.HDR.DV.HEVC-SPAMKiNGS.zh-Hant.srt.tmp.2903780514314573825: permission denied`
)

func TestSubtitleRunFailureReason(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want string
	}{
		// 正式機, verbatim.
		{"prod: AI_INVALID_RESPONSE (末日光明 S01E04) is other", prodInvalidResponse, SubtitleRunFailureOther},
		{"prod: AI_TIMEOUT (動物方城市2)", prodAITimeout, SubtitleRunFailureTimeout},
		{"prod: ffmpeg extraction timed out (四海好傢伙)", prodFFmpegTimeout, SubtitleRunFailureTimeout},
		{"prod: placer permission denied after translating (和平使者 S01E01)", prodPlacerDenied, SubtitleRunFailureTargetNotWritable},

		// One per code.
		{
			"queue-timeout prefix",
			"queue-timeout: subtitle pipeline: movie x: route: subtitle route: extract /media/movies/x.mkv: SUBTITLE_EXTRACT_WAIT_ABORTED: gave up waiting for the extraction slot: x.mkv (queued behind 1 for 15m0s): context deadline exceeded",
			SubtitleRunFailureQueueTimeout,
		},
		{"cancelled prefix", "cancelled: subtitle pipeline: episode x: translate: context canceled", SubtitleRunFailureCancelled},
		{"bare context canceled", "subtitle pipeline: movie x: extract: context canceled", SubtitleRunFailureCancelled},
		{"budget", "subtitle pipeline: episode x: translate: AI_BUDGET_EXCEEDED: per-run AI budget ceiling reached", SubtitleRunFailureBudgetExceeded},
		{"credit (legacy code)", "transcribe: AI_CREDIT_EXHAUSTED: openai account has no credit left (insufficient_quota)", SubtitleRunFailureCreditExhausted},
		{"credit (solo runner sentence)", "OpenAI 帳號沒有額度，請儲值後再試", SubtitleRunFailureCreditExhausted},
		{"target sentinel", "subtitle pipeline: episode x: SUBTITLE_TARGET_NOT_WRITABLE: target directory is not writable", SubtitleRunFailureTargetNotWritable},
		{"fsprobe sentinel", "target directory is not writable: /media/tv/x: open .vido-probe: permission denied", SubtitleRunFailureTargetNotWritable},
		{"placer read-only fs", "place: placer: write failed: write temp file: open /media/x.srt.tmp: read-only file system", SubtitleRunFailureTargetNotWritable},
		{"no audio", "extract audio: no audio track found in media file", SubtitleRunFailureNoAudioTrack},
		{"ffmpeg missing", "extract audio: ffmpeg not available", SubtitleRunFailureFFmpegMissing},
		{"asr unconfigured (whisper)", "whisper: OpenAI API key not configured", SubtitleRunFailureASRUnconfigured},
		{"asr unconfigured (resume)", "transcription unavailable and resume not possible", SubtitleRunFailureASRUnconfigured},
		{"unauthorized (LLM)", "translate: AI_UNAUTHORIZED: Provider rejected the API key", SubtitleRunFailureUnauthorized},
		{"unauthorized (whisper)", "whisper: request failed with status 401", SubtitleRunFailureUnauthorized},
		{"deadline exceeded", "subtitle pipeline: movie x: context deadline exceeded", SubtitleRunFailureTimeout},
		{"empty", "", SubtitleRunFailureOther},
		{"unknown", "something nobody planned for", SubtitleRunFailureOther},

		// Order.
		{
			// fsprobe.ProbeWritableContext's timeout wording: both "not
			// writable" and "timed out" — the folder is the user's fix.
			"write-probe timeout is target_not_writable, not timeout",
			"subtitle pipeline: episode x: target directory is not writable: /media/tv/x: probe timed out (context deadline exceeded)",
			SubtitleRunFailureTargetNotWritable,
		},
		{"cancel beats budget", "cancelled: AI_BUDGET_EXCEEDED: per-run AI budget ceiling reached", SubtitleRunFailureCancelled},
		{"budget beats timeout", "AI_BUDGET_EXCEEDED after AI_TIMEOUT", SubtitleRunFailureBudgetExceeded},
		{
			"placer failure without a permission cause is other",
			"place: placer: write failed: write temp file: no space left on device",
			SubtitleRunFailureOther,
		},
		{
			"permission denied outside the placer is not a write failure",
			"extract /media/x.mkv: open /media/x.mkv: permission denied",
			SubtitleRunFailureOther,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, SubtitleRunFailureReason(tc.msg))
		})
	}
}

// disc-2026-10-run-timeout-not-cancel: a deadline that ran out while the item
// was QUEUED for the extraction slot is queue_timeout, whether the row carries
// the new prefix, the pre-v2 `cancelled: ` prefix, or no prefix at all — and a
// real cancel while queued stays cancelled.
func TestSubtitleRunFailureReason_QueueTimeout(t *testing.T) {
	const waitAborted = "subtitle pipeline: movie 1f0c: route: subtitle route: extract /media/movies/Big (2024)/Big.2160p.mkv: SUBTITLE_EXTRACT_WAIT_ABORTED: gave up waiting for the extraction slot: Big.2160p.mkv"
	cases := []struct {
		name string
		msg  string
		want string
	}{
		{"new prefix", "queue-timeout: " + waitAborted + " (queued behind 1 for 15m0s): context deadline exceeded", SubtitleRunFailureQueueTimeout},
		{
			// Written before this story: the queue timeout was recorded under
			// the cancelled prefix. The rule must sit ABOVE cancelled for this.
			"pre-v2 row: cancelled prefix + deadline words",
			"cancelled: " + waitAborted + " (queued behind 2 for 15m0s): context deadline exceeded",
			SubtitleRunFailureQueueTimeout,
		},
		{
			// The prefix alone decides: the stored row was cut (~1000 bytes)
			// before the trailing ctx error.
			"new prefix, truncated before the deadline words",
			"queue-timeout: " + waitAborted + ": SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed: ffmpeg stopped by the caller's deadline after 3m0s on Big.2160p.mkv (ffmpeg stderr: size=      34kB time=00:28:10.85…",
			SubtitleRunFailureQueueTimeout,
		},
		{"cancelled while queued stays cancelled", "cancelled: " + waitAborted + " (queued behind 1 for 3m0s): context canceled", SubtitleRunFailureCancelled},
		{"in-memory shape (no prefix)", waitAborted + " (queued behind 1 for 15m0s): context deadline exceeded", SubtitleRunFailureQueueTimeout},
		{
			// extractor.go: queued, then ffmpeg ran out of the caller's budget.
			"ffmpeg stopped by the caller's deadline after a wait",
			"subtitle pipeline: movie x: route: subtitle route: extract /media/movies/x.mkv: SUBTITLE_EXTRACT_WAIT_ABORTED: gave up waiting for the extraction slot: SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed: ffmpeg stopped by the caller's deadline after 3m0s on x.mkv (file 61.2 GB; 12m0s of that budget went to waiting for the extraction slot): context deadline exceeded",
			SubtitleRunFailureQueueTimeout,
		},
		// The deadline words alone are an ordinary per-item timeout.
		{"prod AI_TIMEOUT stays timeout", prodAITimeout, SubtitleRunFailureTimeout},
		{"prod ffmpeg timeout stays timeout", prodFFmpegTimeout, SubtitleRunFailureTimeout},
		{
			// Known limit: a pre-v2 row cut off (~1000 bytes) before its
			// trailing "deadline exceeded" cannot be told from a cancel.
			"pre-v2 row truncated before the deadline words stays cancelled",
			"cancelled: " + waitAborted + ": SUBTITLE_EXTRACT_FAILED: ffmpeg subtitle extraction failed: ffmpeg stopped by the caller's deadline after 3m0s on Big.2160p.mkv (file 61.2 GB; 12m0s of that budget went to waiting for the extraction slot) (ffmpeg stderr: size=      34kB time=00:28:10.85 bitrate=   0.2kbits/s speed=2.85x",
			SubtitleRunFailureCancelled,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, SubtitleRunFailureReason(tc.msg))
		})
	}
}

// disc-2026-10-live-activity-failure-reasons-a AC #5: every input of the web
// client's former raw-string matcher (generationEventCopy.spec.ts
// describe('failureCopy'), deleted by -b along with the matcher) lives here, with the code
// whose words the client showed for it. All of them agreed when moved
// (2026-10-19) — the client's 已達預算上限 is budget_exceeded, 找不到可用的音軌
// no_audio_track, and so on.
func TestSubtitleRunFailureReason_LiveLogStrings(t *testing.T) {
	cases := []struct {
		msg  string
		want string
	}{
		{"translate: translate: translation stopped at block 3: AI_BUDGET_EXCEEDED: per-run AI budget ceiling reached", SubtitleRunFailureBudgetExceeded},
		{"select audio track: no audio track found in media file", SubtitleRunFailureNoAudioTrack},
		{"extract audio: ffmpeg not available", SubtitleRunFailureFFmpegMissing},
		{"list audio tracks: ffmpeg not available", SubtitleRunFailureFFmpegMissing},
		{"transcription unavailable and resume not possible", SubtitleRunFailureASRUnconfigured},
		{"transcribe: transcribe chunk 2/9: AI_UNAUTHORIZED: invalid key", SubtitleRunFailureUnauthorized},
		{"transcribe: AI_TIMEOUT: provider did not answer", SubtitleRunFailureTimeout},
		{"transcribe: transcription run timed out after 45m2s in transcribing (media 157 min, budget 4712 s — raise TRANSCRIPTION_SECONDS_PER_MEDIA_MINUTE): context deadline exceeded", SubtitleRunFailureTimeout},
		{"extract audio: audio extraction timed out: ffmpeg timed out after 33m24s on Movie.mkv (file 66.8 GB, timeout 2004 s — raise SUBTITLE_EXTRACT_PER_GB_SECONDS for slow disks): context deadline exceeded", SubtitleRunFailureTimeout},
		{`transcribe: whisper: API error: status 401 — {"error":"invalid key"}`, SubtitleRunFailureUnauthorized},
		{"transcribe: whisper: OpenAI API key not configured", SubtitleRunFailureASRUnconfigured},
		{"OpenAI 帳號沒有額度了，請到 https://platform.openai.com/settings/organization/billing 儲值後再按一次生成", SubtitleRunFailureCreditExhausted},
		{"transcribe: transcribe chunk 1/6: whisper: API error: AI_CREDIT_EXHAUSTED: openai account has no credit left (insufficient_quota)", SubtitleRunFailureCreditExhausted},
		{"transcribe: whisper: request timed out", SubtitleRunFailureTimeout},
		{"extract audio: context deadline exceeded", SubtitleRunFailureTimeout},
		// The client's fallbacks (生成失敗).
		{"save SRT: open /mnt/media/Movies/Dune (2024)/Dune.en.srt: permission denied", SubtitleRunFailureOther},
		{"save SRT: open /mnt/media/secret/path.srt: no space left on device", SubtitleRunFailureOther},
		{"", SubtitleRunFailureOther},
		// The client's "first match wins" case: a budget stop inside a
		// timeout-shaped message is still a budget stop.
		{"transcribe: timed out after AI_BUDGET_EXCEEDED: ceiling", SubtitleRunFailureBudgetExceeded},
	}
	for _, tc := range cases {
		t.Run(tc.msg, func(t *testing.T) {
			assert.Equal(t, tc.want, SubtitleRunFailureReason(tc.msg))
		})
	}
}

// Every code the function can return is one of the eleven the contract names —
// the web client's FAILURE_COPY_BY_REASON has exactly these keys.
func TestSubtitleRunFailureReason_CodesAreTheContract(t *testing.T) {
	contract := map[string]bool{
		"queue_timeout": true, "cancelled": true, "budget_exceeded": true, "credit_exhausted": true,
		"target_not_writable": true, "no_audio_track": true, "ffmpeg_missing": true,
		"asr_unconfigured": true, "unauthorized": true, "timeout": true, "other": true,
	}
	seen := map[string]bool{SubtitleRunFailureOther: true}
	for _, r := range subtitleRunFailureRules {
		assert.Truef(t, contract[r.reason], "rule code %q is not in the [@contract-v2] list", r.reason)
		assert.Falsef(t, seen[r.reason], "code %q has two rules — merge them so the order stays readable", r.reason)
		seen[r.reason] = true
	}
	assert.Len(t, seen, len(contract), "every contract code must be reachable")
}
