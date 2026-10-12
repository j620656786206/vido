package services

// disc-2026-09-generation-resume-c-estimate-deduction — the quote on 生成字幕
// takes off what an earlier, unfinished run already paid for.

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/segkey"
)

// ─── Pricing (stub plan source) ────────────────────────────────────────────

func TestTranscriptionEstimate_StoredASRMinutesComeOffTheSpeechRecognitionHalf(t *testing.T) {
	plan := &stubPlanSource{available: true, translate: true, asrDone: 30}
	est := newTestEstimator(plan, false).Estimate(context.Background(), sixtyMinuteMovie())

	assert.Equal(t, TranscriptionPlanFull, est.Plan)
	// 30 × 0.006 (the half not yet transcribed) + 60 × 0.0216 (translation has
	// not started) = 0.18 + 1.296 = 1.476 → $1.48, down from $1.66.
	assert.Equal(t, 1.48, est.EstimatedUSD)
	assert.Equal(t, 30.0, est.ResumedASRMinutes)
	assert.Equal(t, 60.0, est.RuntimeMinutes, "runtime_minutes still means the whole film")
	assert.Zero(t, est.CachedTranslationShare)
	assert.Equal(t, []string{"asr:movie-1"}, plan.creditAsk)
}

func TestTranscriptionEstimate_EnglishOnlyRunPaysOnlyTheMissingAudio(t *testing.T) {
	est := newTestEstimator(&stubPlanSource{available: true, translate: false, asrDone: 45}, false).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, 0.09, est.EstimatedUSD) // 15 × 0.006
}

func TestTranscriptionEstimate_FullyTranscribedPaysOnlyTranslation(t *testing.T) {
	est := newTestEstimator(&stubPlanSource{available: true, translate: true, asrDone: 60}, false).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, 1.3, est.EstimatedUSD) // 60 × 0.0216
}

func TestTranscriptionEstimate_CachedCuesComeOffATranslateOnlyResume(t *testing.T) {
	plan := &stubPlanSource{available: true, translate: true, resumable: true, cachedShare: 0.75}
	est := newTestEstimator(plan, false).Estimate(context.Background(), sixtyMinuteMovie())

	assert.Equal(t, TranscriptionPlanTranslateOnly, est.Plan)
	assert.Equal(t, 0.32, est.EstimatedUSD) // 60 × 0.0216 × 0.25 = 0.324
	assert.Equal(t, 0.75, est.CachedTranslationShare)
	assert.Zero(t, est.ResumedASRMinutes)
	assert.Equal(t, []string{"translation:movie:movie-1"}, plan.creditAsk)

	all := newTestEstimator(&stubPlanSource{available: true, translate: true, resumable: true, cachedShare: 1}, false).
		Estimate(context.Background(), sixtyMinuteMovie())
	assert.Equal(t, 0.0, all.EstimatedUSD, "every cue cached — the resume sends nothing to the model")
}

func TestTranscriptionEstimate_CreditIsOnlyAskedWhereTheRunWouldUseIt(t *testing.T) {
	t.Run("no translation key: a resume bills nothing, nothing to look up", func(t *testing.T) {
		plan := &stubPlanSource{available: true, translate: false, resumable: true, cachedShare: 0.5}
		est := newTestEstimator(plan, false).Estimate(context.Background(), sixtyMinuteMovie())
		assert.Equal(t, 0.0, est.EstimatedUSD)
		assert.Empty(t, plan.creditAsk)
	})
	t.Run("speech recognition unavailable: the click would 503, no credit", func(t *testing.T) {
		plan := &stubPlanSource{available: false, translate: true, asrDone: 30}
		est := newTestEstimator(plan, false).Estimate(context.Background(), sixtyMinuteMovie())
		assert.Equal(t, 1.66, est.EstimatedUSD)
		assert.Empty(t, plan.creditAsk)
	})
	t.Run("extract lane: the pipeline translates the embedded track, not this cache", func(t *testing.T) {
		plan := &stubPlanSource{available: true, translate: true, asrDone: 30, cachedShare: 0.5}
		target := sixtyMinuteMovie()
		target.SubtitleTracksJSON = engTrackJSON
		est := routedEstimator(plan, &stubDurationPredictor{route: RouteExtract}).Estimate(context.Background(), target)
		require.Equal(t, TranscriptionPlanExtract, est.Plan)
		assert.Empty(t, plan.creditAsk)
		assert.Zero(t, est.ResumedASRMinutes)
	})
}

// ─── ASR half (real service, fake cache) ───────────────────────────────────

func chunkStoreFixture(t *testing.T) (svc *TranscriptionService, repo *fakeCacheRepo, media string, id ASRChunkIdentity) {
	t.Helper()
	media = filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))
	size, mtime, err := osFileIdentity(media)
	require.NoError(t, err)
	repo = &fakeCacheRepo{}
	svc = NewTranscriptionService(nil, &countingASR{}, nil, nil)
	svc.SetASRChunkStore(NewASRChunkStore(repo))
	id = ASRChunkIdentity{MediaID: uuidA, FileSize: size, FileMTime: mtime, TrackIndex: 1, Lang: "en",
		ChunkSeconds: 120, Endpoint: "|" + ai.WhisperModel}
	return svc, repo, media, id
}

