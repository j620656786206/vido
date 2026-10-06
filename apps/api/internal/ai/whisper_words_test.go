package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// disc-2026-10-asr-coarse-timestamps — whisper's segment times are whole
// seconds glued end to end; its word times are not.

func TestParseVerboseTranscription_CarriesWords(t *testing.T) {
	vt, err := parseVerboseTranscription(`{"language":"english","duration":9.0,"text":"I just heard Baba Voss",
		"segments":[{"id":0,"start":0.0,"end":9.0,"text":"I just heard Baba Voss","avg_logprob":-0.2,"compression_ratio":1.1,"no_speech_prob":0.01}],
		"words":[{"word":"I","start":6.1,"end":6.2},{"word":"just","start":6.2,"end":6.4},{"word":"heard","start":6.4,"end":6.7},{"word":"Baba","start":6.8,"end":7.1},{"word":"Voss","start":7.1,"end":7.5}]}`)
	require.NoError(t, err)
	require.Len(t, vt.Words, 5)
	assert.Equal(t, "Voss", vt.Words[4].Word)
}

func TestTightenSegmentsWithWords(t *testing.T) {
	t.Run("boundaries move IN to the first and last word; never out", func(t *testing.T) {
		segs := []whisperSegment{
			{Start: 0, End: 9, Text: "I just heard Baba Voss"},
			{Start: 9, End: 15, Text: "calling the witch's name."},
		}
		words := []whisperWord{
			{"I", 6.1, 6.2}, {"just", 6.2, 6.4}, {"heard", 6.4, 6.7}, {"Baba", 6.8, 7.1}, {"Voss", 7.1, 7.5},
			{"calling", 9.3, 9.8}, {"the", 9.8, 9.9}, {"witch's", 9.9, 10.4}, {"name.", 10.4, 15.9},
		}
		n := tightenSegmentsWithWords(segs, words)
		assert.Equal(t, 2, n)
		assert.InDelta(t, 6.1, segs[0].Start, 1e-9)
		assert.InDelta(t, 7.5, segs[0].End, 1e-9)
		assert.InDelta(t, 9.3, segs[1].Start, 1e-9)
		assert.InDelta(t, 15.0, segs[1].End, 1e-9, "a word running past the segment does not push the cue out")
	})

	t.Run("CR M1: the previous line's straggling last word does not pin this cue to the coarse start", func(t *testing.T) {
		segs := []whisperSegment{{Start: 0, End: 9, Text: "Baba Voss"}, {Start: 9, End: 15, Text: "calling the witch's name."}}
		words := []whisperWord{{"Baba", 6.8, 7.1}, {"Voss", 7.1, 9.4}, {"calling", 11.0, 11.5}, {"name.", 11.5, 15.0}}
		assert.Equal(t, 2, tightenSegmentsWithWords(segs, words))
		assert.InDelta(t, 11.0, segs[1].Start, 1e-9, "the cue opens when its own first word is spoken, not at the whole second")
		assert.InDelta(t, 9.0, segs[0].End, 1e-9)
	})

	t.Run("only a straggler inside: the engine's start stays", func(t *testing.T) {
		segs := []whisperSegment{{Start: 9, End: 15, Text: "..."}}
		words := []whisperWord{{"Voss", 7.1, 12.0}}
		assert.Equal(t, 1, tightenSegmentsWithWords(segs, words))
		assert.InDelta(t, 9.0, segs[0].Start, 1e-9)
		assert.InDelta(t, 12.0, segs[0].End, 1e-9)
	})

	t.Run("blank words are ignored", func(t *testing.T) {
		segs := []whisperSegment{{Start: 0, End: 5, Text: "Hi"}}
		words := []whisperWord{{" ", 0.1, 0.2}, {"Hi", 2.0, 3.0}, {"", 4.9, 5.0}}
		assert.Equal(t, 1, tightenSegmentsWithWords(segs, words))
		assert.InDelta(t, 2.0, segs[0].Start, 1e-9)
		assert.InDelta(t, 3.0, segs[0].End, 1e-9)
	})

	t.Run("no words → nothing changes", func(t *testing.T) {
		segs := []whisperSegment{{Start: 0, End: 4, Text: "Hi"}}
		assert.Zero(t, tightenSegmentsWithWords(segs, nil))
		assert.Equal(t, whisperSegment{Start: 0, End: 4, Text: "Hi"}, segs[0])
	})

	t.Run("a segment with no word inside keeps its timing", func(t *testing.T) {
		segs := []whisperSegment{{Start: 0, End: 4, Text: "Hi"}, {Start: 4, End: 8, Text: "There"}}
		words := []whisperWord{{"There", 5, 6}}
		assert.Equal(t, 1, tightenSegmentsWithWords(segs, words))
		assert.Equal(t, whisperSegment{Start: 0, End: 4, Text: "Hi"}, segs[0])
		assert.InDelta(t, 5, segs[1].Start, 1e-9)
	})

	t.Run("a cue that would become too short keeps the engine's timing", func(t *testing.T) {
		segs := []whisperSegment{{Start: 10, End: 12, Text: "No!"}}
		words := []whisperWord{{"No!", 10.9, 11.2}}
		assert.Zero(t, tightenSegmentsWithWords(segs, words))
		assert.InDelta(t, 10, segs[0].Start, 1e-9)
	})

	t.Run("the cue count and order never change", func(t *testing.T) {
		segs := []whisperSegment{{Start: 0, End: 3, Text: "A"}, {Start: 3, End: 6, Text: "B"}, {Start: 6, End: 9, Text: "C"}}
		words := []whisperWord{{"A", 1, 2.5}, {"B", 3.5, 5}, {"C", 7, 8.5}}
		tightenSegmentsWithWords(segs, words)
		assert.Equal(t, []string{"A", "B", "C"}, []string{segs[0].Text, segs[1].Text, segs[2].Text})
		for i := 1; i < len(segs); i++ {
			assert.LessOrEqual(t, segs[i-1].End, segs[i].Start)
		}
	})
}

