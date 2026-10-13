package services

// disc-2026-09-generation-resume-c2 — the consent list takes off speech
// recognition an earlier, unfinished run already paid for.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
)

// resumeLibrary is three real files on disk (the credit reads their size and
// mtime) and a transcription service whose chunk store the test fills.
func resumeLibrary(t *testing.T) (svc *TranscriptionService, repo *fakeCacheRepo, paths map[string]string) {
	t.Helper()
	dir := t.TempDir()
	paths = map[string]string{}
	for _, id := range []string{uuidA, uuidB, uuidC} {
		p := filepath.Join(dir, id+".mkv")
		require.NoError(t, os.WriteFile(p, []byte(id), 0o600))
		paths[id] = p
	}
	repo = &fakeCacheRepo{}
	svc = NewTranscriptionService(nil, &countingASR{}, nil, nil)
	svc.SetASRChunkStore(NewASRChunkStore(repo))
	return svc, repo, paths
}

// storeDone records a manifest for mediaID listing starts, and stores each chunk.
func storeDone(t *testing.T, repo *fakeCacheRepo, mediaID, path string, starts ...int) {
	t.Helper()
	size, mtime, err := osFileIdentity(path)
	require.NoError(t, err)
	id := ASRChunkIdentity{MediaID: mediaID, FileSize: size, FileMTime: mtime, TrackIndex: 1, Lang: "en",
		ChunkSeconds: 120, Endpoint: "|" + ai.WhisperModel}
	done := make([]string, len(starts))
	for i, s := range starts {
		done[i] = strconv.Itoa(s)
		storeChunkAt(t, repo, id, s)
	}
	storeManifest(t, repo, id, `{"track":1,"lang":"en","chunk_seconds":120,"done":[`+strings.Join(done, ",")+`]}`)
}

func candidateByID(t *testing.T, res *GenerationCandidateResult, id string) GenerationCandidate {
	t.Helper()
	for _, c := range res.Candidates {
		if c.MediaID == id {
			return c
		}
	}
	t.Fatalf("no candidate %s", id)
	return GenerationCandidate{}
}

func TestAnalyze_StoredTranscriptionComesOffTheASRQuote(t *testing.T) {
	ts, repo, paths := resumeLibrary(t)
	storeDone(t, repo, uuidA, paths[uuidA], 0, 120) // 4 minutes done
	// An extract row with a manifest (it once ran through speech recognition)
	// is not charged for speech recognition, so there is nothing to take off.
	storeDone(t, repo, uuidC, paths[uuidC], 0)

	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow(uuidA, "A", paths[uuidA], 60, ""),
		movieRow(uuidB, "B", paths[uuidB], 60, ""),
		movieRow(uuidC, "C", paths[uuidC], 60, engTrackJSON),
	}}
	svc := NewGenerationCandidateService(movies, nil, nil,
		&stubPredictor{fromTracks: RouteExtract, probeRoute: RouteASR}, false, 0, nil)
	svc.SetResumeCredit(ts)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	rate := ai.EstimatedASRPerMinute(false)
	model := ai.DefaultClaudeModel
	a, b, c := candidateByID(t, res, uuidA), candidateByID(t, res, uuidB), candidateByID(t, res, uuidC)

	assert.Equal(t, 4.0, a.ResumedASRMinutes)
	assert.Equal(t, estimateUSDWithCredit(RouteASR, 60, 4, rate, model).InexactFloat64(), a.EstimatedUSD)
	assert.Less(t, a.EstimatedUSD, b.EstimatedUSD, "same length, A has 4 minutes already transcribed")
	assert.Equal(t, 60.0, a.RuntimeMinutes, "runtime_minutes still means the whole film")

	assert.Zero(t, b.ResumedASRMinutes)
	assert.Equal(t, estimateUSD(RouteASR, 60, rate, model).InexactFloat64(), b.EstimatedUSD)
	assert.Zero(t, c.ResumedASRMinutes, "an extract row is never credited")
	assert.Equal(t, estimateUSD(RouteExtract, 60, rate, model).InexactFloat64(), c.EstimatedUSD)

	// The rows still add up to every total.
	sum := decimal.Zero
	for _, cand := range res.Candidates {
		sum = sum.Add(decimal.NewFromFloat(cand.EstimatedUSD))
	}
	assert.Equal(t, sum.InexactFloat64(), res.Summary.EstimatedTotalUSD)
	est := res.EstimatesByModel[model]
	assert.Equal(t, sum.InexactFloat64(), est.TotalUSD)
	assert.Equal(t, a.EstimatedUSD, est.PerCandidate[uuidA])
}

