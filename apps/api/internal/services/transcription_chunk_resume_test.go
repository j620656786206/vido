package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// ─── disc-2026-09-generation-resume-b-asr-chunk-store AC #3/#4 ─────────────
//
// A batch that hits the money ceiling at chunk 9 of 16 used to throw away the
// eight transcripts it had already paid for. Now each chunk's text is stored
// the moment it comes back, and the next run sends only the missing chunks.

// writePayloadWAV writes a 16 kHz mono WAV whose header claims `seconds`
// (32 KB/s) and whose file is just past the chunking limit — sparse, so a
// 16-chunk "film" costs no real disk. NeedsChunking keys on the file size,
// SplitAudioChunks on the header duration; both see a long file.
func writePayloadWAV(t *testing.T, path string, seconds int) {
	t.Helper()
	const byteRate = 32000
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(0))
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(16000))
	_ = binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(2))
	_ = binary.Write(&buf, binary.LittleEndian, uint16(16))
	buf.WriteString("data")
	_ = binary.Write(&buf, binary.LittleEndian, uint32(seconds*byteRate))
	f, err := os.Create(path)
	require.NoError(t, err)
	_, err = f.Write(buf.Bytes())
	require.NoError(t, err)
	require.NoError(t, f.Truncate(int64(ai.WhisperChunkTargetBytes)+4096))
	require.NoError(t, f.Close())
}

// installFakeChunkFFmpeg puts an `ffmpeg` on PATH that answers the chunk
// split (`-ss`) with a tiny header-only WAV — the chunk content is irrelevant,
// the fake ASR keys its answer on the call ORDER.
func installFakeChunkFFmpeg(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell shim not portable to windows")
	}
	tiny := filepath.Join(t.TempDir(), "tiny.wav")
	writeHeaderOnlyWAV(t, tiny, 600)
	bin := t.TempDir()
	script := fmt.Sprintf(`#!/bin/sh
out=""
for a in "$@"; do out="$a"; done
cp %q "$out"
`, tiny)
	require.NoError(t, os.WriteFile(filepath.Join(bin, "ffmpeg"), []byte(script), 0o755))
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// countingASR answers chunk N with a distinct cue and can fail at a given call.
type countingASR struct {
	mu     sync.Mutex
	calls  int
	failAt int // 1-based call index that returns ErrBudgetExceeded; 0 = never
	base   int // added to the call number in the cue text (labels a resumed run's chunks 9..16)
}

func (a *countingASR) Transcribe(ctx context.Context, p string) (string, error) {
	return a.TranscribeWithLanguage(ctx, p, "")
}

func (a *countingASR) TranscribeWithLanguage(ctx context.Context, _ string, _ string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	if a.failAt > 0 && a.calls == a.failAt {
		return "", ai.ErrBudgetExceeded
	}
	return fmt.Sprintf("1\n00:00:01,000 --> 00:00:02,000\ncall %d\n", a.base+a.calls), nil
}

func (a *countingASR) count() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.calls
}

func chunkScopeFor(id ASRChunkIdentity) *asrChunkScope {
	return &asrChunkScope{identity: id}
}

