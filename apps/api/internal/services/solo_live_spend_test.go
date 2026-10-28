package services

// disc-2026-10-solo-live-spend — the ASR leg of a pipeline-mode click reads
// the click's Budget from the SoloSpendBoard for its transcription_* events,
// without the click being counted twice anywhere.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
)

func TestCostFields_ASRLegOfASoloClickReadsTheBoard(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	board := NewSoloSpendBoard()
	svc.SetSoloSpendBoard(board)

	_, err := svc.acquireJob("ep-asr-leg", "", false) // the ASR leg: Solo=false
	require.NoError(t, err)
	_, err = svc.acquireJob("batch-item", "", false)
	require.NoError(t, err)

	click := ai.NewBudget(5)
	board.Track("ep-asr-leg", click)

	first := map[string]interface{}{"media_id": "ep-asr-leg", "phase": "transcribing"}
	svc.costFields(first)
	assert.Equal(t, 0.0, first["spent_usd"], "before anything is paid: a recorded 0")
	assert.Equal(t, 5.0, first["budget_usd"])

	click.RecordASRWithRate(60, 0.42)
	second := map[string]interface{}{"media_id": "ep-asr-leg", "phase": "transcribing"}
	svc.costFields(second)
	assert.InDelta(t, 0.42, second["spent_usd"], 1e-9, "the next event reads the meter again")

	item := map[string]interface{}{"media_id": "batch-item"}
	svc.costFields(item)
	assert.NotContains(t, item, "spent_usd", "a batch item is not on the board — absent, never 0")
	assert.NotContains(t, item, "budget_usd")
}

func TestCostFields_ARunsOwnBudgetWinsOverTheBoard(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	board := NewSoloSpendBoard()
	svc.SetSoloSpendBoard(board)
	_, err := svc.acquireJob("m", "片", true)
	require.NoError(t, err)
	own := ai.NewBudget(2)
	own.RecordASRWithRate(60, 0.5)
	svc.trackRunBudget("m", own)
	board.Track("m", ai.NewBudget(9)) // cannot happen (single flight) — precedence only

	data := map[string]interface{}{"media_id": "m"}
	svc.costFields(data)
	assert.InDelta(t, 0.5, data["spent_usd"], 1e-9)
	assert.Equal(t, 2.0, data["budget_usd"])
}

// AC #4: the board figure feeds the events only — /ai/usage and the Activity
// list do not see the ASR leg (the SoloRunner reports the click once).
func TestASRLegIsNotCountedAgain(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	board := NewSoloSpendBoard()
	svc.SetSoloSpendBoard(board)
	_, err := svc.acquireJob("ep-asr-leg", "", false)
	require.NoError(t, err)
	click := ai.NewBudget(5)
	click.RecordASRWithRate(60, 0.42)
	board.Track("ep-asr-leg", click)

	_, _, ok := svc.ActiveManualUsage()
	assert.False(t, ok, "the ASR leg owns no Budget — the SoloRunner counts the click")
	assert.Empty(t, svc.ActiveSoloJobs(), "the ASR leg is not a row of its own")
}

func TestActiveSoloJobs_LegacyJobCarriesItsSpendOnceItHasAMeter(t *testing.T) {
	svc := NewTranscriptionService(nil, nil, nil, nil)
	_, err := svc.acquireJob("m", "片", true)
	require.NoError(t, err)

	snaps := svc.ActiveSoloJobs()
	require.Len(t, snaps, 1)
	assert.Nil(t, snaps[0].SpentUSD, "no Budget attached yet → no figure, not 0")

	b := ai.NewBudget(2)
	b.RecordASRWithRate(60, 0.5)
	svc.trackRunBudget("m", b)
	snaps = svc.ActiveSoloJobs()
	require.Len(t, snaps, 1)
	require.NotNil(t, snaps[0].SpentUSD)
	assert.InDelta(t, 0.5, *snaps[0].SpentUSD, 1e-9)
}

// spendingASR is a hosted provider as far as the meter is concerned: every
// call records one minute on the ctx Budget (what whisper.go / scribe.go do).
type spendingASR struct {
	mu    sync.Mutex
	calls int
	srt   string
}

func (a *spendingASR) Transcribe(ctx context.Context, audioPath string) (string, error) {
	return a.TranscribeWithLanguage(ctx, audioPath, "")
}

