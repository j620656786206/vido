package subtitle

import (
	"fmt"

	"github.com/vido/api/internal/sse"
)

// translateChunkProgressFormat is the wire between pipeline.go's chunk loop and
// the zh-TW message this bridge composes.
//
// It is a shared const rather than two independent string literals ON PURPOSE:
// the bridge parses the chunk numbers back out (they are the only progress
// signal during the multi-minute half of a run), and a silent drift between an
// fmt.Sprintf here and an fmt.Sscanf there would degrade "翻譯中（第 7/60 段）"
// into a static "翻譯中…" with nothing failing anywhere.
const translateChunkProgressFormat = "translating chunk %d/%d"

// The sub-6-2 AC #4 transport-retry narration, same Sscanf convention: the
// chunk that timed out (or failed transiently), the attempt about to start,
// and the attempt budget; then the verdict when the budget ran out.
const (
	translateChunkTimeoutFormat = "chunk %d/%d timed out, retrying %d/%d"
	translateChunkRetryFormat   = "chunk %d/%d failed transiently, retrying %d/%d"
	translateChunkEnglishFormat = "chunk %d/%d kept English after transport retries"
)

// ProgressBroadcaster is the narrow port over *sse.Hub — one method, so the
// bridge is testable without standing up a hub goroutine.
type ProgressBroadcaster interface {
	Broadcast(event sse.Event)
}

// NewSSEProgressHook adapts the Pipeline's progress hook (sub-1-5b) onto the
// existing SSE hub (AC #6). main.go installs the result with WithProgress.
//
// Three deliberate choices:
//
//   - The event TYPE is unchanged (`subtitle_progress`) and so is the payload
//     shape the search engine already broadcasts, so sse/hub.go stays locked and
//     the frontend keeps the listener it has. The four new stage values
//     (`probing`/`extracting`/`translating`/`skipped`, sub-1-3 [@contract-v1])
//     reach the wire here for the first time; the FE consumes `stage` as a loose
//     string and renders `message`, so it degrades gracefully today.
//   - {media_id, media_type} ride every event because ONE Pipeline serves both
//     pool workers — without them a concurrent season batch could not attribute
//     an event to an item.
//   - The zh-TW composition happens HERE, at the wiring layer, not in the
//     stages: the pipeline's own messages are log lines (English, for the
//     operator), and Rule 3 keeps user-facing strings at the edge.
func NewSSEProgressHook(hub ProgressBroadcaster) func(ref MediaRef, stage PipelineStage, message string) {
	return func(ref MediaRef, stage PipelineStage, message string) {
		if hub == nil {
			return
		}
		hub.Broadcast(sse.Event{
			Type: sse.EventSubtitleProgress,
			Data: map[string]string{
				"media_id":   ref.ID,
				"media_type": ref.MediaType,
				"stage":      string(stage),
				"message":    zhTWStageMessage(stage, message),
			},
		})
	}
}

// zhTWStageMessage turns one stage (plus the pipeline's English detail) into the
// user-facing line. An unmapped stage passes its message through rather than
// rendering an empty bubble — the search-path stages (searching/scoring/…)
// never reach this hook, but a stage added later would still say something.
func zhTWStageMessage(stage PipelineStage, message string) string {
	switch stage {
	case StageProbing:
		return "偵測內嵌字幕軌…"
	case StageExtracting:
		return "抽取內嵌字幕中…"
	case StageTranslating:
		var chunk, total, next, max int
		if n, err := fmt.Sscanf(message, translateChunkProgressFormat, &chunk, &total); n == 2 && err == nil {
			return fmt.Sprintf("翻譯中（第 %d/%d 段）", chunk, total)
		}
		if n, err := fmt.Sscanf(message, translateChunkTimeoutFormat, &chunk, &total, &next, &max); n == 4 && err == nil {
			return fmt.Sprintf("第 %d/%d 段逾時，重試 %d/%d", chunk, total, next, max)
		}
		if n, err := fmt.Sscanf(message, translateChunkRetryFormat, &chunk, &total, &next, &max); n == 4 && err == nil {
			return fmt.Sprintf("第 %d/%d 段暫時失敗，重試 %d/%d", chunk, total, next, max)
		}
		if n, err := fmt.Sscanf(message, translateChunkEnglishFormat, &chunk, &total); n == 2 && err == nil {
			return fmt.Sprintf("第 %d/%d 段暫時無法翻譯，保留英文繼續", chunk, total)
		}
		return "翻譯中…"
	case StageConverting:
		return "轉換為繁體中文…"
	case StagePlacing:
		return "寫入字幕檔…"
	case StageComplete:
		return "字幕已生成"
	case StageSkipped:
		// The reason is machine-authored (a route verdict) and is the only thing
		// that tells the user WHY nothing happened, so it survives verbatim.
		return "已略過：" + message
	case StageFailed:
		return "字幕生成失敗：" + message
	default:
		return message
	}
}
