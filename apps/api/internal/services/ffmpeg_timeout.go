package services

import "time"

// BytesPerGB is the decimal gigabyte ffmpeg users think in.
const BytesPerGB = 1_000_000_000

// The two environment variables that decide how long one ffmpeg pass over a
// media file may take. Named here so a timeout message, the subtitle
// extractor, the audio extractor and docs/deployment.md cannot drift apart.
const (
	FFmpegTimeoutFloorEnv = "SUBTITLE_EXTRACT_TIMEOUT_SECONDS"
	FFmpegTimeoutPerGBEnv = "SUBTITLE_EXTRACT_PER_GB_SECONDS"
)

// SizedFFmpegTimeout is the deadline one ffmpeg read of a sizeBytes-large file
// gets: max(floor, size × perGB). Reading a file end to end is disk I/O
// proportional to its size, so a fixed number is wrong at both ends — the
// subtitle extractor learned this in sub-6-3, and the ASR audio extractor
// ran on a fixed 5 minutes until disc-2026-09-transcription-run-5min-hard-timeout
// (a 66.8 GB remux needed 4:55 of it on the NAS).
//
// An unknown size (≤ 0) or a non-positive allowance gets the floor — the
// bound the caller configured, never less.
func SizedFFmpegTimeout(floor, perGB time.Duration, sizeBytes int64) time.Duration {
	if sizeBytes <= 0 || perGB <= 0 {
		return floor
	}
	sized := time.Duration(float64(sizeBytes) / BytesPerGB * float64(perGB))
	if sized > floor {
		return sized
	}
	return floor
}

// sizedFFmpegTimeoutKnob is SizedFFmpegTimeout plus WHICH knob produced the
// bound, so a timeout message can name the one an operator would actually
// have to change: telling someone to raise the floor when the size term
// already exceeds it is advice that does nothing.
func sizedFFmpegTimeoutKnob(floor, perGB time.Duration, sizeBytes int64) (time.Duration, string) {
	got := SizedFFmpegTimeout(floor, perGB, sizeBytes)
	if got > floor {
		return got, FFmpegTimeoutPerGBEnv
	}
	return got, FFmpegTimeoutFloorEnv
}
