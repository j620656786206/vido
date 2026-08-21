package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vido/api/internal/ai"
)

// ─── 9R-5: whole-file guard against an over-eager hallucination filter ───────

// plainASR implements only ai.ASRProvider — the shape an alternative engine
// that never grew the detailed seam would have.
type plainASR struct{ srt string }

func (p *plainASR) Transcribe(ctx context.Context, audioPath string) (string, error) {
	return p.srt, nil
}
func (p *plainASR) TranscribeWithLanguage(ctx context.Context, audioPath, lang string) (string, error) {
	return p.srt, nil
}

// detailedASR also implements ai.DetailedTranscriber, so the guard can see what
// the filter removed.
type detailedASR struct {
	plainASR
	detail ai.TranscriptionDetail
	calls  int
}

func (d *detailedASR) TranscribeDetailed(ctx context.Context, audioPath, lang string) (ai.TranscriptionDetail, error) {
	d.calls++
	return d.detail, nil
}

// smallAudio writes a file below the chunking threshold so transcribeAudio
// takes the single-request path.
func smallAudio(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audio.wav")
	require.NoError(t, os.WriteFile(path, []byte("RIFFfakewavdata"), 0644))
	return path
}

func newASRWiredService(asr ai.ASRProvider) *TranscriptionService {
	svc := NewTranscriptionService(nil, asr, nil, nil)
	return svc
}

// The headline recovery: the filter ate everything, so the unfiltered text
// ships instead. Without this the pipeline writes a 0-byte .en.srt and the row
// ends up claiming `untranslated` over an empty file.
func TestTranscribeAudio_EmptyAfterFilteringFallsBackToUnfiltered(t *testing.T) {
	unfiltered := "1\n00:00:01,000 --> 00:00:02,000\nThanks for watching\n\n"
	asr := &detailedASR{detail: ai.TranscriptionDetail{
		SRT: "", Unfiltered: unfiltered, SegmentsIn: 1, SegmentsKept: 0, Filtered: true,
	}}

	got, err := newASRWiredService(asr).transcribeAudio(context.Background(), smallAudio(t), "en")
	require.NoError(t, err)
	assert.Equal(t, unfiltered, got, "an entirely emptied file must deliver the unfiltered text")
	assert.Equal(t, 1, asr.calls, "the guard must not re-transcribe")
}

// The normal case must be untouched by the guard.
func TestTranscribeAudio_PartiallyFilteredKeepsTheFilteredText(t *testing.T) {
	filtered := "1\n00:00:01,000 --> 00:00:02,000\nReal line\n\n"
	unfiltered := filtered + "2\n00:01:40,000 --> 00:01:43,000\nLike and subscribe\n\n"
	asr := &detailedASR{detail: ai.TranscriptionDetail{
		SRT: filtered, Unfiltered: unfiltered, SegmentsIn: 2, SegmentsKept: 1, Filtered: true,
	}}

	got, err := newASRWiredService(asr).transcribeAudio(context.Background(), smallAudio(t), "en")
	require.NoError(t, err)
	assert.Equal(t, filtered, got)
	assert.NotContains(t, got, "subscribe")
}

// Genuinely silent audio yields nothing from BOTH sides — the guard must not
// invent content, and must not treat "nothing to recover" as a failure.
func TestTranscribeAudio_NothingToRecoverStaysEmpty(t *testing.T) {
	asr := &detailedASR{detail: ai.TranscriptionDetail{SRT: "", Unfiltered: "", Filtered: true}}

	got, err := newASRWiredService(asr).transcribeAudio(context.Background(), smallAudio(t), "en")
	require.NoError(t, err)
	assert.Equal(t, "", got)
}

// An engine without the detailed seam degrades cleanly: same string both ways,
// so the guard is a no-op rather than an error.
func TestTranscribeAudio_ProviderWithoutDetailedSeamIsUnaffected(t *testing.T) {
	srt := "1\n00:00:01,000 --> 00:00:02,000\nPlain engine\n\n"

	got, err := newASRWiredService(&plainASR{srt: srt}).transcribeAudio(context.Background(), smallAudio(t), "en")
	require.NoError(t, err)
	assert.Equal(t, srt, got)
}

func TestCountSRTCues(t *testing.T) {
	assert.Equal(t, 0, countSRTCues(""))
	assert.Equal(t, 0, countSRTCues("not a subtitle at all"))
	assert.Equal(t, 2, countSRTCues(
		"1\n00:00:01,000 --> 00:00:02,000\nA\n\n2\n00:00:03,000 --> 00:00:04,000\nB\n\n"))
}
