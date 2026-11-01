package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/zhtw"
)

// Implements: disc-2026-11-burned-in-subs-b1 AC #1, #2, #3, #5, #7

// burnedInModel is the model that passed the burned-in exam
// (disc-2026-11-burned-in-subs-a: 29/29 burned, 7/7 not, every decidable
// script right). Pinned, not "the translation default": Haiku sat the same
// exam and transcribed 《白髮》's Traditional lines as Simplified.
const burnedInModel = "claude-sonnet-5"

// burnedInMaxTokens: the exam's replies were ~50 output tokens; the cap
// leaves room for six strips' worth of lines with thinking off.
const burnedInMaxTokens = 512

// burnedInPrompt is the exam's prompt, verbatim
// (eval/burned-in-exam-2026-11/run.py). The script is copied, never judged:
// OpenCC decides Simplified vs Traditional (zhtw.ClassifyScript).
const burnedInPrompt = "This image stacks 6 strips. Each strip is the bottom third of one frame sampled from the same video episode. " +
	"Decide whether the video has Chinese subtitles burned into the picture (open captions: lines of Chinese dialogue text, usually centred near the bottom). " +
	"Ignore channel logos, watermarks, on-screen signs, credits rolls and English-only captions. " +
	"Copy every Chinese subtitle line you can read exactly as written, character by character, without converting between simplified and traditional. " +
	"Reply with exactly one line of JSON and nothing else: {\"burned_chinese\": true|false, \"lines\": [\"...\"]}."

// Why a season check decided nothing — BurnedInCheckResult.Reason.
const (
	BurnedInReasonNoEpisode        = "no_episode_to_check" // no episode with a file whose tracks were read empty
	BurnedInReasonFramesUnreadable = "frames_unavailable"  // fewer than 3 frames could be grabbed
	BurnedInReasonReplyUnreadable  = "reply_unreadable"    // the model's reply was not the JSON asked for
	BurnedInReasonScriptUnreadable = "script_check_failed" // OpenCC could not run on the copied lines
)

// ErrBurnedInNotConfigured: no Claude key — the check cannot run. Wraps
// ai.ErrAINotConfigured.
var ErrBurnedInNotConfigured = fmt.Errorf("%w: the burned-in subtitle check needs a Claude API key", ai.ErrAINotConfigured)

// ErrBurnedInModelCall: the model call itself failed (network, 5xx, a rejected
// key) — nothing was decided and nothing written.
var ErrBurnedInModelCall = errors.New("burned-in model call failed")

// BurnedInEpisodeStore is the episode access CheckSeason needs.
type BurnedInEpisodeStore interface {
	FindBySeasonNumber(ctx context.Context, seriesID string, seasonNumber int) ([]models.Episode, error)
	SetSeasonBurnedInSubtitle(ctx context.Context, seriesID string, season int, value string, checkedAt time.Time) (int64, error)
	MarkSeasonBurnedInChecked(ctx context.Context, seriesID string, season int, checkedAt time.Time) (int64, error)
}

// BurnedInVision is the model access CheckSeason needs —
// ClaudeProviderHolder in production.
type BurnedInVision interface {
	IsConfigured(ctx context.Context) bool
	ai.ImageCompleter
}

// BurnedInSpendRecorder writes the ai_side_spend row for each model call.
type BurnedInSpendRecorder interface {
	Create(ctx context.Context, row *models.AISideSpend) error
}

// BurnedInCheckResult is one season check's outcome
// (POST /api/v1/subtitles/burned-in/check).
type BurnedInCheckResult struct {
	// Result is zh-Hans / zh-Hant / zh / none; nil = not decided, see Reason.
	Result *string `json:"result"`
	// Reason says why Result is nil; empty when it is not.
	Reason string `json:"reason,omitempty"`
	// EpisodesUpdated is how many episodes of the season got Result written.
	EpisodesUpdated int64 `json:"episodes_updated"`
	// CostUSD is what the model calls cost together (0 when none was made).
	CostUSD float64 `json:"cost_usd"`
	// CheckedEpisodeIDs are the episodes whose frames were read, in the order
	// they were checked: one, or two when the first said "none". Never null.
	CheckedEpisodeIDs []string `json:"checked_episode_ids"`
}