func TestTranscribeAudio_ResumesFromStoredChunks(t *testing.T) {
	installFakeChunkFFmpeg(t)
	wav := filepath.Join(t.TempDir(), "film.wav")
	writePayloadWAV(t, wav, 16*600) // 16 chunks of 600 s
	repo := &fakeCacheRepo{}

	// First run: the ceiling trips at chunk 9.
	first := &countingASR{failAt: 9}
	svc := NewTranscriptionService(nil, first, nil, nil)
	svc.SetASRChunkStore(NewASRChunkStore(repo))
	id := baseIdentity()
	_, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
	require.Error(t, err)
	assert.ErrorIs(t, err, ai.ErrBudgetExceeded)
	assert.Equal(t, 9, first.count())

	chunkRows := 0
	var manifest asrManifest
	for k, e := range repo.entries {
		switch {
		case e.Type == asrChunkType && k == asrManifestKey(id.MediaID, id.FileSize, id.FileMTime, id.Endpoint):
			m, ok := decodeASRManifest(e.Value)
			require.True(t, ok)
			manifest = m
		case e.Type == asrChunkType:
			chunkRows++
		}
	}
	assert.Equal(t, 8, chunkRows, "the eight paid-for chunks are remembered")
	assert.Equal(t, asrManifest{Track: 1, Lang: "en", ChunkSeconds: 600, Done: []int{0, 600, 1200, 1800, 2400, 3000, 3600, 4200}}, manifest)

	// Second run: only the eight missing chunks reach the engine.
	second := &countingASR{base: 8}
	svc2 := NewTranscriptionService(nil, second, nil, nil)
	svc2.SetASRChunkStore(NewASRChunkStore(repo))
	got, err := svc2.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
	require.NoError(t, err)
	assert.Equal(t, 8, second.count(), "chunks 1–8 came from the store")

	// …and the merged transcript is byte-identical to an uninterrupted run
	// (chunk-local timestamps re-offset at merge time, in source order).
	third := &countingASR{}
	svc3 := NewTranscriptionService(nil, third, nil, nil)
	want, err := svc3.transcribeAudio(context.Background(), wav, "en", nil)
	require.NoError(t, err)
	assert.Equal(t, 16, third.count())
	assert.Equal(t, want, got)
	assert.Equal(t, 16, countSRTCues(got))
	assert.Contains(t, got, "02:20:01,000 --> 02:20:02,000\ncall 15\n", "chunk 15 (offset 8400 s) keeps its merged timestamp")

	// Success clears the film's rows.
	svc2.clearChunkCache(context.Background(), chunkScopeFor(id))
	assert.Equal(t, 8+8+1, len(repo.deleted), "16 chunks + the manifest")
}

func TestTranscribeAudio_IdentityChangeMeansMiss(t *testing.T) {
	installFakeChunkFFmpeg(t)
	wav := filepath.Join(t.TempDir(), "film.wav")
	writePayloadWAV(t, wav, 2*600)
	repo := &fakeCacheRepo{}
	id := baseIdentity()

	seed := &countingASR{}
	svc := NewTranscriptionService(nil, seed, nil, nil)
	svc.SetASRChunkStore(NewASRChunkStore(repo))
	_, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
	require.NoError(t, err)
	require.Equal(t, 2, seed.count())

	for name, mutate := range map[string]func(*ASRChunkIdentity){
		"file rewritten (mtime)": func(i *ASRChunkIdentity) { i.FileMTime++ },
		"other engine":           func(i *ASRChunkIdentity) { i.Endpoint = "https://groq/v1|whisper-large-v3" },
		"other language":         func(i *ASRChunkIdentity) { i.Lang = "ja" },
	} {
		changed := baseIdentity()
		mutate(&changed)
		asr := &countingASR{}
		s := NewTranscriptionService(nil, asr, nil, nil)
		s.SetASRChunkStore(NewASRChunkStore(repo))
		_, err := s.transcribeAudio(context.Background(), wav, changed.Lang, chunkScopeFor(changed))
		require.NoError(t, err, name)
		assert.Equalf(t, 2, asr.count(), "%s → every chunk is a miss", name)
	}

	same := &countingASR{}
	s := NewTranscriptionService(nil, same, nil, nil)
	s.SetASRChunkStore(NewASRChunkStore(repo))
	_, err = s.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
	require.NoError(t, err)
	assert.Equal(t, 0, same.count(), "the unchanged identity is served entirely from the store")
}

