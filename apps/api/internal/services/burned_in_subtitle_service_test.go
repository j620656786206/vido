package services

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vido/api/internal/ai"
	"github.com/vido/api/internal/database"
	"github.com/vido/api/internal/database/migrations"
	"github.com/vido/api/internal/models"
	"github.com/vido/api/internal/repository"
)

// --- fakes ------------------------------------------------------------------

// testFrame is a 480×270 JPEG: red picture, blue bottom 32% (where captions
// would sit), so a test can see which part of the frame survived.
func testFrame(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 480, 270))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 270-86, 480, 270), image.NewUniform(color.RGBA{B: 255, A: 255}), image.Point{}, draw.Src)
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}))
	return buf.Bytes()
}

type fakeFrameGrabber struct {
	frame     []byte
	failAt    map[int]bool    // sample index (across all calls) → fail
	failPath  map[string]bool // every frame of this file fails
	paths     []string
	duration  float64
	durations int
	at        []float64
}

func (g *fakeFrameGrabber) Duration(context.Context, string) (float64, error) {
	g.durations++
	if g.duration <= 0 {
		return 0, errors.New("no duration")
	}
	return g.duration, nil
}

func (g *fakeFrameGrabber) Frame(_ context.Context, path string, at float64) ([]byte, error) {
	i := len(g.at)
	g.at = append(g.at, at)
	if len(g.paths) == 0 || g.paths[len(g.paths)-1] != path {
		g.paths = append(g.paths, path)
	}
	if g.failAt[i] || g.failPath[path] {
		return nil, errors.New("decode error")
	}
	return g.frame, nil
}

type fakeVision struct {
	configured bool
	reply      string   // every call's reply, unless replies has one for it
	replies    []string // per-call replies, in order
	err        error
	calls      int
	req        ai.ImageCompletionRequest
	thinkOff   bool
	model      string
}

func (v *fakeVision) IsConfigured(context.Context) bool { return v.configured }

// CompleteImageWithUsage meters like the real provider: the usage lands on
// the ctx's Budget priced for the ctx's model.
func (v *fakeVision) CompleteImageWithUsage(ctx context.Context, req ai.ImageCompletionRequest) (ai.CompletionResult, error) {
	v.calls++
	v.req, v.thinkOff, v.model = req, ai.ThinkingOffFromContext(ctx), ai.ModelIDFromContext(ctx)
	ai.BudgetFromContext(ctx).RecordLLM(v.model, 520, 50)
	if v.err != nil {
		return ai.CompletionResult{}, v.err
	}
	if v.calls <= len(v.replies) {
		return ai.CompletionResult{Text: v.replies[v.calls-1]}, nil
	}
	return ai.CompletionResult{Text: v.reply}, nil
}

type fakeBurnedStore struct {
	episodes []models.Episode
	writes   []string
	marks    int // undecided checks stamped
}

func (s *fakeBurnedStore) FindBySeasonNumber(context.Context, string, int) ([]models.Episode, error) {
	return s.episodes, nil
}

func (s *fakeBurnedStore) SetSeasonBurnedInSubtitle(_ context.Context, _ string, _ int, value string, _ time.Time) (int64, error) {
	s.writes = append(s.writes, value)
	return int64(len(s.episodes)), nil
}

func (s *fakeBurnedStore) MarkSeasonBurnedInChecked(context.Context, string, int, time.Time) (int64, error) {
	s.marks++
	return 1, nil
}

type fakeSpend struct{ rows []models.AISideSpend }

func (s *fakeSpend) Create(_ context.Context, row *models.AISideSpend) error {
	s.rows = append(s.rows, *row)
	return nil
}

// burnedConverter models OpenCC on the test lines.
type burnedConverter struct{ err error }

var (
	burnedS2T = strings.NewReplacer("这", "這", "个", "個", "们", "們", "说", "說", "静", "靜", "师", "師", "门", "門")
	burnedT2S = strings.NewReplacer("這", "这", "個", "个", "們", "们", "說", "说", "靜", "静", "師", "师", "門", "门")
)

