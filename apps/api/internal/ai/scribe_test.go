package ai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scribeTestWAV writes a 120-second 16 kHz mono WAV header (the chunk size
// the pipeline uploads) and returns its path.
func scribeTestWAV(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "scribe-*.wav")
	require.NoError(t, err)
	writeTestWAV(t, f, 120*32000)
	return f.Name()
}

// word / spacing / event build Scribe tokens for the cueing tests.
func word(text string, start, end float64) scribeWord {
	return scribeWord{Text: text, Type: "word", Start: start, End: end}
}
func spacing(at float64) scribeWord {
	return scribeWord{Text: " ", Type: "spacing", Start: at, End: at}
}
func event(text string, start, end float64) scribeWord {
	return scribeWord{Text: text, Type: "audio_event", Start: start, End: end}
}

const scribeOK = `{"language_code":"eng","language_probability":1.0,"text":"Hey. Face me.","words":[
{"text":"Hey.","type":"word","start":1.0,"end":1.3},{"text":" ","type":"spacing","start":1.3,"end":1.3},
{"text":"Face","type":"word","start":2.0,"end":2.4},{"text":" ","type":"spacing","start":2.4,"end":2.4},
{"text":"me.","type":"word","start":2.4,"end":2.9}],"audio_duration_secs":120}`

// ─── Request shape ──────────────────────────────────────────────────────────

func TestScribe_SendsElevenLabsFormAndRendersCues(t *testing.T) {
	var gotKey, gotContentType string
	var fields map[string][]string
	var fileName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("xi-api-key")
		gotContentType = r.Header.Get("Content-Type")
		require.NoError(t, r.ParseMultipartForm(1<<20))
		fields = r.MultipartForm.Value
		if fh := r.MultipartForm.File["file"]; len(fh) == 1 {
			fileName = fh[0].Filename
		}
		_, _ = io.WriteString(w, scribeOK)
	}))
	defer srv.Close()

	c := NewScribeClient(" el-key ", WithScribeBaseURL(srv.URL))
	wav := scribeTestWAV(t)
	srt, err := c.TranscribeWithLanguage(context.Background(), wav, "en")
	require.NoError(t, err)

	assert.Equal(t, "el-key", gotKey, "ElevenLabs authenticates with xi-api-key, not Bearer")
	assert.Empty(t, fields["Authorization"])
	assert.Contains(t, gotContentType, "multipart/form-data")
	assert.Equal(t, filepath.Base(wav), fileName)
	assert.Equal(t, []string{"scribe_v2"}, fields["model_id"])
	assert.Equal(t, []string{"word"}, fields["timestamps_granularity"])
	assert.Equal(t, []string{"true"}, fields["tag_audio_events"])
	assert.Equal(t, []string{"en"}, fields["language_code"])
	for _, never := range []string{"diarize", "additional_formats", "keyterms"} {
		assert.NotContains(t, fields, never, "%s must not be sent (surcharge / rejected on the exam)", never)
	}

	// "Hey." ends a sentence; "Face me." is the next cue. One-word cues are
	// held on screen for scribeMinCueSeconds but never into the next cue.
	assert.Equal(t,
		"1\n00:00:01,000 --> 00:00:01,600\nHey.\n\n"+
			"2\n00:00:02,000 --> 00:00:02,900\nFace me.\n\n",
		srt)
}

func TestScribe_OmitsLanguageWhenUnknown(t *testing.T) {
	var fields map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(1<<20))
		fields = r.MultipartForm.Value
		_, _ = io.WriteString(w, scribeOK)
	}))
	defer srv.Close()

	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	require.NoError(t, err)
	assert.NotContains(t, fields, "language_code", "an unknown track language lets Scribe detect it")
}

