package subtitle

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
)

// disc-2026-10-reattach-progress-starts-at-extracting

const trackedID = "792f235f-bca9-4533-abd5-bb65cae37498"

func asrEvent(t sse.EventType, extra map[string]interface{}) sse.Event {
	data := map[string]interface{}{"job_id": "j1", "media_id": trackedID, "message": "msg"}
	for k, v := range extra {
		data[k] = v
	}
	return sse.Event{Type: t, Data: data}
}

func stageEvent(stage PipelineStage) sse.Event {
	return sse.Event{Type: sse.EventSubtitleProgress, Data: map[string]string{
		"media_id": trackedID, "media_type": "episode", "stage": string(stage), "message": "stage msg",
	}}
}

func TestPhaseTracker_RecordsEachProgressEvent(t *testing.T) {
	pct := 42.5
	tests := []struct {
		name string
		ev   sse.Event
		want PhaseSnapshot
	}{
		{"solo start / ASR extract", asrEvent(services.EventTranscriptionExtracting, nil), PhaseSnapshot{Phase: "extracting", Message: "msg"}},
		{"ASR transcribing", asrEvent(services.EventTranscriptionProgress, nil), PhaseSnapshot{Phase: "transcribing", Message: "msg"}},
		{"ASR translating with percentage", asrEvent(services.EventTranscriptionTranslating, map[string]interface{}{"percentage": pct}), PhaseSnapshot{Phase: "translating", Message: "msg", Percentage: &pct}},
		{"translating without percentage", asrEvent(services.EventTranscriptionTranslating, nil), PhaseSnapshot{Phase: "translating", Message: "msg"}},
		{"pipeline probing", stageEvent(StageProbing), PhaseSnapshot{Phase: "extracting", Message: "stage msg"}},
		{"pipeline extracting", stageEvent(StageExtracting), PhaseSnapshot{Phase: "extracting", Message: "stage msg"}},
		{"pipeline translating", stageEvent(StageTranslating), PhaseSnapshot{Phase: "translating", Message: "stage msg"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := NewPhaseTracker()
			tr.Observe(tt.ev)
			got, ok := tr.Phase(trackedID)
			require.True(t, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPhaseTracker_IntegerPercentage(t *testing.T) {
	tr := NewPhaseTracker()
	// translateAndPersist's first translation_progress sends a literal 0 (int).
	tr.Observe(asrEvent(services.EventTranscriptionTranslating, map[string]interface{}{"percentage": 0}))
	got, ok := tr.Phase(trackedID)
	require.True(t, ok)
	require.NotNil(t, got.Percentage)
	assert.Equal(t, 0.0, *got.Percentage)
}

func TestPhaseTracker_LatestEventWins(t *testing.T) {
	tr := NewPhaseTracker()
	tr.Observe(asrEvent(services.EventTranscriptionExtracting, nil))
	tr.Observe(asrEvent(services.EventTranscriptionProgress, nil))
	got, _ := tr.Phase(trackedID)
	assert.Equal(t, "transcribing", got.Phase)
}

func TestPhaseTracker_TerminalsForget(t *testing.T) {
	terminals := []sse.Event{
		asrEvent(services.EventTranscriptionComplete, nil),
		asrEvent(services.EventTranscriptionFailed, nil),
		stageEvent(StageComplete),
		stageEvent(StageFailed),
		stageEvent(StageSkipped),
	}
	for _, term := range terminals {
		tr := NewPhaseTracker()
		tr.Observe(asrEvent(services.EventTranscriptionProgress, nil))
		tr.Observe(term)
		_, ok := tr.Phase(trackedID)
		assert.False(t, ok, "%s %v should forget the item", term.Type, term.Data)
	}
}

func TestPhaseTracker_IgnoresUnrelatedEvents(t *testing.T) {
	tr := NewPhaseTracker()
	tr.Observe(asrEvent(services.EventTranscriptionProgress, nil))
	before, _ := tr.Phase(trackedID)

	for _, ev := range []sse.Event{
		stageEvent(StageSearching), // search path: not generation
		{Type: sse.EventScanProgress, Data: map[string]interface{}{"media_id": trackedID, "phase": "x"}},
		{Type: services.EventTranscriptionProgress, Data: map[string]interface{}{"phase": "transcribing"}}, // no media_id
		{Type: services.EventTranscriptionProgress, Data: "not a map"},
		{Type: services.EventTranscriptionComplete, Data: map[string]interface{}{"media_id": "another"}},
	} {
		tr.Observe(ev)
	}

	after, ok := tr.Phase(trackedID)
	require.True(t, ok)
	assert.Equal(t, before, after)
	_, ok = tr.Phase("unknown")
	assert.False(t, ok)
}

func TestPhaseTracker_ObservesHub(t *testing.T) {
	hub := sse.NewHub()
	defer hub.Close()
	tr := NewPhaseTracker()
	hub.AddObserver(tr.Observe)

	hub.Broadcast(asrEvent(services.EventTranscriptionProgress, nil))

	got, ok := tr.Phase(trackedID)
	require.True(t, ok, "Broadcast reaches the observer synchronously")
	assert.Equal(t, "transcribing", got.Phase)
}
