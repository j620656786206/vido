package services

// disc-2026-10-activity-generation-progress-detail — the run reports which
// chunk / cue it is on, and the Activity rows show it.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/qbittorrent"
)

// ─── Transcription: one report per finished chunk ──────────────────────────

func TestTranscribeAudio_ReportsEachFinishedChunk(t *testing.T) {
	installFakeChunkFFmpeg(t)
	wav := filepath.Join(t.TempDir(), "film.wav")
	writePayloadWAV(t, wav, 4*120) // 4 chunks on the 120 s grid
	repo := &fakeCacheRepo{}
	id := baseIdentity()

	// A first run stops at chunk 3: chunks 1–2 are stored.
	first := NewTranscriptionService(nil, &countingASR{failAt: 3}, nil, nil)
	first.SetASRChunkStore(NewASRChunkStore(repo))
	var firstReports [][2]int
	_, err := first.transcribeAudioWithProgress(context.Background(), wav, "en", chunkScopeFor(id),
		func(done, total int) { firstReports = append(firstReports, [2]int{done, total}) })
	require.Error(t, err)
	assert.Equal(t, [][2]int{{1, 4}, {2, 4}}, firstReports, "no report for the chunk that failed")

	// The resume counts the stored chunks as done, in order.
	second := NewTranscriptionService(nil, &countingASR{base: 2}, nil, nil)
	second.SetASRChunkStore(NewASRChunkStore(repo))
	var reports [][2]int
	_, err = second.transcribeAudioWithProgress(context.Background(), wav, "en", chunkScopeFor(id),
		func(done, total int) { reports = append(reports, [2]int{done, total}) })
	require.NoError(t, err)
	assert.Equal(t, [][2]int{{1, 4}, {2, 4}, {3, 4}, {4, 4}}, reports)
}

func TestTranscribeAudio_OneChunkFileReportsNothing(t *testing.T) {
	var reports int
	_, err := newASRWiredService(&countingASR{}).transcribeAudioWithProgress(context.Background(), smallAudio(t), "en", nil,
		func(int, int) { reports++ })
	require.NoError(t, err)
	assert.Zero(t, reports, `"1/1" says nothing the stage name does not`)
}

func TestTranscriptionChunkMessage(t *testing.T) {
	assert.Equal(t, "正在轉錄音訊（第 2/6 段）", transcriptionChunkMessage(2, 6))
}

// ─── Translation: cue counts on translation_progress ───────────────────────

func TestTranslateSRT_ProgressCarriesCueCounts(t *testing.T) {
	tmp := t.TempDir()
	enPath := filepath.Join(tmp, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(multiCueSRT(25)), 0o644))
	movie := untranslatedMovie(uuidD, enPath)
	svc, client := titleTestService(t, &countingCompleter{}, &fakeStateReader{movie: movie})

	_, err := svc.StartTranscription(context.Background(), uuidD, filepath.Join(tmp, "Movie.mkv"), tmp, WithTranslation())
	require.NoError(t, err)

	var counted [][2]int
	for _, ev := range collectUntilTerminal(t, client, uuidD) {
		if ev.Type != EventTranscriptionTranslating {
			continue
		}
		data := ev.Data.(map[string]interface{})
		cur, hasCur := data["current"]
		total, hasTotal := data["total"]
		if !hasCur || !hasTotal {
			continue // the 0% opening frame carries no count
		}
		counted = append(counted, [2]int{cur.(int), total.(int)})
	}
	// 25 cues in batches of 10: 10, 20, 25 of 25.
	require.NotEmpty(t, counted)
	assert.Equal(t, [2]int{25, 25}, counted[len(counted)-1])
	for _, c := range counted {
		assert.Equal(t, 25, c[1])
		assert.LessOrEqual(t, c[0], 25)
	}
}

// ─── Activity rows ─────────────────────────────────────────────────────────

type namedBatch struct {
	fakeBatch
	mediaID string
	named   bool
}

func (n namedBatch) ActiveMediaID() (string, bool) { return n.mediaID, n.named }

type fakeSteps map[string]ActiveJobStep

func (f fakeSteps) Step(mediaID string) (string, int, int, bool) {
	s, ok := f[mediaID]
	return s.Phase, s.Current, s.Total, ok
}

func activityWith(generation, transcription batchJobSource) *ActivityService {
	return NewActivityService(fakeScan{}, fakeBatch{}, generation, transcription,
		fakeDownloads{counts: &qbittorrent.DownloadCounts{}}, fakeParse{})
}

func jobOfKind(t *testing.T, a *ActivitySummary, kind string) ActiveJob {
	t.Helper()
	for _, j := range a.ActiveJobs.Jobs {
		if j.Kind == kind {
			return j
		}
	}
	t.Fatalf("no %s job in %+v", kind, a.ActiveJobs.Jobs)
	return ActiveJob{}
}