func TestScribe_MissingKeyFailsBeforeAnyCall(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer srv.Close()
	_, err := NewScribeClient("  ", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	assert.ErrorIs(t, err, ErrScribeNotConfigured)
	assert.Zero(t, calls.Load())
}

func TestScribe_EndpointFingerprintNamesProviderAndModel(t *testing.T) {
	assert.Equal(t, "elevenlabs|scribe_v2", NewScribeClient("k").EndpointFingerprint())
	assert.Equal(t, "elevenlabs|scribe_v3", NewScribeClient("k", WithScribeModel("scribe_v3")).EndpointFingerprint())
	assert.NotEqual(t, "|whisper-1", NewScribeClient("k").EndpointFingerprint(),
		"a chunk Whisper transcribed must never be served back as Scribe's")
}

// ─── Errors ─────────────────────────────────────────────────────────────────

func TestScribe_ValidationErrorFailsFastQuotingTheServer(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = io.WriteString(w, `{"detail":[{"loc":["body","model_id"],"msg":"Input should be 'scribe_v2'","type":"enum"}]}`)
	}))
	defer srv.Close()

	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	require.ErrorIs(t, err, ErrScribeAPIError)
	assert.Contains(t, err.Error(), "Input should be 'scribe_v2'")
	assert.Equal(t, int32(1), calls.Load(), "a 422 must not retry")
}

func TestScribe_BadRequestQuotesStatusAndMessage(t *testing.T) {
	shrinkRetryDelays(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		// Real 400 body recorded 2026-10-14 (additional_formats without diarize).
		_, _ = io.WriteString(w, `{"detail":{"type":"validation_error","code":"invalid_parameters","message":"Requesting additional formats must have diarization and timestamps enabled.","status":"invalid_parameters","request_id":"2d42","param":"additional_formats"}}`)
	}))
	defer srv.Close()
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	require.ErrorIs(t, err, ErrScribeAPIError)
	assert.Contains(t, err.Error(), "invalid_parameters: Requesting additional formats")
}

func TestScribe_RejectedKeyFailsFast(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"detail":{"status":"invalid_api_key","message":"Invalid API key"}}`)
	}))
	defer srv.Close()
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	require.ErrorIs(t, err, ErrScribeAPIError)
	assert.NotErrorIs(t, err, ErrAIQuotaExceeded, "a bad key is not an empty account")
	assert.Equal(t, int32(1), calls.Load())
}

func TestScribe_ServerErrorRetriesThenSucceeds(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = io.WriteString(w, scribeOK)
	}))
	defer srv.Close()
	srt, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	require.NoError(t, err)
	assert.Contains(t, srt, "Face me.")
	assert.Equal(t, int32(3), calls.Load())
}

func TestScribe_RateLimitRetriesToExhaustion(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = io.WriteString(w, `{"detail":{"status":"too_many_concurrent_requests","message":"slow down"}}`)
	}))
	defer srv.Close()
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	assert.ErrorIs(t, err, ErrAIRetriesExhausted)
	assert.Equal(t, int32(retryMaxAttempts), calls.Load())
}

func TestScribe_CreditExhaustedStopsWithoutRetry(t *testing.T) {
	shrinkRetryDelays(t)
	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"quota_exceeded status in body": {http.StatusUnauthorized, `{"detail":{"status":"quota_exceeded","message":"This request exceeds your quota of 16200 remaining characters."}}`},
		"402 payment required":          {http.StatusPaymentRequired, `{"detail":"payment required"}`},
	} {
		t.Run(name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
			require.ErrorIs(t, err, ErrAIQuotaExceeded)
			var credit *CreditExhaustedError
			require.ErrorAs(t, err, &credit)
			assert.Equal(t, "elevenlabs", credit.Provider)
			assert.Equal(t, "ElevenLabs", credit.DisplayName())
			assert.Equal(t, ElevenLabsBillingURL, credit.BillingURL)
			assert.Equal(t, int32(1), calls.Load(), "waiting will not refill an empty account")
		})
	}
}

func TestScribe_PerAttemptTimeoutIsTransient(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL), WithScribeTimeout(20*time.Millisecond)).
		Transcribe(context.Background(), scribeTestWAV(t))
	assert.ErrorIs(t, err, ErrScribeTimeout)
	assert.ErrorIs(t, err, ErrAIRetriesExhausted)
	assert.Equal(t, int32(retryMaxAttempts), calls.Load())
}

func TestScribe_CallerCancelStopsWithoutRetry(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer srv.Close()
	defer close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(ctx, scribeTestWAV(t))
	assert.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), calls.Load())
}

func TestScribe_MalformedReplyIsInvalidResponse(t *testing.T) {
	for name, body := range map[string]string{
		"not json":            `<html>oops</html>`,
		"transcript no words": `{"text":"Hello there."}`,
		"empty object":        `{}`,
	} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, body)
			}))
			defer srv.Close()
			_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
			assert.ErrorIs(t, err, ErrAIInvalidResponse)
		})
	}
}

func TestScribe_SilenceIsAnEmptyFilteredDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"language_code":"eng","text":"","words":[{"text":"[dramatic music]","type":"audio_event","start":0,"end":119.5}],"audio_duration_secs":120}`)
	}))
	defer srv.Close()
	detail, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).TranscribeDetailed(context.Background(), scribeTestWAV(t), "en")
	require.NoError(t, err)
	assert.Empty(t, detail.SRT)
	assert.True(t, detail.Filtered)
	assert.Zero(t, detail.SegmentsIn)
}