func (c burnedConverter) ConvertS2TWP(b []byte) ([]byte, error) { return c.ConvertS2TW(b) }
func (c burnedConverter) ConvertS2TW(b []byte) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []byte(burnedS2T.Replace(string(b))), nil
}
func (c burnedConverter) ConvertT2S(b []byte) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return []byte(burnedT2S.Replace(string(b))), nil
}

func tracklessEpisode(id string, n int) models.Episode {
	return models.Episode{
		ID: id, SeriesID: "s", SeasonNumber: 1, EpisodeNumber: n,
		FilePath:        models.NewNullString("/tv/s/" + id + ".mkv"),
		SubtitleTracks:  models.NewNullString("[]"),
		DurationSeconds: models.NewNullInt64(2400),
	}
}

type burnedHarness struct {
	store  *fakeBurnedStore
	vision *fakeVision
	frames *fakeFrameGrabber
	spend  *fakeSpend
	svc    *BurnedInSubtitleService
}

func newBurnedHarness(t *testing.T, reply string) *burnedHarness {
	h := &burnedHarness{
		store: &fakeBurnedStore{episodes: []models.Episode{
			{ID: "e1", EpisodeNumber: 1, FilePath: models.NewNullString("/tv/s/e1.mkv"), SubtitleTracks: models.NewNullString(`[{"language":"eng"}]`)},
			{ID: "e2", EpisodeNumber: 2, SubtitleTracks: models.NewNullString("[]")}, // no file
			tracklessEpisode("e3", 3),
			tracklessEpisode("e4", 4),
		}},
		vision: &fakeVision{configured: true, reply: reply},
		frames: &fakeFrameGrabber{frame: testFrame(t), duration: 3000},
		spend:  &fakeSpend{},
	}
	h.svc = NewBurnedInSubtitleService(h.store, h.vision, h.frames, burnedConverter{}, h.spend, nil)
	h.svc.fileExists = func(string) bool { return true } // the fixture's paths are not on disk
	return h
}

// --- frame stacking ------------------------------------------------------------

func TestStackFrameStrips_KeepsTheBottomOfEachFrame(t *testing.T) {
	f := testFrame(t)
	out, err := stackFrameStrips([][]byte{f, f, f, f, f, f})
	require.NoError(t, err)

	img, err := png.Decode(bytes.NewReader(out))
	require.NoError(t, err)
	assert.Equal(t, 480, img.Bounds().Dx())
	assert.Equal(t, 6*86, img.Bounds().Dy(), "six strips of round(270×0.32)")
	for _, y := range []int{2, 86 + 43, 6*86 - 2} {
		r, g, b, _ := img.At(240, y).RGBA()
		assert.Greater(t, b, r+g, "y=%d is from the blue bottom strip, not the red picture", y)
	}
}

func TestStackFrameStrips_FewerThanThreeFramesIsNoVerdict(t *testing.T) {
	f := testFrame(t)
	_, err := stackFrameStrips([][]byte{f, f})
	assert.ErrorIs(t, err, ErrTooFewFrames)

	_, err = stackFrameStrips([][]byte{f, f, []byte("not an image"), {}, []byte("x"), nil})
	assert.ErrorIs(t, err, ErrTooFewFrames, "undecodable frames do not count")

	_, err = stackFrameStrips([][]byte{f, f, f})
	assert.NoError(t, err, "three is enough")
}

// --- reply parsing ---------------------------------------------------------------

func TestParseBurnedInReply(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		ok     bool
		burned bool
		lines  []string
	}{
		{"exact JSON", `{"burned_chinese": true, "lines": ["这个", "他们说"]}`, true, true, []string{"这个", "他们说"}},
		{"not burned", `{"burned_chinese": false, "lines": []}`, true, false, []string{}},
		{"wrapped in prose", "Here is the answer:\n{\"burned_chinese\": true, \"lines\": [\"師門\"]}\nHope that helps.", true, true, []string{"師門"}},
		{"code fence", "```json\n{\"burned_chinese\": false, \"lines\": []}\n```", true, false, []string{}},
		{"lines missing", `{"burned_chinese": true}`, true, true, nil},
		{"burned_chinese missing", `{"lines": ["这个"]}`, false, false, nil},
		{"garbage", "I cannot read this image.", false, false, nil},
		{"broken JSON", `{"burned_chinese": tru`, false, false, nil},
		{"wrong type", `{"burned_chinese": "yes", "lines": []}`, false, false, nil},
		{"empty", "", false, false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			burned, lines, ok := parseBurnedInReply(tt.text)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.burned, burned)
			assert.Equal(t, tt.lines, lines)
		})
	}
}

