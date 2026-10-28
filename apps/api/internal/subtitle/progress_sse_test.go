package subtitle

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/services"
	"github.com/vido/api/internal/sse"
)

// fakeBroadcaster captures what would have gone on the wire.
type fakeBroadcaster struct{ events []sse.Event }

func (f *fakeBroadcaster) Broadcast(event sse.Event) { f.events = append(f.events, event) }

func (f *fakeBroadcaster) last(t *testing.T) map[string]interface{} {
	t.Helper()
	require.NotEmpty(t, f.events)
	payload, ok := f.events[len(f.events)-1].Data.(map[string]interface{})
	require.True(t, ok, "payload is a map[string]interface{} since disc-2026-10-solo-live-spend (cost keys are numbers)")
	return payload
}

// ─── AC #6: the progress hook → SSE bridge ─────────────────────────────────

func TestSSEProgressHook_EmitsSubtitleProgressWithTheItemIdentity(t *testing.T) {
	hub := &fakeBroadcaster{}
	hook := NewSSEProgressHook(hub)

	hook(MediaRef{ID: "ep-1", MediaType: "episode"}, StageExtracting, "extracting embedded subtitle track")

	require.Len(t, hub.events, 1)
	assert.Equal(t, sse.EventSubtitleProgress, hub.events[0].Type,
		"event type is UNCHANGED — sse/hub.go stays locked and the frontend keeps its existing listener")

	payload := hub.last(t)
	assert.Equal(t, "ep-1", payload["media_id"])
	assert.Equal(t, "episode", payload["media_type"],
		"one Pipeline serves both workers — without the identity a season batch could not attribute an event")
	assert.Equal(t, "extracting", payload["stage"])
}

func TestSSEProgressHook_TranslatesStageMessagesToZhTW(t *testing.T) {
	cases := []struct {
		stage PipelineStage
		raw   string
		want  string
	}{
		{StageProbing, "probing embedded subtitle tracks", "偵測內嵌字幕軌…"},
		{StageExtracting, "extracting embedded subtitle track", "抽取內嵌字幕中…"},
		{StageExtracting, "waiting for the extraction slot (2 ahead)", "等待抽軌（前方 2 件）"},
		{StagePlacing, "placing subtitle file", "寫入字幕檔…"},
		{StageComplete, "subtitle generated", "字幕已生成"},
		// sub-6-2 AC #4: the transport-retry narration keeps its numbers.
		{StageTranslating, "chunk 7/60 timed out, retrying 2/3", "第 7/60 段逾時，重試 2/3"},
		{StageTranslating, "chunk 7/60 failed transiently, retrying 3/3", "第 7/60 段暫時失敗，重試 3/3"},
		{StageTranslating, "chunk 7/60 kept English after transport retries", "第 7/60 段暫時無法翻譯，保留英文繼續"},
		{StageComplete, "subtitle generated with 10 cues kept in English — queued for a retry", "字幕已生成（10 句暫留英文，將再排程翻譯）"},
	}
	for _, tc := range cases {
		t.Run(string(tc.stage)+"/"+tc.raw, func(t *testing.T) {
			hub := &fakeBroadcaster{}
			NewSSEProgressHook(hub)(MediaRef{ID: "m1", MediaType: "movie"}, tc.stage, tc.raw)
			assert.Equal(t, tc.want, hub.last(t)["message"],
				"the pipeline emits English for the log; the wire carries zh-TW for the user (Rule 3)")
		})
	}
}

// TestSSEProgressHook_TranslateChunkCountsSurviveTheTranslation — the "第 N/M 段"
// counter is the ONLY progress signal during the multi-minute half of a run
// (AC #7b: a 600-cue episode is ~60 chunks). Dropping the numbers would leave
// the user staring at a static string for three minutes.
func TestSSEProgressHook_TranslateChunkCountsSurviveTheTranslation(t *testing.T) {
	hub := &fakeBroadcaster{}
	raw := fmt.Sprintf(translateChunkProgressFormat, 7, 60)

	NewSSEProgressHook(hub)(MediaRef{ID: "m1", MediaType: "movie"}, StageTranslating, raw)

	assert.Equal(t, "翻譯中（第 7/60 段）", hub.last(t)["message"])
}

func TestSSEProgressHook_UnparseableTranslateMessageStillReadsAsProgress(t *testing.T) {
	hub := &fakeBroadcaster{}
	NewSSEProgressHook(hub)(MediaRef{ID: "m1", MediaType: "movie"}, StageTranslating, "translating")

	assert.Equal(t, "翻譯中…", hub.last(t)["message"],
		"a message the bridge cannot parse must degrade to a generic zh-TW line, never leak English")
}