// ─── Budget ─────────────────────────────────────────────────────────────────

func TestScribe_MetersAudioSecondsAtTheScribeRate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, scribeOK) // audio_duration_secs: 120
	}))
	defer srv.Close()
	b := NewBudget(1)
	ctx := WithBudget(context.Background(), b)
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(ctx, scribeTestWAV(t))
	require.NoError(t, err)
	snap := b.Snapshot()
	// 2 minutes × $0.22/60 = $0.007333…
	assert.InDelta(t, 0.22/60*2, snap.SpentUSD, 1e-9)
	assert.InDelta(t, 120, snap.ASRSeconds, 1e-9)
	assert.Equal(t, 1, snap.ASRCalls)
	assert.Less(t, snap.SpentUSD, 0.012, "cheaper than the same two minutes on Whisper")
}

func TestScribe_ExhaustedBudgetSkipsTheCall(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1) }))
	defer srv.Close()
	b := NewBudget(0.001)
	b.RecordASRWithRate(600, ScribePerMinuteUSD())
	require.True(t, b.Exceeded())
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(WithBudget(context.Background(), b), scribeTestWAV(t))
	assert.ErrorIs(t, err, ErrBudgetExceeded)
	assert.Zero(t, calls.Load())
}

func TestScribePerMinute_IsExactly22CentsAnHour(t *testing.T) {
	assert.Equal(t, "0.22", ScribePerMinute().Mul(decimal.NewFromInt(60)).Round(8).String())
	assert.InDelta(t, 0.22/60, ScribePerMinuteUSD(), 1e-12)
}

// ─── Cueing (pure) ──────────────────────────────────────────────────────────

func TestScribeCues_SplitsOnSentenceEndPauseAndEvent(t *testing.T) {
	segs := scribeCues([]scribeWord{
		word("Who's", 1.0, 1.2), spacing(1.2), word("there?", 1.2, 1.5),
		spacing(1.5), word("Answer", 1.6, 1.9), spacing(1.9), word("me,", 1.9, 2.1),
		spacing(2.1), word("who's", 2.2, 2.4), spacing(2.4), word("there?", 2.4, 2.7),
		spacing(2.7), event("[bear roaring]", 2.7, 4.0),
		word("Hey", 5.0, 5.2), spacing(5.2), word("face", 5.3, 5.5), spacing(5.5), word("me", 5.5, 5.8),
		spacing(5.8), word("Come", 7.0, 7.2), spacing(7.2), word("to", 7.2, 7.3), spacing(7.3), word("me", 7.3, 7.6),
	})
	texts := make([]string, len(segs))
	for i, s := range segs {
		texts[i] = s.Text
	}
	assert.Equal(t, []string{"Who's there?", "Answer me, who's there?", "Hey face me", "Come to me"}, texts)
	// 'Hey face me' (5.0–5.8) and 'Come to me' (7.0–7.6): a 1.2 s pause split them.
	assert.InDelta(t, 5.0, segs[2].Start, 1e-9)
	assert.InDelta(t, 5.8, segs[2].End, 1e-9)
	assert.InDelta(t, 7.0, segs[3].Start, 1e-9)
	for _, s := range segs {
		assert.NotContains(t, s.Text, "[", "sound tags never reach a cue")
		assert.Zero(t, s.NoSpeechProb)
	}
}