// --- CheckSeason --------------------------------------------------------------------

func TestCheckSeason_SimplifiedBurnedIn(t *testing.T) {
	h := newBurnedHarness(t, `{"burned_chinese": true, "lines": ["这个人", "他们说的"]}`)

	res, err := h.svc.CheckSeason(context.Background(), "s", 1)
	require.NoError(t, err)
	require.NotNil(t, res.Result)
	assert.Equal(t, models.BurnedInZhHans, *res.Result)
	assert.Equal(t, []string{"e3"}, res.CheckedEpisodeIDs, "first episode with a file and no tracks; burned needs no second look")
	assert.Equal(t, int64(4), res.EpisodesUpdated)
	assert.Equal(t, []string{models.BurnedInZhHans}, h.store.writes)

	// One image, the exam's prompt, thinking off, the exam's model.
	assert.Equal(t, 1, h.vision.calls)
	assert.Equal(t, "image/png", h.vision.req.MediaType)
	assert.Equal(t, burnedInPrompt, h.vision.req.Prompt)
	assert.GreaterOrEqual(t, h.vision.req.MaxTokens, 400)
	assert.True(t, h.vision.thinkOff)
	assert.Equal(t, "claude-sonnet-5", h.vision.model)
	_, err = png.Decode(bytes.NewReader(h.vision.req.Image))
	require.NoError(t, err)

	// Six frames at 15–90% of the stored duration; ffprobe not needed.
	assert.Equal(t, []float64{360, 720, 1080, 1440, 1800, 2160}, h.frames.at)
	assert.Zero(t, h.frames.durations)

	// Spend: one row, priced from the real Sonnet 5 rates (520 in / 50 out).
	require.Len(t, h.spend.rows, 1)
	row := h.spend.rows[0]
	assert.Equal(t, models.AISideSpendKindBurnedInCheck, row.Kind)
	assert.Equal(t, "claude", row.Provider)
	assert.Equal(t, "claude-sonnet-5", row.ModelID)
	assert.Equal(t, int64(520), row.InputTokens)
	assert.Equal(t, int64(50), row.OutputTokens)
	assert.InDelta(t, 0.00231, row.SpentUSD, 1e-9)
	assert.InDelta(t, 0.00231, res.CostUSD, 1e-9)
}

func TestCheckSeason_ScriptVerdicts(t *testing.T) {
	tests := []struct {
		name  string
		reply string
		want  string
	}{
		{"traditional with one stray simplified char (《白髮》)", `{"burned_chinese": true, "lines": ["師門們說", "此處很静"]}`, models.BurnedInZhHant},
		{"no distinguishing characters", `{"burned_chinese": true, "lines": ["你好", "我在山上"]}`, models.BurnedInZh},
		{"burned but no lines copied", `{"burned_chinese": true, "lines": []}`, models.BurnedInZh},
		{"not burned", `{"burned_chinese": false, "lines": []}`, models.BurnedInNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBurnedHarness(t, tt.reply)
			res, err := h.svc.CheckSeason(context.Background(), "s", 1)
			require.NoError(t, err)
			require.NotNil(t, res.Result)
			assert.Equal(t, tt.want, *res.Result)
			assert.Equal(t, []string{tt.want}, h.store.writes)
		})
	}
}