// TestSSEProgressHook_TerminalStagesCarryTheirReason — `skipped` and `failed`
// carry a machine-authored reason (a route verdict, an error). It is the only
// thing that tells the user WHY, so it must survive.
func TestSSEProgressHook_TerminalStagesCarryTheirReason(t *testing.T) {
	hub := &fakeBroadcaster{}
	hook := NewSSEProgressHook(hub)

	hook(MediaRef{ID: "m1", MediaType: "movie"}, StageSkipped, "already has zh-Hant")
	assert.Equal(t, "已略過：already has zh-Hant", hub.last(t)["message"])

	hook(MediaRef{ID: "m1", MediaType: "movie"}, StageFailed, "ffmpeg exited 1")
	assert.Equal(t, "字幕生成失敗：ffmpeg exited 1", hub.last(t)["message"])
}

func TestSSEProgressHook_NilHubIsANoOp(t *testing.T) {
	hook := NewSSEProgressHook(nil)
	require.NotNil(t, hook, "a nil hub must not produce a nil hook — WithProgress would install nothing")
	assert.NotPanics(t, func() {
		hook(MediaRef{ID: "m1", MediaType: "movie"}, StageProbing, "probing")
	})
}

// TestSSEProgressHook_WiresIntoThePipelineAsAProgressOption — the bridge is
// only useful if it satisfies WithProgress's signature; this pins that.
func TestSSEProgressHook_WiresIntoThePipelineAsAProgressOption(t *testing.T) {
	hub := &fakeBroadcaster{}
	p := NewPipeline(&fakeTranslator{}, &recordingConverter{}, nil,
		WithProgress(NewSSEProgressHook(hub)))

	p.emitProgress(MediaRef{ID: "m1", MediaType: "movie"}, StageProbing, "probing embedded subtitle tracks")

	require.Len(t, hub.events, 1)
	assert.Equal(t, "偵測內嵌字幕軌…", hub.last(t)["message"])
}

// ─── disc-2026-10-solo-live-spend AC #2 / #3 ───────────────────────────────

func TestSSEProgressHook_SoloItemCarriesTheJobsLiveSpend(t *testing.T) {
	hub := &fakeBroadcaster{}
	board := services.NewSoloSpendBoard()
	budget := ai.NewBudget(5)
	board.Track("ep-1", budget)
	hook := NewSSEProgressHook(hub, WithProgressSpend(board))
	ref := MediaRef{ID: "ep-1", MediaType: "episode"}

	hook(ref, StageTranslating, "translating chunk 1/3")
	first := hub.last(t)
	assert.Equal(t, 0.0, first["spent_usd"], "nothing paid yet is a recorded 0")
	assert.Equal(t, 5.0, first["budget_usd"])

	budget.RecordLLM("claude-sonnet-4-5", 100_000, 20_000)
	hook(ref, StageTranslating, "translating chunk 2/3")
	second := hub.last(t)
	assert.Equal(t, budget.SpentUSD(), second["spent_usd"], "the next frame reads the meter again")
	assert.Greater(t, second["spent_usd"].(float64), 0.0)
	assert.Equal(t, "翻譯中（第 2/3 段）", second["message"], "the existing keys are untouched")
}

func TestSSEProgressHook_NoCeilingSendsNoBudgetKey(t *testing.T) {
	hub := &fakeBroadcaster{}
	board := services.NewSoloSpendBoard()
	board.Track("mv-1", ai.NewBudget(0))
	NewSSEProgressHook(hub, WithProgressSpend(board))(MediaRef{ID: "mv-1", MediaType: "movie"}, StageProbing, "")

	payload := hub.last(t)
	assert.Contains(t, payload, "spent_usd")
	assert.NotContains(t, payload, "budget_usd", "absent = no ceiling, never 0")
}

func TestSSEProgressHook_ItemNotOnTheBoardKeepsItsFourKeys(t *testing.T) {
	hub := &fakeBroadcaster{}
	board := services.NewSoloSpendBoard()
	board.Track("someone-else", ai.NewBudget(5))
	for _, hook := range []func(MediaRef, PipelineStage, string){
		NewSSEProgressHook(hub),                           // no board wired
		NewSSEProgressHook(hub, WithProgressSpend(board)), // a batch / pool item
		NewSSEProgressHook(hub, WithProgressSpend(nil)),   // nil board
	} {
		hook(MediaRef{ID: "batch-item", MediaType: "movie"}, StageTranslating, "translating chunk 1/2")
		raw, err := json.Marshal(hub.events[len(hub.events)-1].Data)
		require.NoError(t, err)
		assert.JSONEq(t,
			`{"media_id":"batch-item","media_type":"movie","stage":"translating","message":"翻譯中（第 1/2 段）"}`,
			string(raw), "the wire shape of a non-solo item is byte-for-byte what it was")
	}
}