// BurnedInSubtitleService answers "is Chinese burned into this season's
// picture?" for episodes whose files carry no subtitle track at all, and
// stores the answer where ChineseSubtitleVerdict reads it
// (episodes.burned_in_subtitle) — ruling 2A: burned Simplified counts as
// having Chinese.
type BurnedInSubtitleService struct {
	episodes BurnedInEpisodeStore
	vision   BurnedInVision
	frames   BurnedInFrameGrabber
	conv     zhtw.ScriptConverter
	spend    BurnedInSpendRecorder
	logger   *slog.Logger
	now      func() time.Time
}

// NewBurnedInSubtitleService builds the service. spend may be nil (nothing is
// recorded); logger may be nil.
func NewBurnedInSubtitleService(episodes BurnedInEpisodeStore, vision BurnedInVision, frames BurnedInFrameGrabber, conv zhtw.ScriptConverter, spend BurnedInSpendRecorder, logger *slog.Logger) *BurnedInSubtitleService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BurnedInSubtitleService{
		episodes: episodes, vision: vision, frames: frames, conv: conv, spend: spend,
		logger: logger.With("component", "burned_in_subtitle"), now: time.Now,
	}
}

// CheckSeason checks the season's episodes whose files were probed and carry
// no subtitle track at all (subtitle_tracks = '[]') and writes the answer onto
// every one of them. A season is one release in practice, so ONE episode
// usually stands for it (~US$0.002) — except when it says "none".
//
// "none" needs a second opinion: six frames of one episode can all land on
// wordless moments (a 4-minute 《白髮》 action clip came back "none", and
// "none" claims the whole season has no Chinese). So when the first episode
// says not burned and the season has another trackless episode, the one in
// the MIDDLE of that list is checked too — not the next one, adjacent
// episodes share structure (cold opens, recaps). The second episode's answer
// is the season's; if it cannot be decided, nothing is written. A season with
// a single trackless episode accepts its one "none".
//
// Anything that keeps it from deciding (frames, an unreadable reply, OpenCC)
// returns a nil Result with a Reason and writes no value: "could not tell" is
// never stored as "none". It does stamp burned_in_checked_at (see undecided). Errors are reserved for a missing key
// (ErrBurnedInNotConfigured), the database, and a model call that failed
// outright.
func (s *BurnedInSubtitleService) CheckSeason(ctx context.Context, seriesID string, season int) (BurnedInCheckResult, error) {
	res := BurnedInCheckResult{CheckedEpisodeIDs: []string{}}
	if !s.vision.IsConfigured(ctx) {
		return res, ErrBurnedInNotConfigured
	}
	eps, err := s.episodes.FindBySeasonNumber(ctx, seriesID, season)
	if err != nil {
		return res, fmt.Errorf("read season episodes: %w", err)
	}
	trackless := tracklessEpisodes(eps)
	if len(trackless) == 0 {
		res.Reason = BurnedInReasonNoEpisode
		return res, nil
	}
	log := s.logger.With("series_id", seriesID, "season", season)

	spent := decimal.Zero
	check := func(ep models.Episode) (episodeCheck, error) {
		res.CheckedEpisodeIDs = append(res.CheckedEpisodeIDs, ep.ID)
		c, err := s.checkEpisode(ctx, ep)
		spent = spent.Add(c.cost)
		res.CostUSD = spent.InexactFloat64()
		return c, err
	}

	first, err := check(trackless[0])
	if err != nil {
		return res, err
	}
	if first.reason != "" {
		return s.undecided(ctx, res, seriesID, season, first.reason)
	}
	decided := first
	if first.value == models.BurnedInNone {
		if len(trackless) > 1 {
			second, err := check(trackless[len(trackless)/2])
			if err != nil {
				return res, err
			}
			if second.reason != "" {
				log.Info("burned-in check: first episode said none, second could not be decided — season left unjudged",
					"first_episode_id", trackless[0].ID, "second_episode_id", trackless[len(trackless)/2].ID, "reason", second.reason)
				return s.undecided(ctx, res, seriesID, season, second.reason)
			}
			decided = second
		} else {
			log.Info("burned-in check: none decided on the season's only trackless episode", "episode_id", trackless[0].ID)
		}
	}

	n, err := s.episodes.SetSeasonBurnedInSubtitle(ctx, seriesID, season, decided.value, s.now())
	if err != nil {
		return res, fmt.Errorf("write burned-in result: %w", err)
	}
	value := decided.value
	res.Result = &value
	res.EpisodesUpdated = n
	log.Info("burned-in check done", "result", value, "lines", decided.lines,
		"checked_episode_ids", res.CheckedEpisodeIDs, "episodes_updated", n, "cost_usd", res.CostUSD)
	return res, nil
}

