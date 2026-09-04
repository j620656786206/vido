package services

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectEnglishTrack_PrefersEnglish(t *testing.T) {
	tracks := []AudioTrack{
		{Index: 1, Language: "jpn", Codec: "aac", Channels: 2},
		{Index: 2, Language: "eng", Codec: "ac3", Channels: 6},
		{Index: 3, Language: "chi", Codec: "aac", Channels: 2},
	}

	selected, err := SelectEnglishTrack(tracks)
	require.NoError(t, err)
	assert.Equal(t, 2, selected.Index)
	assert.Equal(t, "eng", selected.Language)
}

func TestSelectEnglishTrack_FallsBackToFirst(t *testing.T) {
	tracks := []AudioTrack{
		{Index: 1, Language: "jpn", Codec: "aac", Channels: 2},
		{Index: 3, Language: "chi", Codec: "aac", Channels: 2},
	}

	selected, err := SelectEnglishTrack(tracks)
	require.NoError(t, err)
	assert.Equal(t, 1, selected.Index)
	assert.Equal(t, "jpn", selected.Language)
}

func TestSelectEnglishTrack_AcceptsEnShortCode(t *testing.T) {
	tracks := []AudioTrack{
		{Index: 0, Language: "en", Codec: "aac", Channels: 2},
	}

	selected, err := SelectEnglishTrack(tracks)
	require.NoError(t, err)
	assert.Equal(t, 0, selected.Index)
	assert.Equal(t, "en", selected.Language)
}

func TestSelectEnglishTrack_EmptyTracks(t *testing.T) {
	_, err := SelectEnglishTrack(nil)
	assert.ErrorIs(t, err, ErrNoAudioTrack)

	_, err = SelectEnglishTrack([]AudioTrack{})
	assert.ErrorIs(t, err, ErrNoAudioTrack)
}

func TestParseAudioStreams(t *testing.T) {
	ffprobeJSON := ffprobeAudioOutput{
		Streams: []ffprobeAudioStream{
			{Index: 1, CodecName: "aac", Channels: 2, Tags: map[string]string{"language": "eng"}},
			{Index: 2, CodecName: "ac3", Channels: 6, Tags: map[string]string{"language": "jpn"}},
		},
	}
	data, err := json.Marshal(ffprobeJSON)
	require.NoError(t, err)

	tracks, err := parseAudioStreams(data)
	require.NoError(t, err)
	require.Len(t, tracks, 2)
	assert.Equal(t, "eng", tracks[0].Language)
	assert.Equal(t, "aac", tracks[0].Codec)
	assert.Equal(t, 2, tracks[0].Channels)
	assert.Equal(t, "jpn", tracks[1].Language)
}

func TestParseAudioStreams_NoLanguageTag(t *testing.T) {
	ffprobeJSON := ffprobeAudioOutput{
		Streams: []ffprobeAudioStream{
			{Index: 0, CodecName: "aac", Channels: 2, Tags: nil},
		},
	}
	data, err := json.Marshal(ffprobeJSON)
	require.NoError(t, err)

	tracks, err := parseAudioStreams(data)
	require.NoError(t, err)
	require.Len(t, tracks, 1)
	assert.Equal(t, "und", tracks[0].Language)
}

func TestParseAudioStreams_InvalidJSON(t *testing.T) {
	_, err := parseAudioStreams([]byte("not json"))
	assert.Error(t, err)
}

func TestNewAudioExtractorService_Defaults(t *testing.T) {
	svc := NewAudioExtractorService(0, 0, nil)
	// Should default to reasonable values without panicking
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.logger)
}

func TestAudioExtractorService_NotAvailable(t *testing.T) {
	svc := &AudioExtractorService{
		available: false,
	}
	assert.False(t, svc.IsAvailable())
}

func TestAudioExtractorService_ExtractAudio_NotAvailable(t *testing.T) {
	svc := &AudioExtractorService{available: false}
	_, err := svc.ExtractAudio(context.Background(), "/test.mkv", 0)
	assert.ErrorIs(t, err, ErrFFmpegNotAvailable)
}

func TestAudioExtractorService_ListAudioTracks_NotAvailable(t *testing.T) {
	svc := &AudioExtractorService{available: false}
	_, err := svc.ListAudioTracks(context.Background(), "/test.mkv")
	assert.ErrorIs(t, err, ErrFFmpegNotAvailable)
}

// ─── sub-6-3 CR M3: ASR audio extraction takes turns on the shared disk gate ─

// fakeExtractSlot records how the shared gate was used.
type fakeExtractSlot struct {
	acquired atomic.Int32
	released atomic.Int32
	err      error
}

func (f *fakeExtractSlot) Acquire(ctx context.Context) (func(), error) {
	if f.err != nil {
		return nil, f.err
	}
	f.acquired.Add(1)
	return func() { f.released.Add(1) }, nil
}

func TestAudioExtractorService_TakesTheSharedSlotAroundFFmpeg(t *testing.T) {
	slot := &fakeExtractSlot{}
	svc := NewAudioExtractorService(1, time.Second, nil, WithAudioExtractSlot(slot))
	if !svc.IsAvailable() {
		t.Skip("ffmpeg not installed — the extraction path cannot be driven here")
	}

	// A path that does not exist: ffmpeg fails fast, which is enough to prove
	// the slot was taken around the call and given back afterwards.
	_, err := svc.ExtractAudio(context.Background(), "/definitely/not/here.mkv", 0)
	require.Error(t, err)

	assert.EqualValues(t, 1, slot.acquired.Load(), "audio extraction is ffmpeg on the same disk — it must queue with subtitle extraction")
	assert.EqualValues(t, 1, slot.released.Load(), "the slot is always given back, failure included")
}

func TestAudioExtractorService_SlotIsOptional(t *testing.T) {
	svc := NewAudioExtractorService(1, time.Second, nil)
	assert.Nil(t, svc.slot, "no gate injected = the pre-sub-6-3 behaviour, serialized against itself only")

	assert.Nil(t, NewAudioExtractorService(1, time.Second, nil, WithAudioExtractSlot(nil)).slot,
		"a nil injection is ignored rather than stored")
}

func TestAudioExtractorService_GiveUpWaitingIsReported(t *testing.T) {
	slot := &fakeExtractSlot{err: context.Canceled}
	svc := NewAudioExtractorService(1, time.Second, nil, WithAudioExtractSlot(slot))
	if !svc.IsAvailable() {
		t.Skip("ffmpeg not installed")
	}

	_, err := svc.ExtractAudio(context.Background(), "/definitely/not/here.mkv", 0)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Contains(t, err.Error(), "waiting for the extraction slot")
}
