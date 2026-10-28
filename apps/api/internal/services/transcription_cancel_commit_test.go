package services

// disc-2026-10-solo-generation-cancel-api CR M1/L2: once the paid work is on
// disk, a stop landing in the bookkeeping that follows must not undo it.

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/models"
)

// ctxCheckingWriter fails on a cancelled ctx, as database/sql's ExecContext does.
type ctxCheckingWriter struct{ fakeSubtitleWriter }

func (w *ctxCheckingWriter) UpdateSubtitleStatus(ctx context.Context, id string, status models.SubtitleStatus, path, language string, score float64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return w.fakeSubtitleWriter.UpdateSubtitleStatus(ctx, id, status, path, language, score)
}

// cancellingPlacer places the zh-Hant sidecar and then the user's stop lands.
type cancellingPlacer struct {
	fakePlacer
	cancel context.CancelFunc
}

func (p *cancellingPlacer) PlaceSubtitle(mediaFilePath string, data []byte, language, format string) (string, error) {
	path, err := p.fakePlacer.PlaceSubtitle(mediaFilePath, data, language, format)
	p.cancel()
	return path, err
}

// CR M1: the sidecar is placed, then the stop arrives — the verdict is still
// written and the leg reports success, so the run is recorded as completed.
func TestTranslateAndPersist_StopAfterThePlaceStillWritesTheVerdict(t *testing.T) {
	writer := &ctxCheckingWriter{}
	svc := newWriterWiredService(t, &translationIntegrationMock{response: "[1] 你好世界"}, writer)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc.SetPlacer(&cancellingPlacer{cancel: cancel})

	tmpDir := t.TempDir()
	zhPath, _, err := svc.translateAndPersist(ctx, "job-1", models.SubtitleRunMediaMovie, uuidB, genTestSRT,
		filepath.Join(tmpDir, "Movie.en.srt"), filepath.Join(tmpDir, "Movie.mkv"), tmpDir, true)

	require.NoError(t, err, "the subtitle is on disk — this leg succeeded")
	require.Error(t, ctx.Err(), "the stop did land")
	calls := writer.snapshot()
	require.Len(t, calls, 1)
	assert.Equal(t, models.SubtitleStatusFound, calls[0].Status)
	assert.Equal(t, zhPath, calls[0].Path)
}

// ctxCheckingChunkStore refuses writes on a cancelled ctx.
type ctxCheckingChunkStore struct {
	mu   sync.Mutex
	sets map[string]string
}

func (s *ctxCheckingChunkStore) GetMany(context.Context, []string) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *ctxCheckingChunkStore) Set(ctx context.Context, key, value string, _ time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sets[key] = value
	return nil
}

func (s *ctxCheckingChunkStore) Delete(context.Context, string) error { return nil }

// CR L2: a chunk that came back just as the stop landed is kept — the user
// paid for it, and the next run must not pay again.
func TestStoreChunk_SurvivesTheStopThatLandedAfterThePaidCall(t *testing.T) {
	store := &ctxCheckingChunkStore{sets: map[string]string{}}
	svc := NewTranscriptionService(nil, nil, nil, nil)
	svc.SetASRChunkStore(store)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	scope := &asrChunkScope{identity: ASRChunkIdentity{MediaID: "m"}, keys: []string{"chunk-0"}}
	svc.storeChunk(ctx, scope, 0, 600, asrChunkValue{Filtered: "1\n00:00:01,000 --> 00:00:02,000\nHi\n"})

	store.mu.Lock()
	defer store.mu.Unlock()
	assert.Contains(t, store.sets, "chunk-0", "the paid chunk is stored")
	assert.Contains(t, store.sets, scope.manifestKey(), "and named in the manifest")
}