func storeManifest(t *testing.T, repo *fakeCacheRepo, id ASRChunkIdentity, raw string) {
	t.Helper()
	k := asrManifestKey(id.MediaID, id.FileSize, id.FileMTime, id.Endpoint)
	require.NoError(t, repo.Set(context.Background(), k, raw, asrChunkType, time.Hour))
}

func storeChunkAt(t *testing.T, repo *fakeCacheRepo, id ASRChunkIdentity, start int) {
	t.Helper()
	id.Start = start
	require.NoError(t, repo.Set(context.Background(), asrChunkKey(id), `{"filtered":"x","unfiltered":"x"}`, asrChunkType, time.Hour))
}

func TestResumedASRMinutes(t *testing.T) {
	ctx := context.Background()

	t.Run("counts only the listed chunks that still exist", func(t *testing.T) {
		svc, repo, media, id := chunkStoreFixture(t)
		storeManifest(t, repo, id, `{"track":1,"lang":"en","chunk_seconds":120,"done":[0,120,240]}`)
		storeChunkAt(t, repo, id, 0)
		storeChunkAt(t, repo, id, 120)
		// chunk 240 expired: the manifest outlives it by up to one chunk's write.
		assert.Equal(t, 4.0, svc.resumedASRMinutes(ctx, uuidA, media, 60))
	})

	t.Run("capped at the runtime — the last chunk is shorter than the grid", func(t *testing.T) {
		svc, repo, media, id := chunkStoreFixture(t)
		storeManifest(t, repo, id, `{"track":1,"lang":"en","chunk_seconds":120,"done":[0,120,240]}`)
		for _, s := range []int{0, 120, 240} {
			storeChunkAt(t, repo, id, s)
		}
		assert.Equal(t, 5.0, svc.resumedASRMinutes(ctx, uuidA, media, 5))
	})

	t.Run("a short file is one chunk covering the whole audio", func(t *testing.T) {
		svc, repo, media, id := chunkStoreFixture(t)
		id.ChunkSeconds = 0
		storeManifest(t, repo, id, `{"track":1,"lang":"en","chunk_seconds":0,"done":[0]}`)
		storeChunkAt(t, repo, id, 0)
		assert.Equal(t, 7.5, svc.resumedASRMinutes(ctx, uuidA, media, 7.5))
	})

	t.Run("chunk keys come from the manifest's track and language", func(t *testing.T) {
		svc, repo, media, id := chunkStoreFixture(t)
		storeManifest(t, repo, id, `{"track":3,"lang":"ja","chunk_seconds":120,"done":[0]}`)
		storeChunkAt(t, repo, id, 0) // stored under track 1 / en — not what the manifest describes
		assert.Zero(t, svc.resumedASRMinutes(ctx, uuidA, media, 60))
		id.TrackIndex, id.Lang = 3, "ja"
		storeChunkAt(t, repo, id, 0)
		assert.Equal(t, 2.0, svc.resumedASRMinutes(ctx, uuidA, media, 60))
	})

	t.Run("anything unreadable is nothing done — the full price stands", func(t *testing.T) {
		svc, repo, media, id := chunkStoreFixture(t)
		assert.Zero(t, svc.resumedASRMinutes(ctx, uuidA, media, 60), "no manifest")

		storeManifest(t, repo, id, `not json`)
		assert.Zero(t, svc.resumedASRMinutes(ctx, uuidA, media, 60), "corrupt manifest")

		storeManifest(t, repo, id, `{"track":1,"lang":"en","chunk_seconds":120,"done":[0]}`)
		assert.Zero(t, svc.resumedASRMinutes(ctx, uuidA, media, 60), "manifest without its chunk")

		storeChunkAt(t, repo, id, 0)
		require.Equal(t, 2.0, svc.resumedASRMinutes(ctx, uuidA, media, 60))
		repo.manyErr = errors.New("database is locked")
		assert.Zero(t, svc.resumedASRMinutes(ctx, uuidA, media, 60), "read failure")
		repo.manyErr = nil

		assert.Zero(t, svc.resumedASRMinutes(ctx, uuidA, "/definitely/missing.mkv", 60), "file gone")
		other := NewTranscriptionService(nil, &countingASR{}, nil, nil)
		assert.Zero(t, other.resumedASRMinutes(ctx, uuidA, media, 60), "no store wired")
	})

	t.Run("a re-encoded file is a different file", func(t *testing.T) {
		svc, repo, media, id := chunkStoreFixture(t)
		storeManifest(t, repo, id, `{"track":1,"lang":"en","chunk_seconds":120,"done":[0]}`)
		storeChunkAt(t, repo, id, 0)
		require.NoError(t, os.WriteFile(media, []byte("re-encoded"), 0o600))
		assert.Zero(t, svc.resumedASRMinutes(ctx, uuidA, media, 60))
	})
}