// undecided finishes a check that could not decide: no value is written, but
// the season's unanswered trackless episodes get burned_in_checked_at, so the
// sweep waits out its retry window instead of paying for the same question on
// every scan (disc-2026-11-burned-in-subs-b2 AC #2). A failed stamp is only a
// warning — the result is still "undecided".
func (s *BurnedInSubtitleService) undecided(ctx context.Context, res BurnedInCheckResult, seriesID string, season int, reason string) (BurnedInCheckResult, error) {
	res.Reason = reason
	if _, err := s.episodes.MarkSeasonBurnedInChecked(ctx, seriesID, season, s.now()); err != nil {
		s.logger.Warn("burned-in check: could not record the undecided check — it may be retried early",
			"series_id", seriesID, "season", season, "error", err)
	}
	return res, nil
}

// Configured reports whether a Claude key resolves — the sweep ends a round
// quietly without one.
func (s *BurnedInSubtitleService) Configured(ctx context.Context) bool {
	return s.vision.IsConfigured(ctx)
}

// episodeCheck is one episode's answer: value is one of the four stored
// values, or reason says why there is none. cost is what its model call cost
// (zero when no call was made).
type episodeCheck struct {
	value  string
	reason string
	lines  int
	cost   decimal.Decimal
}

// checkEpisode samples one episode, asks the model and reads the script.
func (s *BurnedInSubtitleService) checkEpisode(ctx context.Context, ep models.Episode) (episodeCheck, error) {
	log := s.logger.With("series_id", ep.SeriesID, "season", ep.SeasonNumber, "episode_id", ep.ID)
	image, err := s.sampleImage(ctx, ep)
	if err != nil {
		log.Warn("burned-in check: frames unavailable", "error", err)
		return episodeCheck{reason: BurnedInReasonFramesUnreadable}, nil
	}

	reply, cost, err := s.ask(ctx, image)
	out := episodeCheck{cost: cost}
	if err != nil {
		switch {
		case errors.Is(err, ai.ErrAIInvalidResponse):
			log.Warn("burned-in check: empty reply", "error", err)
			out.reason = BurnedInReasonReplyUnreadable
			return out, nil
		case errors.Is(err, ai.ErrAINotConfigured):
			return out, ErrBurnedInNotConfigured
		}
		return out, fmt.Errorf("%w: %w", ErrBurnedInModelCall, err)
	}

	burned, lines, ok := parseBurnedInReply(reply)
	if !ok {
		log.Warn("burned-in check: reply is not the JSON asked for", "reply", reply)
		out.reason = BurnedInReasonReplyUnreadable
		return out, nil
	}
	out.lines = len(lines)
	if !burned {
		out.value = models.BurnedInNone
		return out, nil
	}
	script, err := s.classifyScript(lines)
	if err != nil {
		log.Warn("burned-in check: OpenCC failed", "error", err)
		out.reason = BurnedInReasonScriptUnreadable
		return out, nil
	}
	out.value = burnedValueOfScript(script)
	return out, nil
}

func (s *BurnedInSubtitleService) classifyScript(lines []string) (string, error) {
	if s.conv == nil {
		return "", errors.New("no OpenCC converter")
	}
	return zhtw.ClassifyScript(s.conv, lines)
}