func (a *spendingASR) TranscribeWithLanguage(ctx context.Context, _, _ string) (string, error) {
	a.mu.Lock()
	a.calls++
	a.mu.Unlock()
	ai.BudgetFromContext(ctx).RecordASRWithRate(60, 0.1)
	return a.srt, nil
}

// AC #1 end to end: the ASR leg of a solo click (Solo=false, the click's
// Budget on the ctx and on the board) sends transcription_progress events
// whose spent_usd climbs chunk by chunk to the click's final figure.
func TestRunTranscription_ASRLegEventsCarryTheClicksLiveSpend(t *testing.T) {
	asr := &spendingASR{srt: genTestSRT}
	// 6 minutes of audio on the 120 s grid: three paid chunks.
	svc, client := runBudgetService(t, 360, asr, time.Minute, time.Second)
	board := NewSoloSpendBoard()
	svc.SetSoloSpendBoard(board)
	click := ai.NewBudget(5)
	board.Track(uuidEight, click)

	media := filepath.Join(t.TempDir(), "m.mkv")
	require.NoError(t, os.WriteFile(media, []byte("x"), 0o600))

	err := svc.RunTranscription(ai.WithBudget(context.Background(), click), uuidEight, media, filepath.Dir(media))
	require.NoError(t, err)
	require.GreaterOrEqual(t, asr.calls, 2, "the fixture must produce several paid chunks")

	var spends []float64
	drain := time.After(2 * time.Second)
loop:
	for {
		select {
		case ev := <-client.Events:
			if !carriesRunCost(ev.Type) {
				continue
			}
			data := ev.Data.(map[string]interface{})
			if data["media_id"] != uuidEight {
				continue
			}
			spent, ok := data["spent_usd"].(float64)
			require.True(t, ok, "%s of the ASR leg must carry spent_usd", ev.Type)
			assert.Equal(t, 5.0, data["budget_usd"], string(ev.Type))
			spends = append(spends, spent)
			if ev.Type == EventTranscriptionComplete || ev.Type == EventTranscriptionFailed {
				break loop
			}
		case <-drain:
			break loop
		}
	}
	require.GreaterOrEqual(t, len(spends), 3)
	assert.Equal(t, 0.0, spends[0], "the extracting event comes before anything is paid")
	for i := 1; i < len(spends); i++ {
		assert.GreaterOrEqual(t, spends[i], spends[i-1], "spend never goes down between events")
	}
	assert.InDelta(t, click.SpentUSD(), spends[len(spends)-1], 1e-9, "the last event shows the click's whole spend — once")
	assert.Greater(t, spends[len(spends)-1], spends[0])

	_, _, ok := svc.ActiveManualUsage()
	assert.False(t, ok)
}

// AC #5 [@contract-v1]: transcription rows carry spent_usd — raw, 0 sent as 0,
// queued rows too, absent when the job has no meter; the batch row never.
func TestActivity_SoloRowCarriesTheJobsLiveSpend(t *testing.T) {
	paid, zero, queuedPaid := 0.123456789, 0.0, 0.31
	svc := activityWith(fakeBatch{active: true, current: 1, total: 2, item: "batch"}, fakeSoloJobs{
		{MediaID: "ep-2", Title: lioness(2), StartedAt: soloT0, SpentUSD: &paid},
		{MediaID: "ep-3", Title: lioness(3), StartedAt: soloT0.Add(time.Second), SpentUSD: &zero},
		{MediaID: "ep-4", Title: lioness(4), StartedAt: soloT0.Add(2 * time.Second)}, // no meter yet
		{MediaID: "ep-5", Title: lioness(5), StartedAt: soloT0, Queued: true, QueuedAt: soloT0, SpentUSD: &queuedPaid},
	})

	blob, err := json.Marshal(svc.GetActivity(context.Background()).ActiveJobs.Jobs)
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"kind":"generation_batch","percent_done":0,"detail":"batch","current":1,"total":2},
		{"kind":"transcription","percent_done":0,"detail":"特種部隊：母獅 S01E02","media_id":"ep-2","spent_usd":0.123456789},
		{"kind":"transcription","percent_done":0,"detail":"特種部隊：母獅 S01E03","media_id":"ep-3","spent_usd":0},
		{"kind":"transcription","percent_done":0,"detail":"特種部隊：母獅 S01E04","media_id":"ep-4"},
		{"kind":"transcription","percent_done":0,"detail":"特種部隊：母獅 S01E05","media_id":"ep-5","queued":true,"spent_usd":0.31}
	]`, string(blob))
}
