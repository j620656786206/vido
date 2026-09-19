package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── disc-2026-09-generation-resume-b-asr-chunk-store AC #2 ─────────────────
//
// A speech-recognition chunk that was paid for is remembered as TEXT (≈6 KB),
// never as audio (≈19 MB per chunk, 300 MB per film — a small NAS cannot keep
// that), keyed by everything that decides what the engine would say.

func baseIdentity() ASRChunkIdentity {
	return ASRChunkIdentity{
		MediaID:      uuidA,
		FileSize:     66_800_000_000,
		FileMTime:    1_760_000_000,
		TrackIndex:   1,
		Lang:         "en",
		Start:        600,
		ChunkSeconds: 600,
		Endpoint:     "|whisper-1",
	}
}

func TestASRChunkKey_EveryFieldParticipates(t *testing.T) {
	base := asrChunkKey(baseIdentity())
	assert.True(t, strings.HasPrefix(base, asrChunkKeyPrefix))
	assert.Equal(t, base, asrChunkKey(baseIdentity()), "same identity → same key")

	variants := map[string]func(*ASRChunkIdentity){
		"media":        func(i *ASRChunkIdentity) { i.MediaID = uuidB },
		"size":         func(i *ASRChunkIdentity) { i.FileSize++ },
		"mtime":        func(i *ASRChunkIdentity) { i.FileMTime++ },
		"track":        func(i *ASRChunkIdentity) { i.TrackIndex = 2 },
		"lang":         func(i *ASRChunkIdentity) { i.Lang = "ja" },
		"start":        func(i *ASRChunkIdentity) { i.Start = 1200 },
		"chunkSeconds": func(i *ASRChunkIdentity) { i.ChunkSeconds = 300 },
		"endpoint":     func(i *ASRChunkIdentity) { i.Endpoint = "https://groq.example/v1|whisper-large-v3" },
	}
	seen := map[string]string{base: "base"}
	for name, mutate := range variants {
		id := baseIdentity()
		mutate(&id)
		key := asrChunkKey(id)
		prev, dup := seen[key]
		assert.Falsef(t, dup, "%s collides with %s", name, prev)
		seen[key] = name
	}
	// A key is a hash: none of the raw fields leak into it (the media id would
	// otherwise make the cache table a list of what the user transcribed).
	for _, raw := range []string{uuidA, "66800000000", "whisper-1", "en"} {
		assert.NotContains(t, base, raw)
	}
}

func TestASRManifestKey_IsPerFileAndEndpoint(t *testing.T) {
	id := baseIdentity()
	k := asrManifestKey(id.MediaID, id.FileSize, id.FileMTime, id.Endpoint)
	assert.True(t, strings.HasPrefix(k, asrManifestKeyPrefix))
	assert.Equal(t, k, asrManifestKey(id.MediaID, id.FileSize, id.FileMTime, id.Endpoint))
	assert.NotEqual(t, k, asrManifestKey(id.MediaID, id.FileSize+1, id.FileMTime, id.Endpoint))
	assert.NotEqual(t, k, asrManifestKey(id.MediaID, id.FileSize, id.FileMTime, "other|model"))
	// The manifest does not depend on start/track/lang: it DESCRIBES them.
	assert.NotContains(t, k, uuidA)
}

func TestASRChunkStore_TagsTheFamilyAndMapsValues(t *testing.T) {
	repo := &fakeCacheRepo{}
	store := NewASRChunkStore(repo)

	require.NoError(t, store.Set(context.Background(), "k1", `{"filtered":"a","unfiltered":"b"}`, asrChunkTTL))
	assert.Equal(t, asrChunkType, repo.lastType,
		"a wrong type tag would orphan the family from ClearByType eviction and audits")
	assert.Equal(t, asrChunkTTL, repo.lastTTL)

	values, err := store.GetMany(context.Background(), []string{"k1", "k-missing"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"k1": `{"filtered":"a","unfiltered":"b"}`}, values,
		"a miss is expressed by absence, and a hit returns the stored value verbatim")
	assert.NoError(t, store.Delete(context.Background(), "k1"))
}

func TestASRChunkValue_RoundTrip(t *testing.T) {
	raw, err := json.Marshal(asrChunkValue{Filtered: "1\n00:00:01,000 --> 00:00:02,000\nHi\n", Unfiltered: "u"})
	require.NoError(t, err)
	got, ok := decodeASRChunkValue(string(raw))
	require.True(t, ok)
	assert.Equal(t, "u", got.Unfiltered)

	_, ok = decodeASRChunkValue("not json")
	assert.False(t, ok, "a corrupt entry is a miss, never an error")
	_, ok = decodeASRChunkValue("")
	assert.False(t, ok)
}

func TestASRManifest_RoundTrip(t *testing.T) {
	m := asrManifest{Track: 1, Lang: "en", ChunkSeconds: 600, Done: []int{0, 600, 1200}}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	got, ok := decodeASRManifest(string(raw))
	require.True(t, ok)
	assert.Equal(t, m, got)
	_, ok = decodeASRManifest("{")
	assert.False(t, ok)
}
