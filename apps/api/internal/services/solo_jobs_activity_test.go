package services

// disc-2026-10-activity-solo-jobs-collapsed: eight 生成字幕 clicks read as ONE
// Activity row (「字幕生成中 · 特種部隊：母獅 S01E01」, badge 1) while eight runs
// were going — three working, five waiting for the extraction slot. These pin
// one row per job, the 排隊中 signal, and the row order.

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var soloT0 = time.Date(2026, 10, 28, 21, 30, 0, 0, time.UTC)

func lioness(ep int) string { return fmt.Sprintf("特種部隊：母獅 S01E%02d", ep) }

// The NAS evidence, as the two sources would hand it over: clicked E01…E08 a
// few seconds apart; E01–E03 got going, E04–E08 wait for the slot.
func eightClicks() fakeSoloJobs {
	var jobs fakeSoloJobs
	for ep := 8; ep >= 1; ep-- { // map order is arbitrary — hand them over scrambled
		snap := SoloJobSnapshot{
			MediaID:   fmt.Sprintf("ep-%d", ep),
			Title:     lioness(ep),
			StartedAt: soloT0.Add(time.Duration(ep) * 5 * time.Second),
		}
		if ep >= 4 {
			snap.Queued = true
			snap.QueuedAt = snap.StartedAt.Add(time.Second)
		}
		jobs = append(jobs, snap)
	}
	return jobs
}

func TestActivity_EightSoloClicksAreEightRows(t *testing.T) {
	svc := activityWith(fakeBatch{}, eightClicks())
	svc.SetStepReader(fakeSteps{
		"ep-1": {Phase: "translating", Current: 120, Total: 659},
		"ep-2": {Phase: "transcribing", Current: 3, Total: 22},
		"ep-3": {Phase: "transcribing"},
		// the tracker still remembers `extracting` for the waiting ones
		"ep-4": {Phase: "extracting"},
		"ep-5": {Phase: "extracting"},
	})

	jobs := svc.GetActivity(context.Background()).ActiveJobs.Jobs
	require.Len(t, jobs, 8, "one row per click — the badge counts rows")

	var titles []string
	for _, j := range jobs {
		assert.Equal(t, "transcription", j.Kind)
		titles = append(titles, j.Detail)
	}
	assert.Equal(t, []string{lioness(1), lioness(2), lioness(3), lioness(4), lioness(5), lioness(6), lioness(7), lioness(8)}, titles)

	// Running rows carry their OWN step.
	assert.Equal(t, &ActiveJobStep{Phase: "translating", Current: 120, Total: 659}, jobs[0].Step)
	assert.Equal(t, 18, jobs[0].PercentDone)
	assert.Equal(t, &ActiveJobStep{Phase: "transcribing", Current: 3, Total: 22}, jobs[1].Step)
	assert.Equal(t, 13, jobs[1].PercentDone)
	assert.Equal(t, &ActiveJobStep{Phase: "transcribing"}, jobs[2].Step)
	assert.Zero(t, jobs[2].PercentDone)
	for _, j := range jobs[:3] {
		assert.False(t, j.Queued)
	}
	// Queued rows: no step (not 提取音訊 while nothing extracts), no percent.
	for _, j := range jobs[3:] {
		assert.True(t, j.Queued, j.Detail)
		assert.Nil(t, j.Step, j.Detail)
		assert.Zero(t, j.PercentDone, j.Detail)
	}
}

func TestActivity_SoloRowOrder(t *testing.T) {
	jobs := fakeSoloJobs{
		// queued first in the input, but waiting the shortest
		{MediaID: "q-late", StartedAt: soloT0, Queued: true, QueuedAt: soloT0.Add(30 * time.Second)},
		{MediaID: "run-new", StartedAt: soloT0.Add(20 * time.Second)},
		{MediaID: "q-early", StartedAt: soloT0.Add(10 * time.Second), Queued: true, QueuedAt: soloT0.Add(11 * time.Second)},
		{MediaID: "run-old", StartedAt: soloT0.Add(5 * time.Second)},
		// ties: queued at the same instant → accepted earlier first; running
		// started at the same instant → media id, so polls do not reshuffle
		{MediaID: "q-tie-b", StartedAt: soloT0.Add(2 * time.Second), Queued: true, QueuedAt: soloT0.Add(40 * time.Second)},
		{MediaID: "q-tie-a", StartedAt: soloT0.Add(3 * time.Second), Queued: true, QueuedAt: soloT0.Add(40 * time.Second)},
		{MediaID: "run-tie-b", StartedAt: soloT0.Add(25 * time.Second)},
		{MediaID: "run-tie-a", StartedAt: soloT0.Add(25 * time.Second)},
	}
	svc := activityWith(fakeBatch{}, jobs)

	var order []string
	for _, j := range svc.GetActivity(context.Background()).ActiveJobs.Jobs {
		order = append(order, j.MediaID)
	}
	assert.Equal(t, []string{
		"run-old", "run-new", "run-tie-a", "run-tie-b",
		"q-early", "q-late", "q-tie-b", "q-tie-a",
	}, order)
}