func TestScribeCues_CapsCueLengthAndWidth(t *testing.T) {
	// 20 words, no punctuation, no pauses: must be split by time (7 s) and width (84 chars).
	var ws []scribeWord
	for i := 0; i < 20; i++ {
		t0 := float64(i) * 0.5
		ws = append(ws, word("wordy"+strings.Repeat("x", 4), t0, t0+0.4), spacing(t0+0.4))
	}
	segs := scribeCues(ws)
	require.Greater(t, len(segs), 1)
	for _, s := range segs {
		assert.LessOrEqual(t, s.End-s.Start, scribeCueMaxSeconds+1e-9)
		assert.LessOrEqual(t, len([]rune(s.Text)), scribeCueMaxChars)
	}
	// Every word survives the split.
	joined := 0
	for _, s := range segs {
		joined += len(strings.Fields(s.Text))
	}
	assert.Equal(t, 20, joined)
}

func TestScribeCues_ShortCueIsHeldButNeverIntoTheNext(t *testing.T) {
	segs := scribeCues([]scribeWord{
		word("No!", 1.0, 1.1),
		word("Hey!", 1.3, 1.5), // only 0.2 s later: the held "No!" must stop at 1.3
		word("Yes.", 9.0, 9.1), // last cue: free to extend to 9.6
	})
	require.Len(t, segs, 3)
	assert.InDelta(t, 1.3, segs[0].End, 1e-9)
	assert.InDelta(t, 1.9, segs[1].End, 1e-9) // 1.3 + 0.6; the next cue is far away
	assert.InDelta(t, 9.6, segs[2].End, 1e-9)
}

func TestScribeCues_CJKWordsJoinWithoutSpaces(t *testing.T) {
	// Scribe returns Chinese one character per token with NO spacing tokens
	// between them; English words have a spacing token between each pair.
	segs := scribeCues([]scribeWord{
		word("我", 1.0, 1.1), word("們", 1.1, 1.2), word("走", 1.2, 1.3), word("吧", 1.3, 1.4), word("。", 1.4, 1.4),
		word("Hey", 3.0, 3.2), spacing(3.2), word("Joe.", 3.2, 3.5),
	})
	require.Len(t, segs, 2)
	assert.Equal(t, "我們走吧。", segs[0].Text)
	assert.Equal(t, "Hey Joe.", segs[1].Text)
}

func TestScribeCues_CJKWidthCountsDouble(t *testing.T) {
	// 50 漢字 with no pause or punctuation: two lines hold 42, so this must split.
	var ws []scribeWord
	for i := 0; i < 50; i++ {
		t0 := 1.0 + float64(i)*0.1
		ws = append(ws, word("字", t0, t0+0.1))
	}
	segs := scribeCues(ws)
	require.Greater(t, len(segs), 1)
	for _, s := range segs {
		assert.LessOrEqual(t, len([]rune(s.Text)), scribeCueMaxChars/2)
	}
}

func TestScribeCues_OneStretchedTokenIsCappedAtMaxSeconds(t *testing.T) {
	segs := scribeCues([]scribeWord{word("Aaaaah!", 10.0, 25.0)})
	require.Len(t, segs, 1)
	assert.InDelta(t, 10.0, segs[0].Start, 1e-9)
	assert.InDelta(t, 10.0+scribeCueMaxSeconds, segs[0].End, 1e-9)
}

func TestEndsSentence_AbbreviationsAndInitials(t *testing.T) {
	for w, want := range map[string]bool{
		"Mr.": false, "Dr.": false, "U.S.": false, "J.": false, "p.m.": false,
		"done.": true, "What?": true, "Hey!": true, "吧。": true, "Fine.\"": true, "stop…": true, "no": false,
	} {
		assert.Equal(t, want, endsSentence(w), w)
	}
}