func TestTranscribeAudio_CacheFailuresNeverFailThePaidRun(t *testing.T) {
	installFakeChunkFFmpeg(t)
	wav := filepath.Join(t.TempDir(), "film.wav")
	writePayloadWAV(t, wav, 2*600)
	id := baseIdentity()

	t.Run("GetMany fails → everything is transcribed", func(t *testing.T) {
		repo := &fakeCacheRepo{manyErr: errors.New("db locked")}
		asr := &countingASR{}
		svc := NewTranscriptionService(nil, asr, nil, nil)
		svc.SetASRChunkStore(NewASRChunkStore(repo))
		_, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
		require.NoError(t, err)
		assert.Equal(t, 2, asr.count())
	})

	t.Run("Set fails → the run still succeeds", func(t *testing.T) {
		repo := &fakeCacheRepo{setErr: errors.New("disk full")}
		asr := &countingASR{}
		svc := NewTranscriptionService(nil, asr, nil, nil)
		svc.SetASRChunkStore(NewASRChunkStore(repo))
		_, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
		require.NoError(t, err)
		assert.Equal(t, 2, asr.count())
	})

	t.Run("a corrupt entry is a miss", func(t *testing.T) {
		repo := &fakeCacheRepo{}
		k := asrChunkKey(ASRChunkIdentity{MediaID: id.MediaID, FileSize: id.FileSize, FileMTime: id.FileMTime,
			TrackIndex: id.TrackIndex, Lang: id.Lang, Start: 0, ChunkSeconds: 600, Endpoint: id.Endpoint})
		require.NoError(t, repo.Set(context.Background(), k, "not json", asrChunkType, time.Hour))
		asr := &countingASR{}
		svc := NewTranscriptionService(nil, asr, nil, nil)
		svc.SetASRChunkStore(NewASRChunkStore(repo))
		_, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
		require.NoError(t, err)
		assert.Equal(t, 2, asr.count(), "the corrupt chunk was transcribed again")
	})

	t.Run("no store, no scope → the pre-story path, byte for byte", func(t *testing.T) {
		asr := &countingASR{}
		svc := NewTranscriptionService(nil, asr, nil, nil)
		_, err := svc.transcribeAudio(context.Background(), wav, "en", nil)
		require.NoError(t, err)
		assert.Equal(t, 2, asr.count())
	})
}

// The stored `unfiltered` text feeds the whole-file hallucination guard: a
// filter that emptied every chunk still delivers the unfiltered transcript.
func TestTranscribeAudio_StoredUnfilteredFeedsTheGuard(t *testing.T) {
	installFakeChunkFFmpeg(t)
	wav := filepath.Join(t.TempDir(), "film.wav")
	writePayloadWAV(t, wav, 2*600)
	id := baseIdentity()
	repo := &fakeCacheRepo{}
	for _, start := range []int{0, 600} {
		k := asrChunkKey(ASRChunkIdentity{MediaID: id.MediaID, FileSize: id.FileSize, FileMTime: id.FileMTime,
			TrackIndex: id.TrackIndex, Lang: id.Lang, Start: start, ChunkSeconds: 600, Endpoint: id.Endpoint})
		raw, _ := json.Marshal(asrChunkValue{Filtered: "", Unfiltered: fmt.Sprintf("1\n00:00:01,000 --> 00:00:02,000\nkept %d\n", start)})
		require.NoError(t, repo.Set(context.Background(), k, string(raw), asrChunkType, time.Hour))
	}
	asr := &countingASR{}
	svc := NewTranscriptionService(nil, asr, nil, nil)
	svc.SetASRChunkStore(NewASRChunkStore(repo))

	got, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))

	require.NoError(t, err)
	assert.Equal(t, 0, asr.count())
	assert.Contains(t, got, "kept 0")
	assert.Contains(t, got, "kept 600")
}

// A short file (no chunking) goes through the same store as ONE chunk.
func TestTranscribeAudio_ShortFileIsOneChunk(t *testing.T) {
	wav := filepath.Join(t.TempDir(), "short.wav")
	writeHeaderOnlyWAV(t, wav, 30)
	id := baseIdentity()
	repo := &fakeCacheRepo{}

	first := &countingASR{}
	svc := NewTranscriptionService(nil, first, nil, nil)
	svc.SetASRChunkStore(NewASRChunkStore(repo))
	_, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
	require.NoError(t, err)
	require.Equal(t, 1, first.count())

	second := &countingASR{}
	svc2 := NewTranscriptionService(nil, second, nil, nil)
	svc2.SetASRChunkStore(NewASRChunkStore(repo))
	_, err = svc2.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(id))
	require.NoError(t, err)
	assert.Equal(t, 0, second.count())
}