func TestCheckSeason_UndecidedWritesNothing(t *testing.T) {
	t.Run("fewer than three frames", func(t *testing.T) {
		h := newBurnedHarness(t, `{"burned_chinese": false, "lines": []}`)
		h.frames.failAt = map[int]bool{0: true, 1: true, 2: true, 3: true}
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonFramesUnreadable, res.Reason)
		assert.Zero(t, h.vision.calls, "no image, no paid call")
		assert.Empty(t, h.store.writes)
		assert.Empty(t, h.spend.rows)
	})
	t.Run("unreadable reply is billed but not stored as none", func(t *testing.T) {
		h := newBurnedHarness(t, "Sorry, I can't tell.")
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonReplyUnreadable, res.Reason)
		assert.Empty(t, h.store.writes)
		assert.Len(t, h.spend.rows, 1, "the call was billed")
		assert.Positive(t, res.CostUSD)
	})
	t.Run("empty reply", func(t *testing.T) {
		h := newBurnedHarness(t, "")
		h.vision.err = fmt.Errorf("%w: %w", ai.ErrAIInvalidResponse, ai.ErrAIEmptyReply)
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonReplyUnreadable, res.Reason)
		assert.Empty(t, h.store.writes)
		assert.Len(t, h.spend.rows, 1)
	})
	t.Run("OpenCC fails", func(t *testing.T) {
		h := newBurnedHarness(t, `{"burned_chinese": true, "lines": ["这个"]}`)
		h.svc.conv = burnedConverter{err: errors.New("opencc missing")}
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonScriptUnreadable, res.Reason)
		assert.Empty(t, h.store.writes)
	})
	t.Run("no episode without tracks", func(t *testing.T) {
		h := newBurnedHarness(t, "")
		h.store.episodes = h.store.episodes[:2]
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonNoEpisode, res.Reason)
		assert.Zero(t, h.vision.calls)
		assert.Empty(t, h.frames.at)
	})
}

func TestCheckSeason_Errors(t *testing.T) {
	t.Run("no Claude key: nothing is grabbed or sent", func(t *testing.T) {
		h := newBurnedHarness(t, "")
		h.vision.configured = false
		_, err := h.svc.CheckSeason(context.Background(), "s", 1)
		assert.ErrorIs(t, err, ErrBurnedInNotConfigured)
		assert.ErrorIs(t, err, ai.ErrAINotConfigured)
		assert.Empty(t, h.frames.at)
		assert.Zero(t, h.vision.calls)
	})
	t.Run("model call fails outright", func(t *testing.T) {
		h := newBurnedHarness(t, "")
		h.vision.err = fmt.Errorf("%w: 529 overloaded", ai.ErrAIProviderError)
		_, err := h.svc.CheckSeason(context.Background(), "s", 1)
		assert.ErrorIs(t, err, ErrBurnedInModelCall)
		assert.Empty(t, h.store.writes)
	})
}

func TestCheckSeason_ProbesDurationWhenNotStored(t *testing.T) {
	h := newBurnedHarness(t, `{"burned_chinese": true, "lines": ["你好"]}`)
	h.store.episodes[2].DurationSeconds = models.NullInt64{}
	_, err := h.svc.CheckSeason(context.Background(), "s", 1)
	require.NoError(t, err)
	assert.Equal(t, 1, h.frames.durations)
	assert.Equal(t, []float64{450, 900, 1350, 1800, 2250, 2700}, h.frames.at)

	h = newBurnedHarness(t, "")
	h.store.episodes[2].DurationSeconds = models.NullInt64{}
	h.frames.duration = 0
	res, err := h.svc.CheckSeason(context.Background(), "s", 1)
	require.NoError(t, err)
	assert.Equal(t, BurnedInReasonFramesUnreadable, res.Reason, "no duration → no frames → not judged")
}

// --- "none" needs a second opinion -----------------------------------------------

const (
	replyNone     = `{"burned_chinese": false, "lines": []}`
	replyHant     = `{"burned_chinese": true, "lines": ["師門們說", "此處很静"]}`
	replyGarbage  = "I cannot tell from these strips."
	fiveTrackless = 5
)

// newSecondOpinionHarness is a season of five trackless episodes, e1…e5.
func newSecondOpinionHarness(t *testing.T, replies ...string) *burnedHarness {
	h := newBurnedHarness(t, "")
	h.store.episodes = nil
	for i := 1; i <= fiveTrackless; i++ {
		h.store.episodes = append(h.store.episodes, tracklessEpisode(fmt.Sprintf("e%d", i), i))
	}
	h.vision.replies = replies
	return h
}