func TestActivity_BatchRowComesBeforeSoloRows(t *testing.T) {
	svc := activityWith(fakeBatch{active: true, current: 2, total: 10, item: "沙丘"},
		fakeSoloJobs{{MediaID: "ep-1", Title: lioness(1)}, {MediaID: "ep-2", Title: lioness(2)}})

	jobs := svc.GetActivity(context.Background()).ActiveJobs.Jobs
	require.Len(t, jobs, 3)
	assert.Equal(t, "generation_batch", jobs[0].Kind, "one row per batch, ahead of the solo rows (k1)")
	assert.Equal(t, "transcription", jobs[1].Kind)
	assert.Equal(t, "transcription", jobs[2].Kind)
}

// AC #7 [@contract-v1]: the exact wire keys.
func TestActivity_SoloRowWireKeys(t *testing.T) {
	svc := activityWith(fakeBatch{}, fakeSoloJobs{
		{MediaID: "ep-2", Title: lioness(2), StartedAt: soloT0},
		{MediaID: "ep-5", Title: lioness(5), StartedAt: soloT0, Queued: true, QueuedAt: soloT0},
	})
	svc.SetStepReader(fakeSteps{"ep-2": {Phase: "transcribing", Current: 3, Total: 22}})

	blob, err := json.Marshal(svc.GetActivity(context.Background()).ActiveJobs.Jobs)
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"kind":"transcription","percent_done":13,"detail":"特種部隊：母獅 S01E02","media_id":"ep-2",
		 "step":{"phase":"transcribing","current":3,"total":22}},
		{"kind":"transcription","percent_done":0,"detail":"特種部隊：母獅 S01E05","media_id":"ep-5","queued":true}
	]`, string(blob))
}

func TestActivity_OtherKindsCarryNoSoloKeys(t *testing.T) {
	svc := activityWith(fakeBatch{active: true, current: 1, total: 2, item: "x"}, fakeSoloJobs(nil))
	blob, err := json.Marshal(svc.GetActivity(context.Background()).ActiveJobs.Jobs)
	require.NoError(t, err)
	assert.NotContains(t, string(blob), "media_id")
	assert.NotContains(t, string(blob), "queued")
}

// ─── TranscriptionService (legacy mode) ────────────────────────────────────

func TestTranscriptionService_ActiveSoloJobsListsEverySoloJob(t *testing.T) {
	svc := newActivityTestService()
	clock := soloT0
	svc.now = func() time.Time { return clock }
	assert.Empty(t, svc.ActiveSoloJobs(), "idle")

	_, err := svc.acquireJob("batch-1", "", false)
	require.NoError(t, err)
	jobA, err := svc.acquireJob("solo-1", "沙丘", true)
	require.NoError(t, err)
	clock = soloT0.Add(time.Minute)
	_, err = svc.acquireJob("solo-2", "奧本海默", true)
	require.NoError(t, err)

	byID := map[string]SoloJobSnapshot{}
	for _, s := range svc.ActiveSoloJobs() {
		byID[s.MediaID] = s
	}
	require.Len(t, byID, 2, "batch / pipeline jobs are the batch row's, never a solo row")
	assert.Equal(t, SoloJobSnapshot{MediaID: "solo-1", Title: "沙丘", StartedAt: soloT0}, byID["solo-1"])
	assert.Equal(t, SoloJobSnapshot{MediaID: "solo-2", Title: "奧本海默", StartedAt: soloT0.Add(time.Minute)}, byID["solo-2"])

	// Waiting for the slot, then holding it.
	clock = soloT0.Add(2 * time.Minute)
	svc.setSoloQueued("solo-1", jobA, true)
	clock = soloT0.Add(3 * time.Minute)
	svc.setSoloQueued("solo-1", jobA, true) // a repeat does not restart the wait
	for _, s := range svc.ActiveSoloJobs() {
		if s.MediaID == "solo-1" {
			assert.True(t, s.Queued)
			assert.Equal(t, soloT0.Add(2*time.Minute), s.QueuedAt)
		}
	}
	svc.setSoloQueued("solo-1", jobA, false)
	for _, s := range svc.ActiveSoloJobs() {
		if s.MediaID == "solo-1" {
			assert.False(t, s.Queued)
			assert.True(t, s.QueuedAt.IsZero())
		}
	}

	// A stale observer (an older job id on the same media) changes nothing.
	svc.setSoloQueued("solo-2", "some-older-job", true)
	for _, s := range svc.ActiveSoloJobs() {
		assert.False(t, s.Queued, s.MediaID)
	}
	// Home readout still sees the bool it reads (AC #9).
	active, _, n, _, _ := svc.ActivityProgress()
	assert.True(t, active)
	assert.Equal(t, 2, n)
}

// CR M1: the observer StartTranscription attaches to the run ctx reaches the
// job — a wait reported anywhere under that ctx (ExtractAudio) flips the row.
func TestTranscriptionService_RunCtxObserverFlipsTheSoloJob(t *testing.T) {
	svc := newActivityTestService()
	svc.now = func() time.Time { return soloT0 }
	jobID, err := svc.acquireJob("solo-1", "沙丘", true)
	require.NoError(t, err)

	ctx := svc.withSoloQueueObserver(context.Background(), "solo-1", jobID)
	ReportExtractWait(ctx, true)
	snaps := svc.ActiveSoloJobs()
	require.Len(t, snaps, 1)
	assert.True(t, snaps[0].Queued)
	assert.Equal(t, soloT0, snaps[0].QueuedAt)

	ReportExtractWait(ctx, false)
	assert.False(t, svc.ActiveSoloJobs()[0].Queued)
}

// ─── The extraction-wait observer ──────────────────────────────────────────

type waitLog struct {
	mu   sync.Mutex
	seen []bool
}

func (w *waitLog) observe(waiting bool) {
	w.mu.Lock()
	w.seen = append(w.seen, waiting)
	w.mu.Unlock()
}

func (w *waitLog) snapshot() []bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]bool(nil), w.seen...)
}

func TestExtractWaitObserver_ReportsAndAdapts(t *testing.T) {
	ReportExtractWait(context.Background(), true) // no observer: a no-op, no panic
	assert.Nil(t, ExtractWaitReporter(context.Background()), "no observer → nil narrator, the gate skips it")

	var log waitLog
	ctx := WithExtractWaitObserver(context.Background(), log.observe)
	ReportExtractWait(ctx, true)
	narrate := ExtractWaitReporter(ctx)
	require.NotNil(t, narrate)
	narrate(3)
	narrate(0)
	assert.Equal(t, []bool{true, true, false}, log.snapshot())

	assert.Equal(t, context.Background(), WithExtractWaitObserver(context.Background(), nil))
}

func TestAudioExtractor_BusySemaphoreIsAWaitForTheSlot(t *testing.T) {
	svc := NewAudioExtractorService(1, time.Second, nil)
	svc.available = true // the wait happens before ffmpeg is ever looked at
	svc.semaphore <- struct{}{}

	var log waitLog
	ctx, cancel := context.WithCancel(WithExtractWaitObserver(context.Background(), log.observe))
	done := make(chan error, 1)
	go func() {
		_, err := svc.ExtractAudio(ctx, "/x.mkv", 0)
		done <- err
	}()

	require.Eventually(t, func() bool { return len(log.snapshot()) == 1 }, time.Second, 5*time.Millisecond)
	assert.Equal(t, []bool{true}, log.snapshot(), "blocked behind another audio extraction = 排隊中")

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	assert.Equal(t, []bool{true, false}, log.snapshot(), "giving up ends the wait")
}

func TestAudioExtractor_SlotGiveUpEndsTheWait(t *testing.T) {
	svc := NewAudioExtractorService(1, time.Second, nil, WithAudioExtractSlot(&fakeExtractSlot{err: context.Canceled}))
	svc.available = true

	var log waitLog
	_, err := svc.ExtractAudio(WithExtractWaitObserver(context.Background(), log.observe), "/x.mkv", 0)

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []bool{false}, log.snapshot(), "a free semaphore never reported a wait; the abort clears any the gate reported")
}