// 600 films stuck at once: a film's lookup still asks for ITS keys only.
func TestTranscribeAudio_LookupIsBoundedByTheFilmNotTheTable(t *testing.T) {
	installFakeChunkFFmpeg(t)
	wav := filepath.Join(t.TempDir(), "film.wav")
	writePayloadWAV(t, wav, 2*600)
	repo := &fakeCacheRepo{}
	for i := 0; i < 10_000; i++ {
		require.NoError(t, repo.Set(context.Background(), fmt.Sprintf("%sfiller-%d", asrChunkKeyPrefix, i), "{}", asrChunkType, time.Hour))
	}
	asr := &countingASR{}
	svc := NewTranscriptionService(nil, asr, nil, nil)
	svc.SetASRChunkStore(NewASRChunkStore(repo))

	_, err := svc.transcribeAudio(context.Background(), wav, "en", chunkScopeFor(baseIdentity()))

	require.NoError(t, err)
	assert.Equal(t, 1, repo.manyCalls, "one batched read per run — no per-chunk manifest round trips")
	assert.Equal(t, 2+1, repo.lastManyKeys, "two chunks + the manifest → three keys, however large the table")
}

// The run passes the film's identity down and clears the store once the
// English SRT is on disk — the end-to-end seam of AC #3.
func TestRunTranscription_StoresChunksAndClearsThemOnSuccess(t *testing.T) {
	repo := &fakeCacheRepo{}
	svc, _ := runBudgetService(t, 30, &countingASR{}, time.Minute, time.Second)
	svc.SetASRChunkStore(NewASRChunkStore(repo))
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	err := svc.RunTranscription(context.Background(), uuidEight, media, filepath.Dir(media))

	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(repo.sets), 2, "one chunk + its manifest were stored during the run")
	assert.Equal(t, len(repo.sets), len(repo.deleted), "…and every stored key was deleted after the SRT was written")
}

// AC #6 seam: the batch asks whether a film has progress worth finishing first.
func TestHasResumeProgress(t *testing.T) {
	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))
	size, mtime, err := osFileIdentity(media)
	require.NoError(t, err)

	t.Run("a manifest counts", func(t *testing.T) {
		repo := &fakeCacheRepo{}
		svc := NewTranscriptionService(nil, &countingASR{}, nil, nil)
		svc.SetASRChunkStore(NewASRChunkStore(repo))
		assert.False(t, svc.HasResumeProgress(context.Background(), models.SubtitleRunMediaMovie, uuidA, media))
		k := asrManifestKey(uuidA, size, mtime, "|"+ai.WhisperModel)
		require.NoError(t, repo.Set(context.Background(), k, `{"track":1,"lang":"en","chunk_seconds":600,"done":[0]}`, asrChunkType, time.Hour))
		assert.True(t, svc.HasResumeProgress(context.Background(), models.SubtitleRunMediaMovie, uuidA, media))
	})

	t.Run("an untranslated row counts", func(t *testing.T) {
		svc := NewTranscriptionService(nil, &countingASR{}, nil, nil)
		svc.SetSubtitleStateReader(&fakeStateReader{movie: untranslatedMovie(uuidA, media)})
		assert.True(t, svc.HasResumeProgress(context.Background(), models.SubtitleRunMediaMovie, uuidA, media))
	})

	t.Run("nothing → false, and a missing file is not an error", func(t *testing.T) {
		svc := NewTranscriptionService(nil, &countingASR{}, nil, nil)
		svc.SetASRChunkStore(NewASRChunkStore(&fakeCacheRepo{}))
		assert.False(t, svc.HasResumeProgress(context.Background(), models.SubtitleRunMediaMovie, uuidA, "/definitely/missing.mkv"))
	})
}
