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

// Every code the function can return is one of the ten the contract names —
// the web client's FAILURE_COPY_BY_REASON has exactly these keys.
func TestSubtitleRunFailureReason_CodesAreTheContract(t *testing.T) {
	contract := map[string]bool{
		"cancelled": true, "budget_exceeded": true, "credit_exhausted": true,
		"target_not_writable": true, "no_audio_track": true, "ffmpeg_missing": true,
		"asr_unconfigured": true, "unauthorized": true, "timeout": true, "other": true,
	}
	seen := map[string]bool{SubtitleRunFailureOther: true}
	for _, r := range subtitleRunFailureRules {
		assert.Truef(t, contract[r.reason], "rule code %q is not in the [@contract-v1] list", r.reason)
		assert.Falsef(t, seen[r.reason], "code %q has two rules — merge them so the order stays readable", r.reason)
		seen[r.reason] = true
	}
	assert.Len(t, seen, len(contract), "every contract code must be reachable")
}