func TestCheckSeason_NoneNeedsASecondOpinion(t *testing.T) {
	t.Run("first none, second burned → the second's script is written", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyNone, replyHant)
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		require.NotNil(t, res.Result)
		assert.Equal(t, models.BurnedInZhHant, *res.Result)
		assert.Equal(t, []string{models.BurnedInZhHant}, h.store.writes)
		assert.Equal(t, []string{"e1", "e3"}, res.CheckedEpisodeIDs, "the second look is the middle episode, not the next one")
		assert.Equal(t, []string{"/tv/s/e1.mkv", "/tv/s/e3.mkv"}, h.frames.paths)
	})
	t.Run("both none → none", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyNone, replyNone)
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		require.NotNil(t, res.Result)
		assert.Equal(t, models.BurnedInNone, *res.Result)
		assert.Equal(t, []string{models.BurnedInNone}, h.store.writes)
		assert.Equal(t, []string{"e1", "e3"}, res.CheckedEpisodeIDs)
	})
	t.Run("first none, second unreadable → nothing written", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyNone, replyGarbage)
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonReplyUnreadable, res.Reason)
		assert.Empty(t, h.store.writes)
		assert.Equal(t, []string{"e1", "e3"}, res.CheckedEpisodeIDs)
	})
	t.Run("first none, second frames unavailable → nothing written", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyNone)
		h.frames.failPath = map[string]bool{"/tv/s/e3.mkv": true}
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonFramesUnreadable, res.Reason)
		assert.Empty(t, h.store.writes)
		assert.Equal(t, 1, h.vision.calls, "no image for the second, no second call")
	})
	t.Run("first none, second OpenCC fails → nothing written", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyNone, replyHant)
		h.svc.conv = burnedConverter{err: errors.New("opencc missing")}
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Nil(t, res.Result)
		assert.Equal(t, BurnedInReasonScriptUnreadable, res.Reason)
		assert.Empty(t, h.store.writes)
	})
	t.Run("two trackless episodes: the second look is the other one", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyNone, replyNone)
		h.store.episodes = h.store.episodes[:2]
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Equal(t, []string{"e1", "e2"}, res.CheckedEpisodeIDs)
	})
	t.Run("a single trackless episode accepts its one none", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyNone)
		h.store.episodes = h.store.episodes[:1]
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		require.NotNil(t, res.Result)
		assert.Equal(t, models.BurnedInNone, *res.Result)
		assert.Equal(t, []string{models.BurnedInNone}, h.store.writes)
		assert.Equal(t, []string{"e1"}, res.CheckedEpisodeIDs)
		assert.Equal(t, 1, h.vision.calls)
	})
	t.Run("first burned → no second look", func(t *testing.T) {
		h := newSecondOpinionHarness(t, replyHant)
		res, err := h.svc.CheckSeason(context.Background(), "s", 1)
		require.NoError(t, err)
		assert.Equal(t, []string{"e1"}, res.CheckedEpisodeIDs)
		assert.Equal(t, 1, h.vision.calls)
	})
}

func TestCheckSeason_SecondOpinionCostsAreSummedAndEachRecorded(t *testing.T) {
	h := newSecondOpinionHarness(t, replyNone, replyGarbage)
	res, err := h.svc.CheckSeason(context.Background(), "s", 1)
	require.NoError(t, err)

	require.Len(t, h.spend.rows, 2, "one ai_side_spend row per model call")
	for _, row := range h.spend.rows {
		assert.Equal(t, models.AISideSpendKindBurnedInCheck, row.Kind)
		assert.InDelta(t, 0.00231, row.SpentUSD, 1e-9)
	}
	assert.InDelta(t, 0.00462, res.CostUSD, 1e-9, "both calls are in cost_usd, even when the season stays unjudged")
}