func TestActivity_TranscriptionRowShowsTheRunningItemsStep(t *testing.T) {
	solo := namedBatch{fakeBatch: fakeBatch{active: true, current: 1, item: "See S01E02"}, mediaID: "ep-1", named: true}
	svc := activityWith(fakeBatch{}, solo)
	svc.SetStepReader(fakeSteps{"ep-1": {Phase: "transcribing", Current: 2, Total: 6}})

	job := jobOfKind(t, ptr(svc.GetActivity(context.Background())), "transcription")
	require.NotNil(t, job.Step)
	assert.Equal(t, ActiveJobStep{Phase: "transcribing", Current: 2, Total: 6}, *job.Step)
	assert.Equal(t, 33, job.PercentDone, "the bar follows the chunks: 2×100÷6")
	assert.Equal(t, "See S01E02", job.Detail)
}

func TestActivity_TranscriptionStepWithoutCountKeepsNoPercent(t *testing.T) {
	solo := namedBatch{fakeBatch: fakeBatch{active: true, current: 1}, mediaID: "ep-1", named: true}
	svc := activityWith(fakeBatch{}, solo)
	svc.SetStepReader(fakeSteps{"ep-1": {Phase: "extracting"}})

	job := jobOfKind(t, ptr(svc.GetActivity(context.Background())), "transcription")
	require.NotNil(t, job.Step)
	assert.Equal(t, "extracting", job.Step.Phase)
	assert.Zero(t, job.PercentDone, "no count — still no fabricated percent")
}

func TestActivity_NoStepWhenItCannotBeNamed(t *testing.T) {
	steps := fakeSteps{"ep-1": {Phase: "transcribing", Current: 2, Total: 6}}
	cases := map[string]*ActivityService{
		"several solo jobs": activityWith(fakeBatch{},
			namedBatch{fakeBatch: fakeBatch{active: true, current: 2}, named: false}),
		"source cannot name its item": activityWith(fakeBatch{}, fakeBatch{active: true, current: 1}),
		"nothing recorded for the item": activityWith(fakeBatch{},
			namedBatch{fakeBatch: fakeBatch{active: true, current: 1}, mediaID: "ep-9", named: true}),
	}
	for name, svc := range cases {
		t.Run(name, func(t *testing.T) {
			svc.SetStepReader(steps)
			job := jobOfKind(t, ptr(svc.GetActivity(context.Background())), "transcription")
			assert.Nil(t, job.Step)
			assert.Zero(t, job.PercentDone)
		})
	}
	t.Run("no reader wired — the row is exactly what it was", func(t *testing.T) {
		svc := activityWith(fakeBatch{}, namedBatch{fakeBatch: fakeBatch{active: true, current: 1}, mediaID: "ep-1", named: true})
		job := jobOfKind(t, ptr(svc.GetActivity(context.Background())), "transcription")
		assert.Nil(t, job.Step)
	})
}

func TestActivity_BatchRowKeepsItemCountAndAddsTheCurrentItemsStep(t *testing.T) {
	batch := namedBatch{fakeBatch: fakeBatch{active: true, percentDone: 15, current: 3, total: 20, item: "See S01E02"},
		mediaID: "ep-1", named: true}
	svc := activityWith(batch, fakeBatch{})
	svc.SetStepReader(fakeSteps{"ep-1": {Phase: "translating", Current: 120, Total: 659}})

	job := jobOfKind(t, ptr(svc.GetActivity(context.Background())), "generation_batch")
	assert.Equal(t, 3, job.Current)
	assert.Equal(t, 20, job.Total)
	assert.Equal(t, 15, job.PercentDone, "the batch bar stays items, not cues")
	require.NotNil(t, job.Step)
	assert.Equal(t, ActiveJobStep{Phase: "translating", Current: 120, Total: 659}, *job.Step)
}

func ptr[T any](v T) *T { return &v }

// ─── The real sources name their one item ──────────────────────────────────

func TestTranscriptionService_ActiveMediaIDCountsSoloJobsOnly(t *testing.T) {
	svc := newActivityTestService()
	_, ok := svc.ActiveMediaID()
	assert.False(t, ok, "idle")

	_, err := svc.acquireJob("batch-1", "", false)
	require.NoError(t, err)
	_, ok = svc.ActiveMediaID()
	assert.False(t, ok, "a batch job is the batch row's")

	_, err = svc.acquireJob("solo-1", "沙丘", true)
	require.NoError(t, err)
	id, ok := svc.ActiveMediaID()
	assert.True(t, ok)
	assert.Equal(t, "solo-1", id)

	_, err = svc.acquireJob("solo-2", "奧本海默", true)
	require.NoError(t, err)
	_, ok = svc.ActiveMediaID()
	assert.False(t, ok, "two solo jobs — which one would the row describe?")
}