// tracklessEpisodes lists, in season order, the episodes with a file whose
// tracks were read and found empty.
func tracklessEpisodes(eps []models.Episode) []models.Episode {
	var out []models.Episode
	for _, e := range eps {
		if e.FilePath.Valid && e.FilePath.String != "" && e.SubtitleTracks.Valid && e.SubtitleTracks.String == "[]" {
			out = append(out, e)
		}
	}
	return out
}

// sampleImage grabs the sample frames and stacks their bottom strips.
func (s *BurnedInSubtitleService) sampleImage(ctx context.Context, ep models.Episode) ([]byte, error) {
	path := ep.FilePath.String
	duration := float64(ep.DurationSeconds.Int64)
	if !ep.DurationSeconds.Valid || duration <= 0 {
		d, err := s.frames.Duration(ctx, path)
		if err != nil {
			return nil, err
		}
		duration = d
	}
	var frames [][]byte
	for _, pct := range burnedInSamplePercents {
		f, err := s.frames.Frame(ctx, path, duration*float64(pct)/100)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			s.logger.Debug("burned-in check: frame skipped", "episode_id", ep.ID, "percent", pct, "error", err)
			continue
		}
		frames = append(frames, f)
	}
	return stackFrameStrips(frames)
}

// ask sends the image under its own uncapped Budget — the ai_service.go
// meteredParse shape — and records what it cost as a side-spend row, even
// when the reply turns out useless: the call was billed.
func (s *BurnedInSubtitleService) ask(ctx context.Context, image []byte) (string, decimal.Decimal, error) {
	budget := ai.NewBudget(0)
	callCtx := ai.WithThinkingOff(ai.WithModelID(ai.WithBudget(ctx, budget), burnedInModel))
	out, err := s.vision.CompleteImageWithUsage(callCtx, ai.ImageCompletionRequest{
		Image: image, MediaType: "image/png", Prompt: burnedInPrompt, MaxTokens: burnedInMaxTokens,
	})
	snap := budget.Snapshot()
	if s.spend != nil && snap.Spent.IsPositive() {
		row := &models.AISideSpend{
			Kind:         models.AISideSpendKindBurnedInCheck,
			Provider:     string(ai.ProviderNameClaude),
			ModelID:      burnedInModel,
			InputTokens:  snap.InputTokens,
			OutputTokens: snap.OutputTokens,
			SpentUSD:     snap.SpentUSD,
		}
		if werr := s.spend.Create(context.WithoutCancel(ctx), row); werr != nil {
			s.logger.Warn("Failed to record burned-in check spend — the check itself is unaffected",
				"error", werr, "spent_usd", snap.SpentUSD)
		}
	}
	return out.Text, snap.Spent, err
}

// burnedInReply is the JSON the prompt asks for. BurnedChinese is a pointer
// so a reply that omits it is unreadable, not "false".
type burnedInReply struct {
	BurnedChinese *bool    `json:"burned_chinese"`
	Lines         []string `json:"lines"`
}

// parseBurnedInReply reads the model's reply. Like the exam's run.py it takes
// the span from the first "{" to the last "}", so a reply wrapped in prose or
// a code fence still reads; anything else is ok=false.
func parseBurnedInReply(text string) (burned bool, lines []string, ok bool) {
	start, end := strings.Index(text, "{"), strings.LastIndex(text, "}")
	if start < 0 || end < start {
		return false, nil, false
	}
	var r burnedInReply
	if err := json.Unmarshal([]byte(text[start:end+1]), &r); err != nil || r.BurnedChinese == nil {
		return false, nil, false
	}
	return *r.BurnedChinese, r.Lines, true
}

// burnedValueOfScript maps a zhtw script onto the stored burned-in value.
func burnedValueOfScript(script string) string {
	switch script {
	case zhtw.ScriptSimplified:
		return models.BurnedInZhHans
	case zhtw.ScriptTraditional:
		return models.BurnedInZhHant
	}
	return models.BurnedInZh
}