func TestScribe_ServerErrorMentioningPaymentStillRetries(t *testing.T) {
	shrinkRetryDelays(t)
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) < 2 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = io.WriteString(w, `<html>upstream payment_required gateway</html>`)
			return
		}
		_, _ = io.WriteString(w, scribeOK)
	}))
	defer srv.Close()
	_, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).Transcribe(context.Background(), scribeTestWAV(t))
	require.NoError(t, err, "a 5xx is never an empty account")
	assert.Equal(t, int32(2), calls.Load())
}

func TestScribeCues_EmptyAndEventsOnly(t *testing.T) {
	assert.Empty(t, scribeCues(nil))
	assert.Empty(t, scribeCues([]scribeWord{event("[music]", 0, 10), spacing(10)}))
}

// ─── Rule 28: the response recorded from the real API ───────────────────────

// TestScribe_RealResponseShape replays the 10-minute See S01E02 transcript
// ElevenLabs returned on 2026-10-14 (eval/asr-exam-2026-10/runs/scribe-v2-1.json)
// and checks the cues this client makes of it against what the exam's answer
// key (the official English track) says is there.
func TestScribe_RealResponseShape(t *testing.T) {
	body, err := os.ReadFile("testdata/scribe_v2_ok.json")
	require.NoError(t, err)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	b := NewBudget(5)
	ctx := WithBudget(context.Background(), b)
	detail, err := NewScribeClient("k", WithScribeBaseURL(srv.URL)).TranscribeDetailed(ctx, scribeTestWAV(t), "en")
	require.NoError(t, err)
	require.True(t, detail.Filtered)

	cues := parseSRTCues(t, detail.SRT)
	require.NotEmpty(t, cues)

	// Billed on Scribe's own duration figure (599.83 s ≈ 10 min → $0.0367),
	// not the 120 s WAV header.
	assert.InDelta(t, 599.83/60*(0.22/60), b.SpentUSD(), 1e-5)

	// The first four minutes are score only: the first cue is at 228.8 s.
	assert.Greater(t, cues[0].start, 228.0)
	assert.Less(t, cues[0].start, 230.0)

	// The bear fight (261–280 s): the official track has six shouted lines
	// there and Whisper heard none of them. Scribe hears them, and no sound
	// tag leaks into a cue.
	fight := 0
	for i, c := range cues {
		if c.start >= 260 && c.start <= 280 {
			fight++
		}
		assert.NotContains(t, c.text, "[", "cue %q carries a sound tag", c.text)
		assert.LessOrEqual(t, c.end-c.start, scribeCueMaxSeconds+1e-6, "cue %q runs too long", c.text)
		// A cue may be shorter than the minimum ONLY because the next cue
		// starts right behind it — holding it longer would overlap.
		if c.end-c.start < scribeMinCueSeconds-1e-6 {
			require.Less(t, i+1, len(cues), "last cue %q was not held on screen", c.text)
			assert.InDelta(t, cues[i+1].start, c.end, 1e-6, "cue %q flashes by with free time after it", c.text)
		}
	}
	assert.GreaterOrEqual(t, fight, 8, "the bear fight is where whisper-1 failed the exam")

	// One official line, three sentences, three cues.
	var around []string
	for _, c := range cues {
		if c.start >= 321 && c.start < 325 {
			around = append(around, c.text)
		}
	}
	assert.Equal(t, []string{"Who's there?", "Answer me.", "Who's there?"}, around)

	// Nothing invented in the score: no cue between the cry at 231 s and the fight at 261 s.
	for _, c := range cues {
		assert.False(t, c.start > 232 && c.start < 260, "cue %q sits in the silent stretch", c.text)
	}
}

type srtCue struct {
	start, end float64
	text       string
}