// ─── Translation half (real service) ───────────────────────────────────────

// keyRecordingStore is a memoryStore that remembers every key it was asked for.
type keyRecordingStore struct {
	*memoryStore
	mu    sync.Mutex
	asked [][]string
}

func (s *keyRecordingStore) GetMany(ctx context.Context, keys []string) (map[string]string, error) {
	s.mu.Lock()
	s.asked = append(s.asked, append([]string(nil), keys...))
	s.mu.Unlock()
	return s.memoryStore.GetMany(ctx, keys)
}

func untranslatedFixture(t *testing.T, cues int, store SegmentStore) (*TranscriptionService, string) {
	t.Helper()
	dir := t.TempDir()
	enPath := filepath.Join(dir, "Movie.en.srt")
	require.NoError(t, os.WriteFile(enPath, []byte(multiCueSRT(cues)), 0o644))
	svc, _ := resumeWiredService(t, &countingCompleter{}, store)
	svc.SetSubtitleStateReader(&fakeStateReader{movie: untranslatedMovie(uuidB, enPath)})
	return svc, enPath
}

func TestCachedTranslationShare(t *testing.T) {
	ctx := context.Background()

	t.Run("share of cues cached under the run's own version", func(t *testing.T) {
		store := newMemoryStore()
		svc, _ := untranslatedFixture(t, 4, store)
		in := svc.translationInputsFor(ctx, models.SubtitleRunMediaMovie, uuidB)
		for _, line := range []string{"Line 1", "Line 3", "Line 4"} {
			require.NoError(t, store.Set(ctx, segkey.SegmentKey(line, in.version), "譯文", segkey.TTL))
		}
		assert.Equal(t, 0.75, svc.cachedTranslationShare(ctx, models.SubtitleRunMediaMovie, uuidB))
	})

	t.Run("cues cached for another model do not count", func(t *testing.T) {
		store := newMemoryStore()
		svc, _ := untranslatedFixture(t, 2, store)
		haiku := ai.WithModelID(ctx, "claude-haiku-4-5")
		in := svc.translationInputsFor(haiku, models.SubtitleRunMediaMovie, uuidB)
		for i := 1; i <= 2; i++ {
			require.NoError(t, store.Set(ctx, segkey.SegmentKey(fmt.Sprintf("Line %d", i), in.version), "譯文", segkey.TTL))
		}
		assert.Equal(t, 1.0, svc.cachedTranslationShare(haiku, models.SubtitleRunMediaMovie, uuidB))
		assert.Zero(t, svc.cachedTranslationShare(ctx, models.SubtitleRunMediaMovie, uuidB),
			"the default model would miss every one of them")
	})

	t.Run("anything unreadable is nothing cached", func(t *testing.T) {
		store := newMemoryStore()
		svc, enPath := untranslatedFixture(t, 2, store)
		store.getErr = errors.New("database is locked")
		assert.Zero(t, svc.cachedTranslationShare(ctx, models.SubtitleRunMediaMovie, uuidB), "read failure")
		store.getErr = nil

		require.NoError(t, os.WriteFile(enPath, nil, 0o644))
		assert.Zero(t, svc.cachedTranslationShare(ctx, models.SubtitleRunMediaMovie, uuidB), "empty SRT")

		require.NoError(t, os.Remove(enPath))
		assert.Zero(t, svc.cachedTranslationShare(ctx, models.SubtitleRunMediaMovie, uuidB), "SRT gone")

		noStore, _ := untranslatedFixture(t, 2, nil)
		assert.Zero(t, noStore.cachedTranslationShare(ctx, models.SubtitleRunMediaMovie, uuidB), "no store wired")
	})
}

// AC #4: the estimate looks up exactly the keys the translate run looks up —
// same file, same cues, same run version. If the two ever drift, the quote
// promises cues the run then pays for.
func TestCachedTranslationShare_LooksUpTheRunsOwnKeys(t *testing.T) {
	ctx := context.Background()
	estStore := &keyRecordingStore{memoryStore: newMemoryStore()}
	svc, enPath := untranslatedFixture(t, 12, estStore)
	svc.cachedTranslationShare(ctx, models.SubtitleRunMediaMovie, uuidB)
	require.Len(t, estStore.asked, 1, "one batched read")

	runStore := &keyRecordingStore{memoryStore: newMemoryStore()}
	run, _ := untranslatedFixture(t, 12, runStore)
	content, err := os.ReadFile(enPath)
	require.NoError(t, err)
	dir := filepath.Dir(enPath)
	_, _, err = run.translateAndPersist(ctx, "job-1", models.SubtitleRunMediaMovie, uuidB,
		string(content), enPath, filepath.Join(dir, "Movie.mkv"), dir, true)
	require.NoError(t, err)
	require.NotEmpty(t, runStore.asked)

	assert.Equal(t, runStore.asked[0], estStore.asked[0])
}