func TestWhisperClient_RequestsWordTimestampsAndFallsBackOnce(t *testing.T) {
	var granularities [][]string
	var formats []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		g := r.MultipartForm.Value["timestamp_granularities[]"]
		granularities = append(granularities, g)
		formats = append(formats, r.FormValue("response_format"))
		if len(g) > 0 {
			// A self-hosted engine that does not know the field.
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":"unknown field timestamp_granularities"}`))
			return
		}
		w.Write([]byte(`{"language":"english","duration":2.0,"text":"hello","segments":[{"id":0,"start":0,"end":2,"text":"hello","avg_logprob":-0.2,"compression_ratio":1.0,"no_speech_prob":0.01}]}`))
	}))
	defer server.Close()

	audio := writeTempAudio(t)
	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL), WithWhisperWordTimestamps(true))

	detail, err := c.TranscribeDetailed(context.Background(), audio, "en")
	require.NoError(t, err)
	assert.True(t, detail.Filtered, "verbose_json still served after the word-timestamp retry")
	require.Len(t, granularities, 2, "one attempt with the field, one without")
	assert.ElementsMatch(t, []string{"word", "segment"}, granularities[0])
	assert.Empty(t, granularities[1])
	assert.Equal(t, []string{"verbose_json", "verbose_json"}, formats)

	// The engine is remembered: the next file does not pay the rejected call.
	granularities = nil
	_, err = c.TranscribeDetailed(context.Background(), audio, "en")
	require.NoError(t, err)
	require.Len(t, granularities, 1)
	assert.Empty(t, granularities[0])
}

func TestWhisperClient_WordTimestampsTightenTheSRT(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"language":"english","duration":9.0,"text":"I just heard Baba Voss",
			"segments":[{"id":0,"start":0.0,"end":9.0,"text":"I just heard Baba Voss","avg_logprob":-0.2,"compression_ratio":1.1,"no_speech_prob":0.01}],
			"words":[{"word":"I","start":6.1,"end":6.2},{"word":"Voss","start":7.1,"end":7.5}]}`))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL), WithWhisperWordTimestamps(true))
	detail, err := c.TranscribeDetailed(context.Background(), writeTempAudio(t), "en")
	require.NoError(t, err)
	assert.Contains(t, detail.SRT, "00:00:06,100 --> 00:00:07,500", "the cue follows the words, not the 0–9 s segment")
	assert.Equal(t, 1, detail.SegmentsKept)
}

func TestRejectsWordTimestamps(t *testing.T) {
	assert.True(t, rejectsWordTimestamps(errors.New(`status 400 — {"error":"unknown field timestamp_granularities"}`)))
	assert.True(t, rejectsWordTimestamps(errors.New(`status 422 — word timestamps not supported`)))
	assert.False(t, rejectsWordTimestamps(errors.New(`status 401 — {"error":"invalid password"}`)), "CR L2: a bare 'word' substring is not a rejection")
	assert.False(t, rejectsWordTimestamps(errors.New(`status 400 — unsupported response_format`)))
}

// A 5xx is a bad minute, not an engine without word timestamps: the latch
// must stay off so the next file asks again.
func TestWhisperClient_ServerErrorDoesNotLatchWordTimestamps(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"word timestamps backend down"}`))
	}))
	defer server.Close()
	client := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL), WithWhisperWordTimestamps(true))
	_, err := client.TranscribeDetailed(context.Background(), writeTempAudio(t), "en")
	require.Error(t, err)
	assert.False(t, client.wordTimestampsUnsupported.Load())
	assert.False(t, client.verboseUnsupported.Load())
}

// disc-2026-10-asr-word-timestamps-default-off: the default request carries no
// timestamp_granularities[] at all — byte-for-byte the pre-#698 body — so a
// deployment that never set VIDO_ASR_WORD_TIMESTAMPS keeps the behaviour that
// heard every line of See S01E02's opening.
func TestWhisperClient_WordTimestampsOffByDefault(t *testing.T) {
	var granularities [][]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(32<<20))
		granularities = append(granularities, r.MultipartForm.Value["timestamp_granularities[]"])
		w.Write([]byte(`{"language":"english","duration":2.0,"text":"hello","segments":[{"id":0,"start":0,"end":2,"text":"hello","avg_logprob":-0.2,"compression_ratio":1.0,"no_speech_prob":0.01}]}`))
	}))
	defer server.Close()

	c := NewWhisperClient("test-key", WithWhisperBaseURL(server.URL))
	_, err := c.TranscribeDetailed(context.Background(), writeTempAudio(t), "en")
	require.NoError(t, err)
	require.Len(t, granularities, 1)
	assert.Empty(t, granularities[0], "no field unless the operator opted in")
}