// AC #5 end to end over the real migration chain: the season's '[]' episodes
// are written, an episode with tracks and the other season are not.
func TestCheckSeason_WritesOnlyTracklessEpisodesOfTheSeason(t *testing.T) {
	db := burnedInLibraryDB(t)
	ctx := context.Background()
	_, err := db.Exec(`INSERT INTO series (id, title, first_air_date) VALUES ('s', 'Show', '2020-01-01')`)
	require.NoError(t, err)
	for _, e := range []struct {
		id      string
		season  int
		episode int
		tracks  interface{}
	}{
		{"s1e1", 1, 1, `[{"language":"eng","format":"subrip","external":false,"stream_index":2}]`},
		{"s1e2", 1, 2, `[]`},
		{"s1e3", 1, 3, `[]`},
		{"s1e4", 1, 4, nil},
		{"s2e1", 2, 1, `[]`},
	} {
		_, err := db.Exec(`INSERT INTO episodes (id, series_id, season_number, episode_number, file_path, duration_seconds, subtitle_status, subtitle_tracks, created_at, updated_at)
			VALUES (?, 's', ?, ?, ?, 2400, 'not_searched', ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
			e.id, e.season, e.episode, "/tv/s/"+e.id+".mkv", e.tracks)
		require.NoError(t, err)
	}

	vision := &fakeVision{configured: true, reply: `{"burned_chinese": true, "lines": ["这个人", "他们说的"]}`}
	svc := NewBurnedInSubtitleService(repository.NewEpisodeRepository(db), vision,
		&fakeFrameGrabber{frame: testFrame(t)}, burnedConverter{}, repository.NewAISideSpendRepository(db), nil)
	svc.fileExists = func(string) bool { return true }
	res, err := svc.CheckSeason(ctx, "s", 1)
	require.NoError(t, err)
	require.NotNil(t, res.Result)
	assert.Equal(t, int64(2), res.EpisodesUpdated)
	assert.Equal(t, []string{"s1e2"}, res.CheckedEpisodeIDs)

	got := map[string]sql.NullString{}
	rows, err := db.Query(`SELECT id, burned_in_subtitle FROM episodes`)
	require.NoError(t, err)
	defer rows.Close()
	for rows.Next() {
		var id string
		var v sql.NullString
		require.NoError(t, rows.Scan(&id, &v))
		got[id] = v
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, "zh-Hans", got["s1e2"].String)
	assert.Equal(t, "zh-Hans", got["s1e3"].String)
	assert.False(t, got["s1e1"].Valid, "an episode with tracks keeps its own answer")
	assert.False(t, got["s1e4"].Valid, "an episode never probed is not written")
	assert.False(t, got["s2e1"].Valid, "another season is not written")

	var kind string
	require.NoError(t, db.QueryRow(`SELECT kind FROM ai_side_spend`).Scan(&kind))
	assert.Equal(t, "burned_in_check", kind)
}

func burnedInLibraryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open(database.DriverName, filepath.Join(t.TempDir(), "burned.db")+"?_pragma=foreign_keys(1)")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	runner, err := migrations.NewRunner(db)
	require.NoError(t, err)
	require.NoError(t, runner.RegisterAll(migrations.GetAll()))
	require.NoError(t, runner.Up(context.Background()))
	return db
}

// The real ffprobe/ffmpeg grabber on a generated clip; skipped where ffmpeg
// is not installed.
func TestFFmpegFrameGrabber_RealClip(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if _, err := exec.LookPath("ffprobe"); err != nil {
		t.Skip("ffprobe not installed")
	}
	clip := filepath.Join(t.TempDir(), "clip.mp4")
	out, err := exec.Command("ffmpeg", "-nostdin", "-v", "error", "-f", "lavfi", "-i", "testsrc=duration=10:size=1280x720:rate=10",
		"-pix_fmt", "yuv420p", "-y", clip).CombinedOutput()
	require.NoError(t, err, string(out))

	g := FFmpegFrameGrabber{}
	ctx := context.Background()
	d, err := g.Duration(ctx, clip)
	require.NoError(t, err)
	assert.InDelta(t, 10, d, 0.5)

	var frames [][]byte
	for _, pct := range burnedInSamplePercents {
		f, err := g.Frame(ctx, clip, d*float64(pct)/100)
		require.NoError(t, err)
		frames = append(frames, f)
	}
	img, _, err := image.Decode(bytes.NewReader(frames[0]))
	require.NoError(t, err)
	assert.Equal(t, burnedInFrameWidth, img.Bounds().Dx())
	assert.Equal(t, 270, img.Bounds().Dy(), "16:9 at 480 wide")

	stacked, err := stackFrameStrips(frames)
	require.NoError(t, err)
	cfg, err := png.DecodeConfig(bytes.NewReader(stacked))
	require.NoError(t, err)
	assert.Equal(t, 480, cfg.Width)
	assert.Equal(t, 6*86, cfg.Height)

	_, err = g.Frame(ctx, filepath.Join(t.TempDir(), "missing.mkv"), 1)
	assert.Error(t, err)
}

// The season list's per-episode verdict reads the stored burned-in value too
// (AC #5: the episode's chinese_subtitle moves, not just the series').
func TestComputeEpisodeSubtitle_ReadsBurnedIn(t *testing.T) {
	ep := tracklessEpisode("e1", 1)
	assert.Equal(t, models.ChineseSubtitleNone, computeEpisodeSubtitle(ep, nil).verdict)
	ep.BurnedInSubtitle = models.NewNullString(models.BurnedInZhHans)
	assert.Equal(t, models.ChineseSubtitleZhHans, computeEpisodeSubtitle(ep, nil).verdict)
}

// disc-2026-11-burned-in-subs-b2 AC #2: a check that could not decide stamps
// burned_in_checked_at (value still NULL) so the sweep waits before paying to
// ask again; a decided check, "no episode", and a failed call do not.
func TestCheckSeason_UndecidedIsStamped(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(h *burnedHarness)
		marked int
	}{
		{"frames unavailable", func(h *burnedHarness) { h.frames.failAt = map[int]bool{0: true, 1: true, 2: true, 3: true} }, 1},
		{"reply unreadable", func(h *burnedHarness) { h.vision.reply = replyGarbage }, 1},
		{"OpenCC fails", func(h *burnedHarness) {
			h.vision.reply = replyHant
			h.svc.conv = burnedConverter{err: errors.New("opencc missing")}
		}, 1},
		{"first none, second unreadable", func(h *burnedHarness) { h.vision.replies = []string{replyNone, replyGarbage} }, 1},
		{"decided", func(h *burnedHarness) { h.vision.reply = replyHant }, 0},
		{"no episode to check", func(h *burnedHarness) { h.store.episodes = h.store.episodes[:2] }, 0},
		{"model call failed", func(h *burnedHarness) { h.vision.err = fmt.Errorf("%w: 529", ai.ErrAIProviderError) }, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newBurnedHarness(t, "")
			tt.setup(h)
			_, _ = h.svc.CheckSeason(context.Background(), "s", 1)
			assert.Equal(t, tt.marked, h.store.marks)
		})
	}
}

func TestCheckSeason_SkipsEpisodesWhoseFileIsGone(t *testing.T) {
	// 《仙逆》S1: the first trackless row pointed at a removed special, and the
	// whole season was left unjudged. A missing file is skipped, not sampled.
	h := newBurnedHarness(t, `{"burned_chinese": true, "lines": ["这个人"]}`)
	h.svc.fileExists = func(path string) bool { return !strings.HasSuffix(path, "e3.mkv") }
	res, err := h.svc.CheckSeason(context.Background(), "s", 1)
	require.NoError(t, err)
	require.NotNil(t, res.Result)
	assert.Equal(t, models.BurnedInZhHans, *res.Result)
	assert.Equal(t, []string{"e4"}, res.CheckedEpisodeIDs, "e3's file is gone — e4 is sampled instead")
}

func TestCheckSeason_EveryFileGoneIsNoEpisodeToCheck(t *testing.T) {
	h := newBurnedHarness(t, `{"burned_chinese": true, "lines": ["这个人"]}`)
	h.svc.fileExists = func(string) bool { return false }
	res, err := h.svc.CheckSeason(context.Background(), "s", 1)
	require.NoError(t, err)
	assert.Nil(t, res.Result)
	assert.Equal(t, BurnedInReasonNoEpisode, res.Reason)
	assert.Empty(t, res.CheckedEpisodeIDs)
}

func TestFileOnDisk(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "a.mkv")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))
	assert.True(t, fileOnDisk(f))
	assert.False(t, fileOnDisk(filepath.Join(dir, "missing.mkv")))
	assert.False(t, fileOnDisk(dir), "a directory is not a video file")
}