func TestAnalyze_ResumeCreditIsTwoReadsForTheWholeList(t *testing.T) {
	ts, repo, paths := resumeLibrary(t)
	for _, id := range []string{uuidA, uuidB, uuidC} {
		storeDone(t, repo, id, paths[id], 0)
	}
	movies := &stubMovieFinder{movies: []models.Movie{
		movieRow(uuidA, "A", paths[uuidA], 60, ""),
		movieRow(uuidB, "B", paths[uuidB], 60, ""),
		movieRow(uuidC, "C", paths[uuidC], 60, ""),
	}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	svc.SetResumeCredit(ts)
	repo.manyCalls = 0

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)

	assert.Equal(t, 2, repo.manyCalls, "every manifest in one read, every chunk in one more")
	for _, c := range res.Candidates {
		assert.Equal(t, 2.0, c.ResumedASRMinutes, c.MediaID)
	}
}

func TestAnalyze_ResumeCreditFailureQuotesTheFullPrice(t *testing.T) {
	ts, repo, paths := resumeLibrary(t)
	storeDone(t, repo, uuidA, paths[uuidA], 0, 120)
	repo.manyErr = errors.New("database is locked")
	movies := &stubMovieFinder{movies: []models.Movie{movieRow(uuidA, "A", paths[uuidA], 60, "")}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	svc.SetResumeCredit(ts)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err, "a cache hiccup must not cost the user the whole list")
	a := candidateByID(t, res, uuidA)
	assert.Zero(t, a.ResumedASRMinutes)
	assert.Equal(t, estimateUSD(RouteASR, 60, ai.EstimatedASRPerMinute(false), ai.DefaultClaudeModel).InexactFloat64(), a.EstimatedUSD)
}

func TestAnalyze_NoCreditSourceIsTheOldQuote(t *testing.T) {
	_, repo, paths := resumeLibrary(t)
	storeDone(t, repo, uuidA, paths[uuidA], 0, 120)
	movies := &stubMovieFinder{movies: []models.Movie{movieRow(uuidA, "A", paths[uuidA], 60, "")}}
	svc := NewGenerationCandidateService(movies, nil, nil, &stubPredictor{probeRoute: RouteASR}, false, 0, nil)
	svc.SetResumeCredit(nil)

	res, err := svc.Analyze(context.Background(), nil)
	require.NoError(t, err)
	a := candidateByID(t, res, uuidA)
	assert.Zero(t, a.ResumedASRMinutes)
	assert.Equal(t, estimateUSD(RouteASR, 60, ai.EstimatedASRPerMinute(false), ai.DefaultClaudeModel).InexactFloat64(), a.EstimatedUSD)
}

func TestEstimateUSDWithCredit(t *testing.T) {
	rate := ai.EstimatedASRPerMinute(false)
	model := ai.DefaultClaudeModel
	full := estimateUSD(RouteASR, 60, rate, model)
	assert.True(t, estimateUSDWithCredit(RouteASR, 60, 0, rate, model).Equal(full), "no credit is the old price")
	assert.True(t, estimateUSDWithCredit(RouteExtract, 60, 30, rate, model).Equal(estimateUSD(RouteExtract, 60, rate, model)),
		"credit only ever touches the ASR route")
	// All 60 minutes transcribed: only the translation half is left.
	all := estimateUSDWithCredit(RouteASR, 60, 60, rate, model)
	assert.True(t, all.Equal(roundUSD(decimal.NewFromInt(60).Mul(asrLegTranslationRatePerMinute(model)))))
	assert.True(t, estimateUSDWithCredit(RouteASR, 60, 90, rate, model).Equal(all), "never below zero audio")
}
