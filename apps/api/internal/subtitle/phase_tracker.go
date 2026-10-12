package subtitle

import (
	"sync"

	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
)

// PhaseTracker remembers where each item's subtitle generation is right now
// (disc-2026-10-reattach-progress-starts-at-extracting).
//
// The 管理字幕 dialog attaches to a run already going — on reopen, or on a
// 409 — and before this it could only start its stepper at 抽取: the next
// event put it right, but speech recognition sends one event at its start and
// nothing for minutes after, so a reopen mid-ASR sat on the wrong stage the
// whole time. GET …/transcribe/status now hands the dialog this snapshot as
// its starting point.
//
// It observes the hub (sse.Hub.AddObserver) rather than being told by each
// producer: a run's progress comes from three of them — the pipeline's
// subtitle_progress, the ASR leg's transcription_* and the solo runner — and
// all three go out through Broadcast. Keyed by media_id alone: movie and
// episode ids are UUIDs, and the ASR leg's events carry no media_type.
type PhaseTracker struct {
	mu      sync.Mutex
	byMedia map[string]PhaseSnapshot
}

// PhaseSnapshot is the last progress one item reported. Phase uses the
// dialog's vocabulary: extracting | transcribing | translating.
type PhaseSnapshot struct {
	Phase   string
	Message string
	// Percentage is translation_progress's 0–100 figure; nil when the event
	// carried none (and always nil outside translating).
	Percentage *float64
	// Current of Total is the count the latest event carried — transcribing:
	// chunks done; translating: cues translated
	// (disc-2026-10-activity-generation-progress-detail). 0/0 = no count, which
	// is also what an event WITHOUT one leaves behind: a count belongs to the
	// event that sent it, never to the phase.
	Current, Total int
}

// NewPhaseTracker returns an empty tracker; register Observe on the hub.
func NewPhaseTracker() *PhaseTracker {
	return &PhaseTracker{byMedia: make(map[string]PhaseSnapshot)}
}

// transcriptionPhases maps the transcription_* family (and the solo runner's
// start event, which reuses transcription_extracting) to the dialog's phases —
// the same table as the web hook's PHASE_EVENTS.
var transcriptionPhases = map[sse.EventType]string{
	services.EventTranscriptionExtracting:  "extracting",
	services.EventTranscriptionProgress:    "transcribing",
	services.EventTranscriptionTranslating: "translating",
}

// pipelineStagePhases maps the generation stages of subtitle_progress — the
// web hook's D6_STAGE_TO_PHASE. Search-path stages are absent and ignored.
var pipelineStagePhases = map[string]string{
	string(StageProbing):     "extracting",
	string(StageExtracting):  "extracting",
	string(StageTranslating): "translating",
}

// Observe records one broadcast. It runs on the broadcaster's goroutine for
// EVERY event, so it does a type switch and one locked map write at most.
func (t *PhaseTracker) Observe(ev sse.Event) {
	switch ev.Type {
	case services.EventTranscriptionComplete, services.EventTranscriptionFailed:
		if id := payloadString(ev.Data, "media_id"); id != "" {
			t.forget(id)
		}
	case sse.EventSubtitleProgress:
		id, stage := payloadString(ev.Data, "media_id"), payloadString(ev.Data, "stage")
		if id == "" {
			return
		}
		if phase, ok := pipelineStagePhases[stage]; ok {
			t.record(id, PhaseSnapshot{Phase: phase, Message: payloadString(ev.Data, "message")})
			return
		}
		switch PipelineStage(stage) {
		case StageComplete, StageFailed, StageSkipped:
			t.forget(id)
		}
	default:
		phase, ok := transcriptionPhases[ev.Type]
		if !ok {
			return
		}
		id := payloadString(ev.Data, "media_id")
		if id == "" {
			return
		}
		snap := PhaseSnapshot{Phase: phase, Message: payloadString(ev.Data, "message")}
		if phase == "translating" {
			snap.Percentage = payloadNumber(ev.Data, "percentage")
		}
		if cur, total := payloadNumber(ev.Data, "current"), payloadNumber(ev.Data, "total"); cur != nil && total != nil && *total > 0 {
			snap.Current, snap.Total = int(*cur), int(*total)
		}
		t.record(id, snap)
	}
}

// Phase returns the item's last reported progress; ok=false when none is
// recorded (nothing running, or running but no progress event yet).
func (t *PhaseTracker) Phase(mediaID string) (PhaseSnapshot, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	snap, ok := t.byMedia[mediaID]
	return snap, ok
}

// Step is Phase in the shape services.GenerationStepReader takes (services
// cannot import this package).
func (t *PhaseTracker) Step(mediaID string) (phase string, current, total int, ok bool) {
	snap, ok := t.Phase(mediaID)
	return snap.Phase, snap.Current, snap.Total, ok
}

func (t *PhaseTracker) record(id string, snap PhaseSnapshot) {
	t.mu.Lock()
	t.byMedia[id] = snap
	t.mu.Unlock()
}

func (t *PhaseTracker) forget(id string) {
	t.mu.Lock()
	delete(t.byMedia, id)
	t.mu.Unlock()
}

// payloadString reads a string key from the two payload shapes the producers use:
// map[string]string (subtitle_progress) and map[string]interface{}
// (transcription_*, solo runner).
func payloadString(data interface{}, key string) string {
	switch m := data.(type) {
	case map[string]string:
		return m[key]
	case map[string]interface{}:
		s, _ := m[key].(string)
		return s
	}
	return ""
}

// payloadNumber reads a numeric key from a map[string]interface{} payload.
func payloadNumber(data interface{}, key string) *float64 {
	m, ok := data.(map[string]interface{})
	if !ok {
		return nil
	}
	var f float64
	switch v := m[key].(type) {
	case float64:
		f = v
	case float32:
		f = float64(v)
	case int:
		f = float64(v)
	case int64:
		f = float64(v)
	default:
		return nil
	}
	return &f
}