func parseSRTCues(t *testing.T, srt string) []srtCue {
	t.Helper()
	var out []srtCue
	for _, block := range strings.Split(strings.TrimSpace(srt), "\n\n") {
		lines := strings.Split(block, "\n")
		require.GreaterOrEqual(t, len(lines), 3, "block %q", block)
		times := strings.Split(lines[1], " --> ")
		require.Len(t, times, 2)
		out = append(out, srtCue{start: srtSeconds(t, times[0]), end: srtSeconds(t, times[1]), text: strings.Join(lines[2:], " ")})
	}
	return out
}

func srtSeconds(t *testing.T, ts string) float64 {
	t.Helper()
	var h, m, s, ms int
	n, err := fmtSscanf(ts, &h, &m, &s, &ms)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	return float64(h*3600+m*60+s) + float64(ms)/1000
}

var errBadTimestamp = errors.New("bad timestamp")

func fmtSscanf(ts string, h, m, s, ms *int) (int, error) {
	parts := strings.FieldsFunc(ts, func(r rune) bool { return r == ':' || r == ',' })
	if len(parts) != 4 {
		return 0, errBadTimestamp
	}
	for i, p := range []*int{h, m, s, ms} {
		v := 0
		for _, ch := range parts[i] {
			if ch < '0' || ch > '9' {
				return i, errBadTimestamp
			}
			v = v*10 + int(ch-'0')
		}
		*p = v
	}
	return 4, nil
}

// disc-2026-10-scribe-keyterms: names on the ctx go out as repeated
// `keyterms` fields and the request is billed with the 20% surcharge; a ctx
// without names sends none and bills the base rate.
func TestScribe_SendsKeytermsFromContextAndBillsTheSurcharge(t *testing.T) {
	var fields map[string][]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(1<<20))
		fields = r.MultipartForm.Value
		_, _ = io.WriteString(w, scribeOK)
	}))
	defer srv.Close()
	c := NewScribeClient("el-key", WithScribeBaseURL(srv.URL))
	wav := scribeTestWAV(t)

	b := NewBudget(10)
	ctx := WithBudget(WithASRKeyterms(context.Background(), []string{"Jerlamarel", "Baba Voss"}), b)
	_, err := c.TranscribeDetailed(ctx, wav, "en")
	require.NoError(t, err)
	assert.Equal(t, []string{"Jerlamarel", "Baba Voss"}, fields["keyterms"], "one repeated field per name")
	assert.InDelta(t, 120*ScribePerMinuteWithKeyterms().InexactFloat64()/60, b.SpentUSD(), 1e-9, "120 s of audio at the surcharged rate")

	b2 := NewBudget(10)
	_, err = c.TranscribeDetailed(WithBudget(context.Background(), b2), wav, "en")
	require.NoError(t, err)
	assert.NotContains(t, fields, "keyterms")
	assert.InDelta(t, 120*ScribePerMinuteUSD()/60, b2.SpentUSD(), 1e-9, "no names, base rate")
}

func TestNormalizeScribeKeyterms_AppliesElevenLabsRulesAndTheCap(t *testing.T) {
	got := NormalizeScribeKeyterms([]string{
		" Jerlamarel ", "jerlamarel", "Baba <Voss>", "",
		"a name with six whole words here", strings.Repeat("x", 50), "Tamacti Jun",
	})
	assert.Equal(t, []string{"Jerlamarel", "Baba Voss", "Tamacti Jun"}, got)

	many := make([]string, 0, ScribeMaxKeyterms+20)
	for i := 0; i < ScribeMaxKeyterms+20; i++ {
		many = append(many, fmt.Sprintf("Name%d", i))
	}
	assert.Len(t, NormalizeScribeKeyterms(many), ScribeMaxKeyterms, "past 100 every request bills 20 s minimum")
	assert.Nil(t, NormalizeScribeKeyterms([]string{"", "  "}))
}

func TestScribeRates_SurchargeIsTwentyPercent(t *testing.T) {
	assert.Equal(t, "0.264", ScribePerMinuteWithKeyterms().Mul(decimal.NewFromInt(60)).Round(8).String())
	assert.True(t, ASRPerMinute("elevenlabs").Equal(ScribePerMinuteWithKeyterms()), "the quote includes the surcharge")
}
